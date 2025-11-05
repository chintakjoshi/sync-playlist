package auth

import (
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/spotify"
)

var (
	GoogleOAuthConfig  *oauth2.Config
	SpotifyOAuthConfig *oauth2.Config
	YouTubeOAuthConfig *oauth2.Config
)

func InitOAuthConfigs() {
	// Google OAuth for app login
	GoogleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("BACKEND_URL") + "/api/auth/google/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	// Spotify OAuth
	SpotifyOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("BACKEND_URL") + "/api/services/callback/spotify",
		Scopes:       []string{"playlist-read-private", "playlist-read-collaborative", "playlist-modify-public", "playlist-modify-private"},
		Endpoint:     spotify.Endpoint,
	}

	// YouTube OAuth (for YouTube Music)
	YouTubeOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("BACKEND_URL") + "/api/services/callback/youtube",
		Scopes: []string{
			"https://www.googleapis.com/auth/youtube",          // Manage YouTube account
			"https://www.googleapis.com/auth/youtube.readonly", // View YouTube data
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func GetOAuthConfig(provider string) *oauth2.Config {
	switch provider {
	case "google":
		return GoogleOAuthConfig
	case "spotify":
		return SpotifyOAuthConfig
	case "youtube":
		return YouTubeOAuthConfig
	default:
		return nil
	}
}
