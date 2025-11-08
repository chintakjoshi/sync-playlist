package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"server/internal/database"
	"server/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TransferRequest struct {
	SourceService      string `json:"source_service" binding:"required"`
	SourcePlaylistID   string `json:"source_playlist_id" binding:"required"`
	TargetService      string `json:"target_service" binding:"required"`
	TargetPlaylistName string `json:"target_playlist_name"`
}

type Track struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
	ISRC     string `json:"isrc"`
}

// In StartTransfer function, make sure we save the transfer before starting the goroutine
func StartTransfer(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate services are connected
	var sourceService, targetService database.UserService
	if err := database.DB.Where("user_id = ? AND service_type = ?", user.ID, req.SourceService).First(&sourceService).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source service not connected"})
		return
	}
	if err := database.DB.Where("user_id = ? AND service_type = ?", user.ID, req.TargetService).First(&targetService).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target service not connected"})
		return
	}

	// Create and save transfer record first
	transfer := database.Transfer{
		UserID:           user.ID,
		SourceService:    req.SourceService,
		SourcePlaylistID: req.SourcePlaylistID,
		TargetService:    req.TargetService,
		Status:           "pending",
	}

	// Save the transfer to get an ID
	if err := database.DB.Create(&transfer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer record"})
		return
	}

	log.Printf("Created transfer record with ID: %d", transfer.ID)

	// Start transfer in background
	go processTransfer(transfer, sourceService, targetService, req.TargetPlaylistName)

	c.JSON(http.StatusOK, gin.H{
		"message":     "Transfer started",
		"transfer_id": transfer.ID,
	})
}

// GetTransfers returns transfer history for the user
func GetTransfers(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var transfers []database.Transfer
	result := database.DB.Where("user_id = ?", user.ID).Order("created_at DESC").Limit(50).Find(&transfers)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transfers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transfers": transfers})
}

