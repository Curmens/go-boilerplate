// controllers/health.go
package controllers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"example.com/database"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Global variable to track start time
var startTime = time.Now()

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Database  string                 `json:"database,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthHandler returns application health status
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		health := HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().UTC(),
			Version:   "1.0.0",
			Uptime:    time.Since(startTime).String(),
		}

		// Check database health
		if db := getDBInstanceFromContext(c); db != nil {
			if err := db.Health(ctx); err != nil {
				health.Status = "unhealthy"
				health.Database = "down"
				health.Error = err.Error()
				c.JSON(http.StatusServiceUnavailable, health)
				return
			}
			health.Database = "up"
		} else if gormDB := getGormDBFromContext(c); gormDB != nil {
			// Fallback to raw gorm.DB check
			sqlDB, err := gormDB.DB()
			if err != nil {
				health.Status = "unhealthy"
				health.Database = "error"
				health.Error = err.Error()
				c.JSON(http.StatusServiceUnavailable, health)
				return
			}
			if err := sqlDB.PingContext(ctx); err != nil {
				health.Status = "unhealthy"
				health.Database = "down"
				health.Error = err.Error()
				c.JSON(http.StatusServiceUnavailable, health)
				return
			}
			health.Database = "up"
		}

		c.JSON(http.StatusOK, health)
	}
}

// LivenessHandler for Kubernetes liveness probe
func LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
		})
	}
}

// ReadinessHandler for Kubernetes readiness probe
func ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		// Check database readiness
		if db := getDBInstanceFromContext(c); db != nil {
			if err := db.Health(ctx); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "not ready",
					"reason": "database unavailable",
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	}
}

// MetricsHandler returns application metrics
func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		metrics := gin.H{
			"timestamp": time.Now().UTC(),
			"uptime":    time.Since(startTime).String(),
			"runtime": gin.H{
				"goroutines":    runtime.NumGoroutine(),
				"go_version":    runtime.Version(),
				"num_cpu":       runtime.NumCPU(),
				"num_cgo_calls": runtime.NumCgoCall(),
			},
			"memory": gin.H{
				"alloc_mb":       bToMb(m.Alloc),
				"total_alloc_mb": bToMb(m.TotalAlloc),
				"sys_mb":         bToMb(m.Sys),
				"heap_alloc_mb":  bToMb(m.HeapAlloc),
				"heap_sys_mb":    bToMb(m.HeapSys),
				"heap_objects":   m.HeapObjects,
				"num_gc":         m.NumGC,
			},
		}

		// Add database stats if available
		if db := getDBInstanceFromContext(c); db != nil {
			metrics["database"] = db.GetStats()
		}

		c.JSON(http.StatusOK, metrics)
	}
}

// getDBInstanceFromContext extracts *database.DB from context
func getDBInstanceFromContext(c *gin.Context) *database.DB {
	if db, exists := c.Get("db"); exists {
		if dbInstance, ok := db.(*database.DB); ok {
			return dbInstance
		}
	}
	return nil
}

// getGormDBFromContext extracts *gorm.DB from context (fallback)
func getGormDBFromContext(c *gin.Context) *gorm.DB {
	if db, exists := c.Get("database"); exists {
		if gormDB, ok := db.(*gorm.DB); ok {
			return gormDB
		}
	}
	return nil
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}
