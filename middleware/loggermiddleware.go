package middleware

import (
	"crypto/rand"
	"encoding/hex"
	logger "example.com/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
	"time"
)

// MiddlewareConfig holds configuration for the logger middleware
type MiddlewareConfig struct {
	Logger            *logger.Logger
	SkipPaths         []string
	EnableBodyLogging bool
	MaxBodySize       int64
}

// LoggerMiddleware creates a new logger middleware with the given configuration
func LoggerMiddleware(config MiddlewareConfig) gin.HandlerFunc {
	// Set defaults
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 32 * 1024 // 32KB default
	}

	skipMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Skip logging for certain paths
			if skipMap[param.Path] {
				return ""
			}

			// Extract request ID from context
			requestID := ""
			if param.Keys != nil {
				if id, exists := param.Keys["requestID"]; exists {
					if idStr, ok := id.(string); ok {
						requestID = idStr
					}
				}
			}

			// Prepare log fields
			fields := map[string]interface{}{
				"request_id":  requestID,
				"method":      param.Method,
				"path":        param.Path,
				"status":      param.StatusCode,
				"duration_ms": param.Latency.Milliseconds(),
				"client_ip":   param.ClientIP,
				"user_agent":  param.Request.UserAgent(),
				"body_size":   param.BodySize,
			}

			// Add error information if present
			if param.ErrorMessage != "" {
				fields["error"] = param.ErrorMessage
			}

			// Determine log level based on status code
			var level logger.LogLevel
			switch {
			case param.StatusCode >= 500:
				level = logger.ErrorLevel
			case param.StatusCode >= 400:
				level = logger.WarnLevel
			default:
				level = logger.InfoLevel
			}

			// Log the request
			message := fmt.Sprintf("%s %s - %d", param.Method, param.Path, param.StatusCode)
			if err := config.Logger.Log(level, message, fields); err != nil {
				// Fallback to stderr if logging fails
				fmt.Fprintf(gin.DefaultErrorWriter, "Failed to log request: %v\n", err)
			}

			return "" // Return empty string as we handle logging ourselves
		},
		Output: io.Discard, // Discard default output since we handle it
	})
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header
		requestID := c.GetHeader("X-Request-ID")

		// Generate new request ID if not provided
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Store in context and set response header
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// BodyLoggingMiddleware logs request and response bodies (use carefully in production)
func BodyLoggingMiddleware(logger *logger.Logger, maxBodySize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := getRequestID(c)

		// Log request body if content type is suitable
		if shouldLogBody(c.Request) && c.Request.ContentLength <= maxBodySize {
			if body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize)); err == nil {
				c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

				logger.Debug("Request body", map[string]interface{}{
					"request_id": requestID,
					"body":       string(body),
					"size":       len(body),
				})
			}
		}

		c.Next()
	}
}

// ErrorLoggingMiddleware logs detailed error information
func ErrorLoggingMiddleware(logger *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log any errors that occurred during request processing
		if len(c.Errors) > 0 {
			requestID := getRequestID(c)

			for _, err := range c.Errors {
				fields := map[string]interface{}{
					"request_id": requestID,
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"error_type": err.Type,
				}

				// Add metadata if available
				if err.Meta != nil {
					fields["error_meta"] = err.Meta
				}

				logger.Error("Request error", fields)
			}
		}
	}
}

// RecoveryMiddleware provides panic recovery with logging
func RecoveryMiddleware(logger *logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := getRequestID(c)

		logger.Error("Panic recovered", map[string]interface{}{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"client_ip":  c.ClientIP(),
			"panic":      fmt.Sprintf("%v", recovered),
		})

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":      "Internal server error",
			"request_id": requestID,
		})
	})
}

// generateRequestID creates a cryptographically secure random request ID
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("req_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
	}
	return "req_" + hex.EncodeToString(bytes)
}

// getRequestID safely extracts request ID from context
func getRequestID(c *gin.Context) string {
	if id, exists := c.Get("requestID"); exists {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}
	return "unknown"
}

// shouldLogBody determines if request body should be logged based on content type
func shouldLogBody(req *http.Request) bool {
	contentType := req.Header.Get("Content-Type")

	// Only log text-based content types
	textTypes := []string{
		"application/json",
		"application/xml",
		"text/",
		"application/x-www-form-urlencoded",
	}

	for _, textType := range textTypes {
		if strings.Contains(strings.ToLower(contentType), textType) {
			return true
		}
	}

	return false
}
