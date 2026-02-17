// controllers/health.go
package controllers

import (
	"context"
	"net/http"
	"time"

	"example.com/database"
	"github.com/gin-gonic/gin"
)

// HealthHandler returns application health status
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		health := gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
			"uptime":    time.Since(startTime).String(),
		}

		// If database is available, check its health
		if db := getDBFromContext(c); db != nil {
			if err := db.Health(ctx); err != nil {
				health["status"] = "unhealthy"
				health["database"] = "down"
				health["error"] = err.Error()
				c.JSON(http.StatusServiceUnavailable, health)
				return
			}
			health["database"] = "up"
		}

		c.JSON(http.StatusOK, health)
	}
}

// MetricsHandler returns application metrics
func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics := gin.H{
			"timestamp": time.Now().UTC(),
			"uptime":    time.Since(startTime).String(),
		}

		// If database is available, include DB stats
		if db := getDBFromContext(c); db != nil {
			metrics["database"] = db.GetStats()
		}

		c.JSON(http.StatusOK, metrics)
	}
}

// Global variable to track start time
var startTime = time.Now()

// Helper function to get database from context
func getDBFromContext(c *gin.Context) *database.DB {
	if db, exists := c.Get("database"); exists {
		if dbInstance, ok := db.(*database.DB); ok {
			return dbInstance
		}
	}
	return nil
}
