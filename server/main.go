package main

import (
	"log"
	"os"

	"server/internal/auth"
	"server/internal/database"
	"server/internal/handlers"
	"server/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize OAuth providers
	auth.InitOAuthConfigs()

	// Set up Gin
	r := gin.Default()

	// CORS configuration for local development
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://client:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// API routes
	api := r.Group("/api")
	{
		// Public auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.GET("/google", handlers.HandleGoogleLogin)
			authGroup.GET("/google/callback", handlers.HandleGoogleCallback)
			authGroup.POST("/logout", handlers.HandleLogout)
		}

		// Service connection routes (public for OAuth flow)
		servicesGroup := api.Group("/services")
		{
			// These need to be public because they're called via browser redirects
			servicesGroup.GET("/connect/:provider", handlers.HandleConnectService)
			servicesGroup.GET("/callback/:provider", handlers.HandleServiceCallback)
		}

		// Protected routes (require JWT)
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/auth/me", handlers.HandleGetCurrentUser)
			protected.GET("/services", handlers.HandleGetConnectedServices)

			// Playlists routes
			playlistsGroup := protected.Group("/playlists")
			{
				playlistsGroup.GET("/:service", handlers.GetPlaylists)
				playlistsGroup.GET("/:service/stored", handlers.GetStoredPlaylists)
				playlistsGroup.POST("/sync", handlers.SyncAllPlaylists)
			}

			transfersGroup := protected.Group("/transfers")
			{
				transfersGroup.POST("", handlers.StartTransfer)
				transfersGroup.GET("", handlers.GetTransfers)
				transfersGroup.GET("/:id", handlers.GetTransferDetails)
			}
		}

		// Health check (public)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}
