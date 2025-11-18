package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"server/internal/auth"
	"server/internal/database"
	"server/internal/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	googleRevocationURL  = "https://oauth2.googleapis.com/revoke"
	spotifyRevocationURL = "https://accounts.spotify.com/api/token"
)

func HandleConnectService(c *gin.Context) {
	provider := c.Param("provider")

	config := auth.GetOAuthConfig(provider)
	if config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported service provider"})
		return
	}

	userIDStr := c.Query("user_id")
	var userID uint = 1

	if userIDStr != "" {
		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			userID = uint(id)
		}
	}

	state := fmt.Sprintf("user-%d", userID)

	var authURL string
	switch provider {
	case "spotify":
		authURL = config.AuthCodeURL(state,
			oauth2.SetAuthURLParam("show_dialog", "true"),
			oauth2.SetAuthURLParam("prompt", "login"),
		)
	case "youtube":
		authURL = config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	default:
		authURL = config.AuthCodeURL(state)
	}

	log.Printf("Redirecting user %d to %s OAuth: %s", userID, provider, authURL)

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func HandleServiceCallback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")
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
	switch provider {
	case "spotify":
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

	case "youtube":
		log.Printf("YouTube token obtained: %+v", token)
		client := config.Client(context.Background(), token)

		// First, try to get basic Google user info (this usually works with any Google scope)
		userInfoResp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			log.Printf("Failed to get Google user info: %v", err)
			// Continue without user info, we'll use a default
		} else {
			defer userInfoResp.Body.Close()
			var userInfo struct {
				Name  string `json:"name"`
				Email string `json:"email"`
				ID    string `json:"id"`
			}
			if err := json.NewDecoder(userInfoResp.Body).Decode(&userInfo); err == nil {
				serviceUserID = userInfo.ID
				serviceUserName = userInfo.Name
				if serviceUserName == "" {
					serviceUserName = userInfo.Email
				}
				log.Printf("YouTube user (from Google profile): %s (%s)", serviceUserName, serviceUserID)
			} else {
				log.Printf("Failed to parse Google user info: %v", err)
			}
		}

		// Try to get YouTube channel info with the readonly scope
		// Note: This might fail with youtube.readonly scope, but let's try
		channelResp, err := client.Get("https://www.googleapis.com/youtube/v3/channels?part=snippet&mine=true")
		if err != nil {
			log.Printf("Failed to get YouTube channels (this might be expected with readonly scope): %v", err)
			// This is expected with youtube.readonly scope, so we don't treat it as an error
		} else {
			defer channelResp.Body.Close()

			if channelResp.StatusCode == http.StatusOK {
				// Parse YouTube channel info
				var youtubeResponse struct {
					Items []struct {
						ID      string `json:"id"`
						Snippet struct {
							Title string `json:"title"`
						} `json:"snippet"`
					} `json:"items"`
				}

				if err := json.NewDecoder(channelResp.Body).Decode(&youtubeResponse); err != nil {
					log.Printf("Failed to parse YouTube response: %v", err)
				} else if len(youtubeResponse.Items) > 0 {
					// If we successfully got channel info, use it instead of basic user info
					serviceUserID = youtubeResponse.Items[0].ID
					serviceUserName = youtubeResponse.Items[0].Snippet.Title
					log.Printf("YouTube channel: %s (%s)", serviceUserName, serviceUserID)
				}
			} else {
				body, _ := io.ReadAll(channelResp.Body)
				log.Printf("YouTube channels API returned status: %d, body: %s", channelResp.StatusCode, string(body))
				// This is normal with reduced scopes, so we don't treat it as an error
			}
		}

		// If we still don't have user info, set a default
		if serviceUserName == "" {
			serviceUserName = "YouTube User"
			log.Printf("Using default YouTube user name")
		}
	}

	// Extract user ID from state parameter for security
	var userID uint = 1 // Default fallback

	// Simple state parsing: "user-{id}"
	if len(state) > 5 && state[:5] == "user-" {
		if id, err := strconv.ParseUint(state[5:], 10, 32); err == nil {
			userID = uint(id)
		}
	}

	userService := database.UserService{
		UserID:          userID,
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
	// Get user from context
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get services for the authenticated user only
	var services []database.UserService
	result := database.DB.Where("user_id = ?", user.ID).Find(&services)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	// Log for debugging
	log.Printf("Returning %d services for user %d", len(services), user.ID)

	c.JSON(http.StatusOK, gin.H{"services": services})
}

func HandleDisconnectService(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	provider := c.Param("provider")

	// Validate provider
	if provider != "spotify" && provider != "youtube" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported service provider"})
		return
	}

	// Get the service connection first
	var userService database.UserService
	result := database.DB.Where("user_id = ? AND service_type = ?", user.ID, provider).First(&userService)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service connection not found"})
		return
	}

	// Revoke the token before deleting
	if err := revokeServiceToken(provider, userService.AccessToken); err != nil {
		log.Printf("Failed to revoke token for %s: %v", provider, err)
		// Continue with deletion even if revocation fails
	}

	// Delete the service connection
	result = database.DB.Where("user_id = ? AND service_type = ?", user.ID, provider).Delete(&database.UserService{})
	if result.Error != nil {
		log.Printf("Failed to delete service connection: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect service"})
		return
	}

	// Also delete any playlists associated with this service
	database.DB.Where("user_id = ? AND service_type = ?", user.ID, provider).Delete(&database.Playlist{})

	log.Printf("User %d disconnected %s service", user.ID, provider)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully disconnected %s", getServiceDisplayName(provider)),
	})
}

func revokeServiceToken(provider, accessToken string) error {
	switch provider {
	case "spotify":
		return revokeSpotifyToken(accessToken)
	case "youtube":
		return revokeGoogleToken(accessToken)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

func revokeSpotifyToken(accessToken string) error {
	config := auth.GetOAuthConfig("spotify")
	if config == nil {
		return fmt.Errorf("spotify OAuth config not found")
	}

	data := url.Values{}
	data.Set("token", accessToken)
	data.Set("token_type_hint", "access_token")

	req, err := http.NewRequest("POST", spotifyRevocationURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(config.ClientID, config.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("spotify revocation returned status: %d", resp.StatusCode)
	}

	log.Printf("Successfully revoked Spotify token")
	return nil
}

func revokeGoogleToken(accessToken string) error {
	req, err := http.NewRequest("POST", googleRevocationURL, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("token", accessToken)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google revocation returned status: %d", resp.StatusCode)
	}

	log.Printf("Successfully revoked Google token")
	return nil
}

func getServiceDisplayName(serviceType string) string {
	switch serviceType {
	case "spotify":
		return "Spotify"
	case "youtube":
		return "YouTube Music"
	default:
		return serviceType
	}
}

func HandleTokenHealth(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get all services for the user
	var services []database.UserService
	result := database.DB.Where("user_id = ?", user.ID).Find(&services)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	healthStatus := make(map[string]interface{})
	for _, service := range services {
		valid, err := tokenManager.ValidateToken(&service)
		status := "healthy"
		if err != nil || !valid {
			status = "unhealthy"
		}

		healthStatus[service.ServiceType] = map[string]interface{}{
			"status":     status,
			"error":      err,
			"expires_in": time.Until(time.Unix(service.TokenExpiry, 0)).String(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"health": healthStatus,
	})
}
