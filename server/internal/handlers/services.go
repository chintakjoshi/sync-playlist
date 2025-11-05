package handlers

import (
	"fmt"
	"net/http"

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
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func HandleServiceCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not provided"})
		return
	}

	config := auth.GetOAuthConfig(provider)
	if config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported service provider"})
		return
	}

	// Exchange code for token
	token, err := config.Exchange(c, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token: " + err.Error()})
		return
	}

	// TODO: In a real implementation, you would:
	// 1. Validate the state parameter (should contain user ID)
	// 2. Get user info from the service using the access token
	// 3. Store the token in database associated with the user

	// For now, we'll just return success
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully connected to %s", provider),
		"token":   token,
	})
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
