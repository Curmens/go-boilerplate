package routes

import (
	"example.com/config"
	"example.com/controllers"
	"example.com/middleware"
	_ "example.com/utils"
	logger "example.com/utils"
	"github.com/gin-gonic/gin"
	"log"
)

func SetupRouter(config config.LoggerConfig) *gin.Engine {
	// Initialize logger
	appLogger, err := logger.NewLogger(config)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer appLogger.Close()

	// Create Gin router
	r := gin.New()

	// Add middleware in order of execution
	r.Use(middleware.RequestIDMiddleware())

	// Recovery middleware should be early in the chain
	r.Use(middleware.RecoveryMiddleware(appLogger))

	// Main logging middleware
	r.Use(middleware.LoggerMiddleware(middleware.MiddlewareConfig{
		Logger:            appLogger,
		SkipPaths:         []string{"/health", "/metrics"}, // Skip logging for these paths
		EnableBodyLogging: false,                           // Enable only if needed
		MaxBodySize:       32 * 1024,                       // 32KB
	}))

	// Error logging middleware
	r.Use(middleware.ErrorLoggingMiddleware(appLogger))

	if gin.Mode() == gin.DebugMode {
		r.Use(middleware.BodyLoggingMiddleware(appLogger, 1024)) // 1KB limit for dev
	}

	r.GET("/ping", controllers.Ping)

	r.Use(middleware.AuthMiddleware)

	// Start server
	appLogger.Info("Starting server", map[string]interface{}{
		"port": "8080",
		"mode": gin.Mode(),
	})

	if err := r.Run(":8080"); err != nil {
		appLogger.Error("Failed to start server", map[string]interface{}{
			"error": err.Error(),
		})
		log.Fatalf("Failed to start server: %v", err)
	}
	return r
}
