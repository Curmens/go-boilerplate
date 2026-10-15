package controllers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"example.com/cache"
	"example.com/database"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// HealthController serves liveness, readiness and runtime metrics endpoints
type HealthController struct {
	DB      *database.DB // nil when the database is disabled
	Version string
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
	Database  string    `json:"database"`
	Error     string    `json:"error,omitempty"`
}

// Health reports overall status including database connectivity
func (h *HealthController) Health(c *gin.Context) {
	health := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   h.Version,
		Uptime:    time.Since(startTime).Round(time.Second).String(),
		Database:  "disabled",
	}

	if h.DB != nil {
		if err := h.DB.Health(c.Request.Context()); err != nil {
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

// Liveness is the Kubernetes liveness probe: the process is up
func (h *HealthController) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Readiness is the Kubernetes readiness probe: dependencies are reachable
func (h *HealthController) Readiness(c *gin.Context) {
	if h.DB != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := h.DB.Health(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": "database unavailable",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// Metrics returns runtime, cache and connection-pool statistics as JSON
func (h *HealthController) Metrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := gin.H{
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(startTime).Round(time.Second).String(),
		"cache":     cache.GetMetrics(),
		"runtime": gin.H{
			"goroutines": runtime.NumGoroutine(),
			"go_version": runtime.Version(),
			"num_cpu":    runtime.NumCPU(),
		},
		"memory": gin.H{
			"alloc_mb":       bToMb(m.Alloc),
			"total_alloc_mb": bToMb(m.TotalAlloc),
			"sys_mb":         bToMb(m.Sys),
			"heap_alloc_mb":  bToMb(m.HeapAlloc),
			"heap_objects":   m.HeapObjects,
			"num_gc":         m.NumGC,
		},
	}

	if h.DB != nil {
		metrics["database"] = h.DB.GetStats()
	}

	c.JSON(http.StatusOK, metrics)
}

func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}
