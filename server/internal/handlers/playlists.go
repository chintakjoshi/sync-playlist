package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"server/internal/auth"
	"server/internal/database"
	"server/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var tokenManager = auth.NewTokenManager(database.DB)

// GetPlaylists fetches playlists from a specific service for the authenticated user
func GetPlaylists(c *gin.Context) {
	serviceType := c.Param("service")
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get the user's service connection
	var userService database.UserService
	result := database.DB.Where("user_id = ? AND service_type = ?", user.ID, serviceType).First(&userService)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not connected"})
		return
	}

	// Refresh token if needed
	if err := tokenManager.RefreshTokenIfNeeded(&userService); err != nil {
		log.Printf("Token refresh failed for %s: %v", serviceType, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token refresh failed: " + err.Error()})
		return
	}

	// Fetch playlists from the service
	playlists, err := fetchPlaylistsFromService(serviceType, userService.AccessToken)
	if err != nil {
		log.Printf("Failed to fetch playlists from %s: %v", serviceType, err)

		// If API call fails, try to validate token
		if valid, _ := tokenManager.ValidateToken(&userService); !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Service connection expired. Please reconnect."})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlists: " + err.Error()})
		return
	}

	// Store playlists in database (async)
	go storePlaylistsInDatabase(user.ID, serviceType, playlists)

	c.JSON(http.StatusOK, gin.H{
		"service":   serviceType,
		"playlists": playlists,
	})
}

// SyncAllPlaylists triggers sync for all connected services
func SyncAllPlaylists(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get user's connected services
	var services []database.UserService
	result := database.DB.Where("user_id = ?", user.ID).Find(&services)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	// Sync each service
	for _, service := range services {
		go syncServicePlaylists(user.ID, service)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Sync started for all services",
		"services": len(services),
	})
}

// GetStoredPlaylists returns playlists from database (faster than API calls)
func GetStoredPlaylists(c *gin.Context) {
	serviceType := c.Param("service")
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var playlists []database.Playlist
	result := database.DB.Where("user_id = ? AND service_type = ?", user.ID, serviceType).Find(&playlists)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service":   serviceType,
		"playlists": playlists,
	})
}

// fetchPlaylistsFromService calls the appropriate service API
func fetchPlaylistsFromService(serviceType string, accessToken string) ([]PlaylistResponse, error) {
	switch serviceType {
	case "spotify":
		return fetchSpotifyPlaylists(accessToken)
	case "youtube":
		return fetchYouTubePlaylists(accessToken)
	default:
		return nil, fmt.Errorf("unsupported service: %s", serviceType)
	}
}

// PlaylistResponse represents a standardized playlist response
type PlaylistResponse struct {
	ServiceID   string `json:"service_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TrackCount  int    `json:"track_count"`
	ImageURL    string `json:"image_url"`
	IsPublic    bool   `json:"is_public"`
}

// Spotify API integration
func fetchSpotifyPlaylists(accessToken string) ([]PlaylistResponse, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me/playlists?limit=50", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API returned status: %d", resp.StatusCode)
	}

	var spotifyResponse struct {
		Items []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Tracks      struct {
				Total int `json:"total"`
			} `json:"tracks"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
			Public bool `json:"public"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		return nil, err
	}

	var playlists []PlaylistResponse
	for _, item := range spotifyResponse.Items {
		imageURL := ""
		if len(item.Images) > 0 {
			imageURL = item.Images[0].URL
		}

		playlists = append(playlists, PlaylistResponse{
			ServiceID:   item.ID,
			Name:        item.Name,
			Description: item.Description,
			TrackCount:  item.Tracks.Total,
			ImageURL:    imageURL,
			IsPublic:    item.Public,
		})
	}

	return playlists, nil
}

// YouTube API integration
func fetchYouTubePlaylists(accessToken string) ([]PlaylistResponse, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://www.googleapis.com/youtube/v3/playlists?part=snippet,contentDetails&mine=true&maxResults=50", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube API returned status: %d", resp.StatusCode)
	}

	var youtubeResponse struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Thumbnails  struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				ItemCount int `json:"itemCount"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&youtubeResponse); err != nil {
		return nil, err
	}

	var playlists []PlaylistResponse
	for _, item := range youtubeResponse.Items {
		playlists = append(playlists, PlaylistResponse{
			ServiceID:   item.ID,
			Name:        item.Snippet.Title,
			Description: item.Snippet.Description,
			TrackCount:  item.ContentDetails.ItemCount,
			ImageURL:    item.Snippet.Thumbnails.Default.URL,
			IsPublic:    true, // YouTube doesn't expose this easily in this endpoint
		})
	}

	return playlists, nil
}

// storePlaylistsInDatabase saves playlists to the database
func storePlaylistsInDatabase(userID uint, serviceType string, playlists []PlaylistResponse) {
	for _, playlist := range playlists {
		var existingPlaylist database.Playlist
		result := database.DB.Where("user_id = ? AND service_type = ? AND service_id = ?", userID, serviceType, playlist.ServiceID).First(&existingPlaylist)

		dbPlaylist := database.Playlist{
			UserID:       userID,
			ServiceType:  serviceType,
			ServiceID:    playlist.ServiceID,
			Name:         playlist.Name,
			Description:  playlist.Description,
			TrackCount:   playlist.TrackCount,
			ImageURL:     playlist.ImageURL,
			IsPublic:     playlist.IsPublic,
			LastSyncedAt: time.Now().Unix(),
		}

		if result.Error == gorm.ErrRecordNotFound {
			// Create new playlist
			database.DB.Create(&dbPlaylist)
		} else if result.Error == nil {
			// Update existing playlist
			existingPlaylist.Name = dbPlaylist.Name
			existingPlaylist.Description = dbPlaylist.Description
			existingPlaylist.TrackCount = dbPlaylist.TrackCount
			existingPlaylist.ImageURL = dbPlaylist.ImageURL
			existingPlaylist.IsPublic = dbPlaylist.IsPublic
			existingPlaylist.LastSyncedAt = dbPlaylist.LastSyncedAt
			database.DB.Save(&existingPlaylist)
		}
	}
	log.Printf("Stored %d %s playlists for user %d", len(playlists), serviceType, userID)
}

// syncServicePlaylists syncs playlists for a specific service
func syncServicePlaylists(userID uint, service database.UserService) {
	playlists, err := fetchPlaylistsFromService(service.ServiceType, service.AccessToken)
	if err != nil {
		log.Printf("Failed to sync %s playlists for user %d: %v", service.ServiceType, userID, err)
		return
	}

	storePlaylistsInDatabase(userID, service.ServiceType, playlists)
}
