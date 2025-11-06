package handlers

import (
	"net/http"

	"server/internal/database"
	"server/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestTrackSearch(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	service := c.Query("service")
	trackName := c.Query("track")
	artist := c.Query("artist")

	if service == "" || trackName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service and track name are required"})
		return
	}

	// Get the service connection
	var userService database.UserService
	result := database.DB.Where("user_id = ? AND service_type = ?", user.ID, service).First(&userService)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not connected"})
		return
	}

	track := Track{
		Name:   trackName,
		Artist: artist,
	}

	foundTrack, confidence, err := searchTrack(service, userService.AccessToken, track)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"track":      foundTrack,
		"confidence": confidence,
	})
}

func TestPlaylistFetch(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	service := c.Query("service")
	playlistID := c.Query("playlist_id")

	if service == "" || playlistID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service and playlist ID are required"})
		return
	}

	// Get the service connection
	var userService database.UserService
	result := database.DB.Where("user_id = ? AND service_type = ?", user.ID, service).First(&userService)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not connected"})
		return
	}

	tracks, playlistName, err := fetchPlaylistTracks(service, userService.AccessToken, playlistID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"playlist_name": playlistName,
		"tracks_count":  len(tracks),
		"tracks":        tracks,
	})
}
