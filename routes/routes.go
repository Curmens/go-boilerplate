package routes

import (
	"example.com/middleware"
	"example.com/utils"

	"github.com/gin-gonic/gin"
)

// RouterConfig holds dependencies for route setup
type RouterConfig struct {
	Logger *utils.Logger
}

// SetupRouter creates and configures the Gin router
func SetupRouter(config RouterConfig) *gin.Engine {
	// Create Gin router
	r := gin.New()

	// Add middleware in order of execution
	r.Use(middleware.RequestIDMiddleware())

	// CORS middleware should be early in the chain
	r.Use(middleware.CORSMiddlewareCustom(middleware.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With", "X-CSRF-Token"},
	}))

	// Recovery middleware should be early in the chain
	r.Use(middleware.RecoveryMiddleware(config.Logger))

	// Main logging middleware
	r.Use(middleware.LoggerMiddleware(middleware.MiddlewareConfig{
		Logger:            config.Logger,
		SkipPaths:         []string{"/health", "/metrics", "/ping"},
		EnableBodyLogging: gin.Mode() == gin.DebugMode,
		MaxBodySize:       32 * 1024, // 32KB
	}))

	// Error logging middleware
	r.Use(middleware.ErrorLoggingMiddleware(config.Logger))

	// Security headers
	//r.Use(middleware.SecurityHeadersMiddleware())

	// Body logging middleware for development
	if gin.Mode() == gin.DebugMode {
		r.Use(middleware.BodyLoggingMiddleware(config.Logger, 1024)) // 1KB limit for dev
	}

	// Setup routes
	setupPublicRoutes(r, config)
	// setupAuthenticatedRoutes(r, config)

	return r
}

// setupPublicRoutes configures public routes that don't require authentication
func setupPublicRoutes(router *gin.Engine, config RouterConfig) {
	// Health check endpoints (boilerplate placeholder handlers)
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	router.GET("/metrics", func(c *gin.Context) { c.JSON(200, gin.H{"metrics": "not_implemented"}) })
	router.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })

	// API v1 group
	api := router.Group("/api/v1")
	{
		// Authentication routes (public, boilerplate placeholder handlers)
		auth := api.Group("/auth")
		{
			auth.POST("/login", func(c *gin.Context) { c.JSON(200, gin.H{"message": "login not implemented"}) })
			auth.POST("/register", func(c *gin.Context) { c.JSON(200, gin.H{"message": "register not implemented"}) })
			auth.POST("/refresh", func(c *gin.Context) { c.JSON(200, gin.H{"message": "refresh not implemented"}) })
			auth.POST("/forgot-password", func(c *gin.Context) { c.JSON(200, gin.H{"message": "forgot-password not implemented"}) })
			auth.POST("/reset-password", func(c *gin.Context) { c.JSON(200, gin.H{"message": "reset-password not implemented"}) })

			// OAuth routes (boilerplate placeholder handlers)
			oauth := auth.Group("/oauth")
			{
				google := oauth.Group("/google")
				{
					google.GET("/login", func(c *gin.Context) { c.JSON(200, gin.H{"message": "google oauth login not implemented"}) })
					google.GET("/callback", func(c *gin.Context) { c.JSON(200, gin.H{"message": "google oauth callback not implemented"}) })
				}
				apple := oauth.Group("/apple")
				{
					apple.GET("/login", func(c *gin.Context) { c.JSON(200, gin.H{"message": "apple oauth login not implemented"}) })
					apple.POST("/callback", func(c *gin.Context) { c.JSON(200, gin.H{"message": "apple oauth callback not implemented"}) })
				}
			}
		}
		// Example: add more public routes here as needed
	}
}