// GetTransferDetails returns detailed information about a transfer
func GetTransferDetails(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	transferID := c.Param("id")
	log.Printf("Fetching transfer details for ID: %s, User: %d", transferID, user.ID)

	// Convert transferID string to uint
	id, err := strconv.ParseUint(transferID, 10, 32)
	if err != nil {
		log.Printf("Invalid transfer ID: %s, error: %v", transferID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer ID"})
		return
	}

	var transfer database.Transfer
	if err := database.DB.Where("id = ? AND user_id = ?", uint(id), user.ID).First(&transfer).Error; err != nil {
		log.Printf("Transfer not found: ID=%d, UserID=%d, Error=%v", uint(id), user.ID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	var transferTracks []database.TransferTrack
	if err := database.DB.Where("transfer_id = ?", transfer.ID).Find(&transferTracks).Error; err != nil {
		log.Printf("Error fetching transfer tracks: %v", err)
		// Continue without tracks
	}

	log.Printf("Found transfer: %+v", transfer)
	log.Printf("Found %d transfer tracks", len(transferTracks))

	c.JSON(http.StatusOK, gin.H{
		"transfer": transfer,
		"tracks":   transferTracks,
	})
}

// Update the processTransfer function to call debug at the beginning:
func processTransfer(transfer database.Transfer, sourceService, targetService database.UserService, targetPlaylistName string) {
	db := database.DB.Session(&gorm.Session{NewDB: true})

	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in transfer %d: %v", transfer.ID, r)
			db.Model(&transfer).Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": fmt.Sprintf("Panic: %v", r),
			})
		}
	}()

	log.Printf("=== STARTING TRANSFER %d ===", transfer.ID)
	log.Printf("Source: %s, Playlist: %s", transfer.SourceService, transfer.SourcePlaylistID)
	log.Printf("Target: %s", transfer.TargetService)
	log.Printf("Source Token: %s...", sourceService.AccessToken[:20])
	log.Printf("Target Token: %s...", targetService.AccessToken[:20])

	// Update transfer status using the new session
	db.Model(&transfer).Update("status", "processing")

	// Fetch source playlist tracks
	log.Printf("Fetching source playlist tracks...")
	sourceTracks, sourcePlaylistName, err := fetchPlaylistTracks(transfer.SourceService, sourceService.AccessToken, transfer.SourcePlaylistID)
	if err != nil {
		log.Printf("Failed to fetch source playlist: %v", err)
		db.Model(&transfer).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": "Failed to fetch source playlist: " + err.Error(),
		})
		return
	}

	log.Printf("Fetched %d tracks from source playlist: %s", len(sourceTracks), sourcePlaylistName)

	if len(sourceTracks) == 0 {
		log.Printf("Source playlist is empty")
		db.Model(&transfer).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": "Source playlist is empty",
		})
		return
	}

	// Update source playlist name
	transfer.SourcePlaylistName = sourcePlaylistName
	db.Save(&transfer)

	// Set target playlist name if not provided
	if targetPlaylistName == "" {
		targetPlaylistName = sourcePlaylistName
	}

	// Create target playlist
	log.Printf("Creating target playlist: %s", targetPlaylistName)
	targetPlaylistID, err := createPlaylist(targetService.ServiceType, targetService.AccessToken, targetPlaylistName, "Transferred from "+transfer.SourceService)
	if err != nil {
		log.Printf("Failed to create target playlist: %v", err)
		db.Model(&transfer).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": "Failed to create target playlist: " + err.Error(),
		})
		return
	}

	log.Printf("Created target playlist: %s", targetPlaylistID)

	transfer.TargetPlaylistID = targetPlaylistID
	transfer.TargetPlaylistName = targetPlaylistName
	transfer.TracksTotal = len(sourceTracks)
	db.Save(&transfer)

	// Match and add tracks
	matchedTracks := 0
	failedTracks := 0

	for i, track := range sourceTracks {
		log.Printf("Processing track %d/%d: %s - %s", i+1, len(sourceTracks), track.Artist, track.Name)

		trackResult := database.TransferTrack{
			TransferID:      transfer.ID,
			SourceTrackID:   track.ID,
			SourceTrackName: track.Name,
			SourceArtist:    track.Artist,
			Status:          "not_found",
			MatchConfidence: 0.0,
		}

		// Search for track on target service
		targetTrack, confidence, err := searchTrack(targetService.ServiceType, targetService.AccessToken, track)
		if err != nil {
			log.Printf("Track search failed: %v", err)
			trackResult.Status = "not_found"
			failedTracks++
		} else if targetTrack.ID != "" {
			log.Printf("Found track match: %s - %s (confidence: %.2f)", targetTrack.Artist, targetTrack.Name, confidence)

			// Add track to target playlist
			err = addTrackToPlaylist(targetService.ServiceType, targetService.AccessToken, targetPlaylistID, targetTrack.ID)
			if err != nil {
				log.Printf("Failed to add track to playlist: %v", err)
				trackResult.Status = "error"
				trackResult.TargetTrackID = targetTrack.ID
				trackResult.TargetTrackName = targetTrack.Name
				trackResult.TargetArtist = targetTrack.Artist
				trackResult.MatchConfidence = confidence
				failedTracks++
			} else {
				log.Printf("Successfully added track to playlist")
				trackResult.TargetTrackID = targetTrack.ID
				trackResult.TargetTrackName = targetTrack.Name
				trackResult.TargetArtist = targetTrack.Artist
				trackResult.Status = "matched"
				trackResult.MatchConfidence = confidence
				matchedTracks++
			}
		} else {
			log.Printf("No match found for track: %s - %s", track.Artist, track.Name)
			failedTracks++
		}

		// Persist the result immediately
		if err := db.Create(&trackResult).Error; err != nil {
			log.Printf("Failed to save track result: %v", err)
		}
	}

	// Update transfer with results
	transfer.TracksMatched = matchedTracks
	transfer.TracksFailed = failedTracks
	status := "failed"
	if matchedTracks > 0 {
		if failedTracks == 0 {
			status = "completed"
		} else {
			status = "completed_with_errors"
		}
	}
	transfer.Status = status

	if err := db.Save(&transfer).Error; err != nil {
		log.Printf("Failed to update transfer status: %v", err)
	}

	log.Printf("Transfer %d completed: %d/%d tracks transferred, %d failed, status: %s",
		transfer.ID, matchedTracks, transfer.TracksTotal, failedTracks, status)
}

