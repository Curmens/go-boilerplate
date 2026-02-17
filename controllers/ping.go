// controllers/ping.go
package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Ping handles ping requests for basic health check
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "pong",
		"timestamp": time.Now().UTC(),
		"status":    "ok",
	})
}
