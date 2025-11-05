package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"server/internal/auth"
	"server/internal/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func HandleConnectService(c *gin.Context) {
	provider := c.Param("provider")

	config := auth.GetOAuthConfig(provider)
	if config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported service provider"})
		return
	}

	// For now, we'll use a placeholder user ID. In a real implementation,
	// you'd get this from JWT middleware
	state := "temp-user-id" // This should be a proper state token with user ID

	url := config.AuthCodeURL(state)
	log.Printf("Redirecting to %s OAuth: %s", provider, url)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func HandleServiceCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	c.Query("state")
	error := c.Query("error")

	if error != "" {
		errorDescription := c.Query("error_description")
		log.Printf("OAuth error from %s: %s - %s", provider, error, errorDescription)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             error,
			"error_description": errorDescription,
		})
		return
	}

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not provided"})
		return
	}

	config := auth.GetOAuthConfig(provider)
	if config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported service provider"})
		return
	}

	log.Printf("Exchanging code for %s token", provider)

	// Exchange code for token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Token exchange error for %s: %v", provider, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token: " + err.Error()})
		return
	}

	log.Printf("Successfully obtained %s token", provider)

	var serviceUserID, serviceUserName string

	// Get user info from the service
	if provider == "spotify" {
		client := config.Client(context.Background(), token)
		resp, err := client.Get("https://api.spotify.com/v1/me")
		if err != nil {
			log.Printf("Failed to get Spotify user profile: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Spotify API returned status: %d", resp.StatusCode)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Spotify API returned non-200 status"})
			return
		}

		// Parse Spotify user info
		var spotifyUser struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Email       string `json:"email"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&spotifyUser); err != nil {
			log.Printf("Failed to parse Spotify user info: %v", err)
		} else {
			serviceUserID = spotifyUser.ID
			serviceUserName = spotifyUser.DisplayName
			if serviceUserName == "" && spotifyUser.Email != "" {
				serviceUserName = spotifyUser.Email
			}
			log.Printf("Spotify user: %s (%s)", serviceUserName, serviceUserID)
		}
	}

	// TODO: In a real implementation, you would:
	// 1. Validate the state parameter (should contain user ID)
	// 2. Get the actual user ID from JWT or state
	// 3. Store the token in database associated with the user

	// For now, we'll store with a placeholder user ID (user ID 1)
	userService := database.UserService{
		UserID:          1, // Placeholder - replace with actual user ID from JWT
		ServiceType:     provider,
		AccessToken:     token.AccessToken,
		RefreshToken:    token.RefreshToken,
		TokenExpiry:     token.Expiry.Unix(),
		ServiceUserID:   serviceUserID,
		ServiceUserName: serviceUserName,
	}

	// Check if service already exists for this user
	var existingService database.UserService
	result := database.DB.Where("user_id = ? AND service_type = ?", userService.UserID, provider).First(&existingService)

	switch result.Error {
	case gorm.ErrRecordNotFound:
		// Create new service connection
		if err := database.DB.Create(&userService).Error; err != nil {
			log.Printf("Failed to create service connection: %v", err)
		} else {
			log.Printf("Created new %s service connection for user %d", provider, userService.UserID)
		}
	case nil:
		// Update existing service connection
		existingService.AccessToken = userService.AccessToken
		existingService.RefreshToken = userService.RefreshToken
		existingService.TokenExpiry = userService.TokenExpiry
		existingService.ServiceUserID = userService.ServiceUserID
		existingService.ServiceUserName = userService.ServiceUserName

		if err := database.DB.Save(&existingService).Error; err != nil {
			log.Printf("Failed to update service connection: %v", err)
		} else {
			log.Printf("Updated %s service connection for user %d", provider, userService.UserID)
		}
	}

	// Redirect to frontend with success message
	frontendURL := os.Getenv("FRONTEND_URL")
	redirectURL := fmt.Sprintf("%s/dashboard?message=%s_connected", frontendURL, provider)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func HandleGetConnectedServices(c *gin.Context) {
	// TODO: Get user ID from JWT and fetch their connected services
	// For now, get all services (we'll filter by user later)
	var services []database.UserService
	result := database.DB.Find(&services)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	// Log the services for debugging
	log.Printf("Returning %d services to frontend", len(services))
	for _, service := range services {
		log.Printf("Service: %s, User: %s", service.ServiceType, service.ServiceUserName)
	}

	c.JSON(http.StatusOK, gin.H{"services": services})
}
