package main

import (
	"log"
	"os"

	"server/internal/auth"
	"server/internal/database"
	"server/internal/handlers"

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
		// Auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.GET("/google", handlers.HandleGoogleLogin)
			authGroup.GET("/google/callback", handlers.HandleGoogleCallback)
			authGroup.POST("/logout", handlers.HandleLogout)
			authGroup.GET("/me", handlers.HandleGetCurrentUser)
		}

		// Service connection routes
		servicesGroup := api.Group("/services")
		{
			servicesGroup.GET("/connect/:provider", handlers.HandleConnectService)
			servicesGroup.GET("/callback/:provider", handlers.HandleServiceCallback)
			servicesGroup.GET("", handlers.HandleGetConnectedServices)
		}

		// Health check
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
