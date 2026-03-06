package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	SessionCookieName = "sync_playlist_session"
	StatePurposeLogin = "google_login"
	StatePurposeLink  = "service_link"
)

type OAuthStateClaims struct {
	Purpose  string `json:"purpose"`
	Provider string `json:"provider,omitempty"`
	UserID   uint   `json:"user_id,omitempty"`
	Nonce    string `json:"nonce"`
	jwt.RegisteredClaims
}

func GenerateOAuthState(purpose, provider string, userID uint) (string, string, error) {
	nonce, err := generateNonce()
	if err != nil {
		return "", "", err
	}

	claims := OAuthStateClaims{
		Purpose:  purpose,
		Provider: provider,
		UserID:   userID,
		Nonce:    nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", "", err
	}

	return signedToken, nonce, nil
}

func ValidateOAuthState(tokenString, expectedPurpose, expectedProvider, cookieNonce string) (*OAuthStateClaims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("missing OAuth state")
	}
	if cookieNonce == "" {
		return nil, fmt.Errorf("missing OAuth state cookie")
	}

	claims := &OAuthStateClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid OAuth state")
	}

	if claims.Purpose != expectedPurpose {
		return nil, fmt.Errorf("invalid OAuth state purpose")
	}
	if expectedProvider != "" && claims.Provider != expectedProvider {
		return nil, fmt.Errorf("invalid OAuth state provider")
	}
	if claims.Nonce != cookieNonce {
		return nil, fmt.Errorf("OAuth state nonce mismatch")
	}

	return claims, nil
}

func OAuthStateCookieName(purpose, provider string) string {
	parts := []string{"oauth_state", sanitizeCookiePart(purpose)}
	if provider != "" {
		parts = append(parts, sanitizeCookiePart(provider))
	}
	return strings.Join(parts, "_")
}

func generateNonce() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func sanitizeCookiePart(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, " ", "_")
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '-':
			return r
		default:
			return -1
		}
	}, value)
}
