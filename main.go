package main

import (
	"example.com/config"
	"example.com/database"
	"example.com/routes"
	logger "example.com/utils"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// load configs
	cfg, _ := config.Load()

	// Initialize logger
	appLogger, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer appLogger.Close()

	//database.ConnectDb(appConfig.Database, appLogger)
	// Initialize database
	db, err := database.New(&cfg.Database, appLogger)
	if err != nil {
		appLogger.Error("Failed to connect to database", map[string]interface{}{
			"error": err.Error(),
		})
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	// Setup router with dependencies
	routerConfig := routes.RouterConfig{
		Logger: appLogger,
	}

	router := routes.SetupRouter(routerConfig)
	// Add database to context for controllers that need it
	router.Use(func(c *gin.Context) {
		c.Set("database", db.DB)
		c.Next()
	})

	// Start server
	host := cfg.Server.Host
	port := cfg.Server.Port

	appLogger.Info("Starting server", map[string]interface{}{
		"host": host,
		"port": port,
		"mode": gin.Mode(),
	})

	// Setup graceful shutdown
	go func() {
		if err := router.Run(host + ":" + port); err != nil {
			appLogger.Error("Failed to start server", map[string]interface{}{
				"error": err.Error(),
			})
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...", nil)
}
