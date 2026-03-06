package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"server/internal/auth"
	"server/internal/database"
	"server/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID uint) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.FormatUint(uint64(userID), 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func HandleGoogleLogin(c *gin.Context) {
	state, nonce, err := auth.GenerateOAuthState(auth.StatePurposeLogin, "google", 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize OAuth state"})
		return
	}

	setCookie(c, auth.OAuthStateCookieName(auth.StatePurposeLogin, "google"), nonce, 600, true)
	url := auth.GoogleOAuthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func HandleGoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not provided"})
		return
	}

	stateCookieName := auth.OAuthStateCookieName(auth.StatePurposeLogin, "google")
	stateCookie, err := c.Cookie(stateCookieName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OAuth state cookie not found"})
		return
	}
	if _, err := auth.ValidateOAuthState(state, auth.StatePurposeLogin, "google", stateCookie); err != nil {
		clearCookie(c, stateCookieName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state"})
		return
	}
	clearCookie(c, stateCookieName)

	// Exchange code for token
	token, err := auth.GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Token exchange error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange token: " + err.Error()})
		return
	}

	// Get user info from Google
	client := auth.GoogleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("User info fetch error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user info: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// Parse user info
	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Printf("User info decode error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user info: " + err.Error()})
		return
	}

	// Create or update user in database
	var user database.User
	result := database.DB.Where("google_id = ?", userInfo.ID).First(&user)
	if result.Error == gorm.ErrRecordNotFound {
		user = database.User{
			GoogleID:  userInfo.ID,
			Email:     userInfo.Email,
			Name:      userInfo.Name,
			AvatarURL: userInfo.Picture,
		}
		if err := database.DB.Create(&user).Error; err != nil {
			log.Printf("User creation error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
			return
		}
		log.Printf("Created new user: %s", user.Email)
	} else if result.Error != nil {
		log.Printf("Database error: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + result.Error.Error()})
		return
	} else {
		log.Printf("Logged in existing user: %s", user.Email)
	}

	// Generate JWT
	jwtToken, err := GenerateJWT(user.ID)
	if err != nil {
		log.Printf("JWT generation error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	setCookie(c, auth.SessionCookieName, jwtToken, int((24 * time.Hour).Seconds()), true)

	// Redirect to frontend with authenticated session cookie
	frontendURL := os.Getenv("FRONTEND_URL")
	redirectURL := fmt.Sprintf("%s/dashboard?message=login_success", frontendURL)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func HandleLogout(c *gin.Context) {
	clearCookie(c, auth.SessionCookieName)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func HandleGetCurrentUser(c *gin.Context) {
	user, exists := middleware.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"name":      user.Name,
			"avatarURL": user.AvatarURL,
		},
	})
}
