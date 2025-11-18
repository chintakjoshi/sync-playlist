package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"server/internal/database"

	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type TokenManager struct {
	db *gorm.DB
}

func NewTokenManager(db *gorm.DB) *TokenManager {
	return &TokenManager{db: db}
}

// RefreshTokenIfNeeded checks if token needs refresh and refreshes it
func (tm *TokenManager) RefreshTokenIfNeeded(userService *database.UserService) error {
	// Check if token is expired or will expire in the next 5 minutes
	if userService.TokenExpiry > time.Now().Add(5*time.Minute).Unix() {
		return nil // Token is still valid
	}

	log.Printf("Refreshing token for %s service (user %d)", userService.ServiceType, userService.UserID)

	config := GetOAuthConfig(userService.ServiceType)
	if config == nil {
		return fmt.Errorf("unsupported service: %s", userService.ServiceType)
	}

	// Create token source with refresh token
	token := &oauth2.Token{
		AccessToken:  userService.AccessToken,
		RefreshToken: userService.RefreshToken,
		Expiry:       time.Unix(userService.TokenExpiry, 0),
	}

	tokenSource := config.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %v", err)
	}

	// Update the service with new tokens
	userService.AccessToken = newToken.AccessToken
	if newToken.RefreshToken != "" {
		userService.RefreshToken = newToken.RefreshToken
	}
	userService.TokenExpiry = newToken.Expiry.Unix()

	return tm.db.Save(userService).Error
}

// ForceRefreshToken forces a token refresh regardless of expiry
func (tm *TokenManager) ForceRefreshToken(userService *database.UserService) error {
	// Set expiry to past to force refresh
	userService.TokenExpiry = time.Now().Add(-1 * time.Hour).Unix()
	return tm.RefreshTokenIfNeeded(userService)
}

// ValidateToken checks if token is valid by making a test API call
func (tm *TokenManager) ValidateToken(userService *database.UserService) (bool, error) {
	// First refresh if needed
	if err := tm.RefreshTokenIfNeeded(userService); err != nil {
		return false, err
	}

	// Make a test API call to validate the token
	switch userService.ServiceType {
	case "spotify":
		return tm.validateSpotifyToken(userService.AccessToken)
	case "youtube":
		return tm.validateYouTubeToken(userService.AccessToken)
	default:
		return false, fmt.Errorf("unsupported service: %s", userService.ServiceType)
	}
}

func (tm *TokenManager) validateSpotifyToken(accessToken string) (bool, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (tm *TokenManager) validateYouTubeToken(accessToken string) (bool, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