// fetchPlaylistTracks gets tracks from a playlist
func fetchPlaylistTracks(serviceType, accessToken, playlistID string) ([]Track, string, error) {
	switch serviceType {
	case "spotify":
		return fetchSpotifyPlaylistTracks(accessToken, playlistID)
	case "youtube":
		return fetchYouTubePlaylistTracks(accessToken, playlistID)
	default:
		return nil, "", fmt.Errorf("unsupported service: %s", serviceType)
	}
}

// fetchSpotifyPlaylistTracks gets tracks from a Spotify playlist
func fetchSpotifyPlaylistTracks(accessToken, playlistID string) ([]Track, string, error) {
	client := &http.Client{}

	// Simple request without fields filter
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Spotify playlist API error: %d, body: %s", resp.StatusCode, string(body))
		return nil, "", fmt.Errorf("spotify API returned status: %d", resp.StatusCode)
	}

	var spotifyResponse struct {
		Name   string `json:"name"`
		Tracks struct {
			Items []struct {
				Track struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
					Album struct {
						Name string `json:"name"`
					} `json:"album"`
				} `json:"track"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		return nil, "", err
	}

	log.Printf("Spotify playlist '%s' has %d tracks", spotifyResponse.Name, len(spotifyResponse.Tracks.Items))

	var tracks []Track
	for _, item := range spotifyResponse.Tracks.Items {
		artist := ""
		if len(item.Track.Artists) > 0 {
			artist = item.Track.Artists[0].Name
		}

		tracks = append(tracks, Track{
			ID:     item.Track.ID,
			Name:   item.Track.Name,
			Artist: artist,
			Album:  item.Track.Album.Name,
		})
	}

	return tracks, spotifyResponse.Name, nil
}

// fetchYouTubePlaylistTracks gets tracks from a YouTube playlist
func fetchYouTubePlaylistTracks(accessToken, playlistID string) ([]Track, string, error) {
	client := &http.Client{}
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet,contentDetails&playlistId=%s&maxResults=50", playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("YouTube playlist items API error: %d, body: %s", resp.StatusCode, string(body))
		return nil, "", fmt.Errorf("youtube API returned status: %d", resp.StatusCode)
	}

	var youtubeResponse struct {
		Items []struct {
			Snippet struct {
				Title      string `json:"title"`
				ResourceID struct {
					VideoID string `json:"videoId"`
				} `json:"resourceId"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&youtubeResponse); err != nil {
		return nil, "", err
	}

	// For YouTube, we need to get the playlist name separately
	playlistName, err := getYouTubePlaylistName(accessToken, playlistID)
	if err != nil {
		playlistName = "YouTube Playlist"
	}

	var tracks []Track
	for _, item := range youtubeResponse.Items {
		// Parse title to extract artist and track name
		title := item.Snippet.Title
		artist, trackName := parseYouTubeTitle(title)

		log.Printf("YouTube track - Original: '%s', Parsed: Artist='%s', Track='%s'", title, artist, trackName)

		tracks = append(tracks, Track{
			ID:     item.Snippet.ResourceID.VideoID,
			Name:   trackName,
			Artist: artist,
		})
	}

	return tracks, playlistName, nil
}

