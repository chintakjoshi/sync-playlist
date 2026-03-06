package middleware

import (
	"net/http"
	"os"
	"strings"

	"server/internal/auth"
	"server/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware verifies JWT tokens and sets user in context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := getTokenFromRequest(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Parse and validate token
		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Get user ID from claims subject
		userID := claims.Subject
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Get user from database
		var user database.User
		result := database.DB.First(&user, userID)
		if result.Error != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Add user to context
		c.Set("user", user)
		c.Next()
	}
}

func getTokenFromRequest(c *gin.Context) (string, error) {
	if cookieToken, err := c.Cookie(auth.SessionCookieName); err == nil && cookieToken != "" {
		return cookieToken, nil
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", httpError("authorization required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", httpError("authorization header format must be Bearer {token}")
	}

	return parts[1], nil
}

type httpError string

func (e httpError) Error() string {
	return string(e)
}

// GetUserFromContext retrieves the user from context (to be used in handlers)
func GetUserFromContext(c *gin.Context) (*database.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	dbUser, ok := user.(database.User)
	if !ok {
		return nil, false
	}

	return &dbUser, true
}
