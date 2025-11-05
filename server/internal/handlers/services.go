package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"server/internal/auth"
	"server/internal/database"

	"github.com/gin-gonic/gin"
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

	// For Spotify, let's get the user's profile to verify the connection
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

		// Parse Spotify user info (simplified)
		// TODO: Parse the response body properly
		log.Printf("Successfully connected to Spotify for user")
	}

	// TODO: In a real implementation, you would:
	// 1. Validate the state parameter (should contain user ID)
	// 2. Get user info from the service using the access token
	// 3. Store the token in database associated with the user

	// For now, redirect to frontend with success message
	frontendURL := os.Getenv("FRONTEND_URL")
	redirectURL := fmt.Sprintf("%s/dashboard?message=%s_connected", frontendURL, provider)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func HandleGetConnectedServices(c *gin.Context) {
	// TODO: Get user ID from JWT and fetch their connected services
	var services []database.UserService
	result := database.DB.Find(&services)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"services": services})
}