// getYouTubePlaylistName gets the name of a YouTube playlist
func getYouTubePlaylistName(accessToken, playlistID string) (string, error) {
	client := &http.Client{}
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlists?part=snippet&id=%s", playlistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("youtube API returned status: %d", resp.StatusCode)
	}

	var response struct {
		Items []struct {
			Snippet struct {
				Title string `json:"title"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("playlist not found")
	}

	return response.Items[0].Snippet.Title, nil
}

// parseYouTubeTitle attempts to parse artist and track name from YouTube video title
// parseYouTubeTitle attempts to parse artist and track name from YouTube video title
func parseYouTubeTitle(title string) (string, string) {
	title = strings.TrimSpace(title)

	// Remove common YouTube suffixes
	suffixes := []string{
		"(Official Video)", "(Official Audio)", "[Official Video]", "[Official Audio]",
		"(Official Music Video)", "[Official Music Video]", "(Lyric Video)", "[Lyric Video]",
		"(Visualizer)", "[Visualizer]", "(Lyrics)", "[Lyrics]", "(Live)", "[Live]",
		"(Acoustic)", "[Acoustic]", "(Remix)", "[Remix]", "(Cover)", "[Cover]",
		"| Official Video", "| Official Audio", "| Official Music Video",
	}

	for _, suffix := range suffixes {
		title = strings.Replace(title, suffix, "", -1)
	}

	title = strings.TrimSpace(title)

	// Try different patterns
	patterns := []string{
		`^(.*?)\s*[-–—]\s*(.*)$`, // "Artist - Track"
		`^(.*?)\s*:\s*(.*)$`,     // "Artist: Track"
		`^(.*?)\s*\|\s*(.*)$`,    // "Artist | Track"
		`^(.*?)\s*-\s*(.*)$`,     // "Artist - Track" (regular dash)
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(title)
		if len(matches) == 3 {
			artist := strings.TrimSpace(matches[1])
			track := strings.TrimSpace(matches[2])
			if artist != "" && track != "" {
				return artist, track
			}
		}
	}

	// If no pattern matches, return the whole title as track name
	return "", title
}

// searchTrack searches for a track on the target service
func searchTrack(serviceType, accessToken string, track Track) (Track, float64, error) {
	switch serviceType {
	case "spotify":
		return searchSpotifyTrack(accessToken, track)
	case "youtube":
		return searchYouTubeTrack(accessToken, track)
	default:
		return Track{}, 0.0, fmt.Errorf("unsupported service: %s", serviceType)
	}
}

// searchSpotifyTrack searches for a track on Spotify
func searchSpotifyTrack(accessToken string, track Track) (Track, float64, error) {
	client := &http.Client{}

	// Build search query - handle empty artist
	var query string
	if track.Artist != "" {
		query = fmt.Sprintf("track:%s artist:%s", track.Name, track.Artist)
	} else {
		query = fmt.Sprintf("track:%s", track.Name)
	}

	encodedQuery := url.QueryEscape(query)

	log.Printf("Searching Spotify for: %s", query)

	req, err := http.NewRequest("GET",
		fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=5", encodedQuery),
		nil)
	if err != nil {
		return Track{}, 0.0, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return Track{}, 0.0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Spotify search API error: %d, body: %s", resp.StatusCode, string(body))
		return Track{}, 0.0, fmt.Errorf("spotify API returned status: %d", resp.StatusCode)
	}

	var searchResponse struct {
		Tracks struct {
			Items []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return Track{}, 0.0, err
	}

	if len(searchResponse.Tracks.Items) == 0 {
		return Track{}, 0.0, fmt.Errorf("track not found")
	}

	// Return the first result for now
	bestMatch := searchResponse.Tracks.Items[0]
	artist := ""
	if len(bestMatch.Artists) > 0 {
		artist = bestMatch.Artists[0].Name
	}

	confidence := calculateMatchConfidence(track.Name, track.Artist, bestMatch.Name, artist)

	log.Printf("Found track: %s - %s (confidence: %.2f)", artist, bestMatch.Name, confidence)

	return Track{
		ID:     bestMatch.ID,
		Name:   bestMatch.Name,
		Artist: artist,
	}, confidence, nil
}

// searchYouTubeTrack searches for a track on YouTube
func searchYouTubeTrack(accessToken string, track Track) (Track, float64, error) {
	client := &http.Client{}

	// Build better search query for music
	query := fmt.Sprintf("%s %s official audio", track.Name, track.Artist)
	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&type=video&maxResults=5&videoCategoryId=10", encodedQuery) // category 10 is music

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Track{}, 0.0, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return Track{}, 0.0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("YouTube search API error: %d, body: %s", resp.StatusCode, string(body))
		return Track{}, 0.0, fmt.Errorf("youtube API returned status: %d", resp.StatusCode)
	}

	var searchResponse struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return Track{}, 0.0, err
	}

	if len(searchResponse.Items) == 0 {
		return Track{}, 0.0, fmt.Errorf("no results found")
	}

	// Find the best match
	bestMatch := searchResponse.Items[0]
	bestConfidence := 0.0

	for _, item := range searchResponse.Items {
		confidence := calculateYouTubeMatchConfidence(track, item.Snippet.Title, item.Snippet.Description)
		if confidence > bestConfidence {
			bestMatch = item
			bestConfidence = confidence
		}
	}

	artist, trackName := parseYouTubeTitle(bestMatch.Snippet.Title)

	return Track{
		ID:     bestMatch.ID.VideoID,
		Name:   trackName,
		Artist: artist,
	}, bestConfidence, nil
}

// Add a YouTube-specific confidence calculator
func calculateYouTubeMatchConfidence(track Track, title, description string) float64 {
	confidence := 0.0
	titleLower := strings.ToLower(title)
	descLower := strings.ToLower(description)
	trackNameLower := strings.ToLower(track.Name)
	artistLower := strings.ToLower(track.Artist)

	// Check for track name in title
	if strings.Contains(titleLower, trackNameLower) {
		confidence += 0.4
	}

	// Check for artist in title
	if strings.Contains(titleLower, artistLower) {
		confidence += 0.3
	}

	// Check for "official" in title (indicates official music video/audio)
	if strings.Contains(titleLower, "official") {
		confidence += 0.2
	}

	// Check for music-related terms
	if strings.Contains(titleLower, "audio") || strings.Contains(descLower, "music") {
		confidence += 0.1
	}

	return confidence
}

// calculateMatchConfidence calculates how well two tracks match
func calculateMatchConfidence(sourceName, sourceArtist, targetName, targetArtist string) float64 {
	confidence := 0.0

	// Normalize strings for comparison
	sourceNameNorm := strings.ToLower(strings.TrimSpace(sourceName))
	targetNameNorm := strings.ToLower(strings.TrimSpace(targetName))
	sourceArtistNorm := strings.ToLower(strings.TrimSpace(sourceArtist))
	targetArtistNorm := strings.ToLower(strings.TrimSpace(targetArtist))

	// Name matching
	if sourceNameNorm == targetNameNorm {
		confidence += 0.6
	} else if strings.Contains(sourceNameNorm, targetNameNorm) || strings.Contains(targetNameNorm, sourceNameNorm) {
		confidence += 0.4
	} else {
		// Try to remove common suffixes
		sourceClean := removeCommonSuffixes(sourceNameNorm)
		targetClean := removeCommonSuffixes(targetNameNorm)
		if sourceClean == targetClean {
			confidence += 0.5
		}
	}

	// Artist matching
	if sourceArtistNorm == targetArtistNorm {
		confidence += 0.4
	} else if strings.Contains(sourceArtistNorm, targetArtistNorm) || strings.Contains(targetArtistNorm, sourceArtistNorm) {
		confidence += 0.2
	}

	return confidence
}

// removeCommonSuffixes removes common track name suffixes
func removeCommonSuffixes(name string) string {
	suffixes := []string{" - remaster", " (remaster", " - live", " (live", " - acoustic", " (acoustic"}
	result := name
	for _, suffix := range suffixes {
		if idx := strings.Index(result, suffix); idx != -1 {
			result = result[:idx]
		}
	}
	return strings.TrimSpace(result)
}

// createPlaylist creates a new playlist on the target service
func createPlaylist(serviceType, accessToken, name, description string) (string, error) {
	switch serviceType {
	case "spotify":
		return createSpotifyPlaylist(accessToken, name, description)
	case "youtube":
		return createYouTubePlaylist(accessToken, name, description)
	default:
		return "", fmt.Errorf("unsupported service: %s", serviceType)
	}
}

// createSpotifyPlaylist creates a Spotify playlist
func createSpotifyPlaylist(accessToken, name, description string) (string, error) {
	// First, get the user's ID to create the playlist for them
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user info: %d", resp.StatusCode)
	}

	var userInfo struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", err
	}

	// Create the playlist
	createData := map[string]interface{}{
		"name":        name,
		"description": description,
		"public":      false,
	}
	createBody, _ := json.Marshal(createData)

	req, err = http.NewRequest("POST", fmt.Sprintf("https://api.spotify.com/v1/users/%s/playlists", userInfo.ID), strings.NewReader(string(createBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Spotify playlist creation error: %d, body: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("failed to create playlist: %d", resp.StatusCode)
	}

	var playlistResponse struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&playlistResponse); err != nil {
		return "", err
	}

	return playlistResponse.ID, nil
}

// createYouTubePlaylist creates a YouTube playlist
func createYouTubePlaylist(accessToken, name, description string) (string, error) {
	client := &http.Client{}

	createData := map[string]interface{}{
		"snippet": map[string]string{
			"title":       name,
			"description": description,
		},
		"status": map[string]string{
			"privacyStatus": "private",
		},
	}
	createBody, _ := json.Marshal(createData)

	req, err := http.NewRequest("POST", "https://www.googleapis.com/youtube/v3/playlists?part=snippet,status", strings.NewReader(string(createBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("YouTube playlist creation error: %d, body: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("failed to create playlist: %d", resp.StatusCode)
	}

	var playlistResponse struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&playlistResponse); err != nil {
		return "", err
	}

	return playlistResponse.ID, nil
}

// addTrackToPlaylist adds a track to a playlist
func addTrackToPlaylist(serviceType, accessToken, playlistID, trackID string) error {
	switch serviceType {
	case "spotify":
		return addTrackToSpotifyPlaylist(accessToken, playlistID, trackID)
	case "youtube":
		return addTrackToYouTubePlaylist(accessToken, playlistID, trackID)
	default:
		return fmt.Errorf("unsupported service: %s", serviceType)
	}
}

// addTrackToSpotifyPlaylist adds a track to a Spotify playlist
func addTrackToSpotifyPlaylist(accessToken, playlistID, trackID string) error {
	client := &http.Client{}

	addData := map[string]interface{}{
		"uris": []string{fmt.Sprintf("spotify:track:%s", trackID)},
	}
	addBody, _ := json.Marshal(addData)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID), strings.NewReader(string(addBody)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Spotify add track error: %d, body: %s", resp.StatusCode, string(body))
		return fmt.Errorf("failed to add track: %d", resp.StatusCode)
	}

	return nil
}

// addTrackToYouTubePlaylist adds a track to a YouTube playlist
func addTrackToYouTubePlaylist(accessToken, playlistID, trackID string) error {
	client := &http.Client{}

	addData := map[string]interface{}{
		"snippet": map[string]interface{}{
			"playlistId": playlistID,
			"resourceId": map[string]string{
				"kind":    "youtube#video",
				"videoId": trackID,
			},
		},
	}
	addBody, _ := json.Marshal(addData)

	req, err := http.NewRequest("POST", "https://www.googleapis.com/youtube/v3/playlistItems?part=snippet", strings.NewReader(string(addBody)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("YouTube add track error: %d, body: %s", resp.StatusCode, string(body))
		return fmt.Errorf("failed to add track: %d", resp.StatusCode)
	}

	return nil
}
