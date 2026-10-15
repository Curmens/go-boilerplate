package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/cache"
	"example.com/utils"

	"github.com/gin-gonic/gin"
)

// MiddlewareConfig holds configuration for the logger middleware
type MiddlewareConfig struct {
	Logger    *utils.Logger
	SkipPaths []string
}

// LoggerMiddleware logs one structured line per request; level follows the status code
func LoggerMiddleware(config MiddlewareConfig) gin.HandlerFunc {
	skipMap := make(map[string]bool, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			if skipMap[param.Path] {
				return ""
			}

			fields := map[string]interface{}{
				"request_id":  param.Keys[utils.RequestIDKey],
				"method":      param.Method,
				"path":        param.Path,
				"status":      param.StatusCode,
				"duration_ms": param.Latency.Milliseconds(),
				"client_ip":   param.ClientIP,
				"user_agent":  param.Request.UserAgent(),
				"body_size":   param.BodySize,
			}

			for _, key := range []string{"cache_backend", "cache_key", "cache_hit"} {
				if value, exists := param.Keys[key]; exists {
					fields[key] = value
				}
			}

			if param.ErrorMessage != "" {
				fields["error"] = param.ErrorMessage
			}

			var level utils.LogLevel
			switch {
			case param.StatusCode >= 500:
				level = utils.ErrorLevel
			case param.StatusCode >= 400:
				level = utils.WarnLevel
			default:
				level = utils.InfoLevel
			}

			message := fmt.Sprintf("%s %s - %d", param.Method, param.Path, param.StatusCode)
			if err := config.Logger.Log(level, message, fields); err != nil {
				fmt.Fprintf(gin.DefaultErrorWriter, "Failed to log request: %v\n", err)
			}

			return "" // We handle logging ourselves
		},
		Output: io.Discard,
	})
}

// CacheTraceMiddleware records which cache key a request touched (and whether it hit)
// so LoggerMiddleware can include it in the request log line.
func CacheTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(cache.WithRequestTrace(c.Request.Context()))

		c.Next()

		trace, ok := cache.GetRequestTrace(c.Request.Context())
		if !ok {
			return
		}
		if trace.Backend != "" {
			c.Set("cache_backend", trace.Backend)
		}
		if trace.Key != "" {
			c.Set("cache_key", trace.Key)
		}
		if trace.HasHit {
			c.Set("cache_hit", trace.Hit)
		}
	}
}

// RequestIDMiddleware propagates X-Request-ID, generating one when absent
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" || len(requestID) > 128 {
			requestID = generateRequestID()
		}

		c.Set(utils.RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// responseBodyWriter captures up to limit bytes of the response body for logging
type responseBodyWriter struct {
	gin.ResponseWriter
	body  *bytes.Buffer
	limit int
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	if remaining := w.limit - w.body.Len(); remaining > 0 {
		if len(b) > remaining {
			w.body.Write(b[:remaining])
		} else {
			w.body.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

// BodyLoggingMiddleware logs request and response bodies at debug level.
// Bodies may contain credentials — only enable in development.
func BodyLoggingMiddleware(logger *utils.Logger, maxBodySize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString(utils.RequestIDKey)

		if isTextContent(c.GetHeader("Content-Type")) && c.Request.ContentLength <= maxBodySize {
			if body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize)); err == nil {
				c.Request.Body = io.NopCloser(bytes.NewReader(body))
				_ = logger.Debug("Request body", map[string]interface{}{
					"request_id": requestID,
					"body":       string(body),
					"size":       len(body),
				})
			}
		}

		rbw := &responseBodyWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}, limit: int(maxBodySize)}
		c.Writer = rbw

		c.Next()

		if isTextContent(rbw.Header().Get("Content-Type")) {
			body := rbw.body.String()
			if rbw.Size() > rbw.body.Len() {
				body += "... [truncated]"
			}
			_ = logger.Debug("Response body", map[string]interface{}{
				"request_id": requestID,
				"status":     rbw.Status(),
				"body":       body,
				"size":       rbw.Size(),
			})
		}
	}
}

// ErrorLoggingMiddleware logs errors attached to the context via c.Error()
func ErrorLoggingMiddleware(logger *utils.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, err := range c.Errors {
			fields := map[string]interface{}{
				"request_id": c.GetString(utils.RequestIDKey),
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"error":      err.Error(),
				"error_type": err.Type,
			}
			if err.Meta != nil {
				fields["error_meta"] = err.Meta
			}
			_ = logger.Error("Request error", fields)
		}
	}
}

// RecoveryMiddleware recovers from panics, logs them and returns a 500 envelope
func RecoveryMiddleware(logger *utils.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		_ = logger.Error("Panic recovered", map[string]interface{}{
			"request_id": c.GetString(utils.RequestIDKey),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"client_ip":  c.ClientIP(),
			"panic":      fmt.Sprintf("%v", recovered),
		})

		utils.AbortWithError(c, http.StatusInternalServerError, utils.ErrCodeInternal, "Internal server error")
	})
}

// generateRequestID creates a cryptographically secure random request ID
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		now := time.Now()
		return fmt.Sprintf("req_%d_%d", now.Unix(), now.UnixNano()%1000000)
	}
	return "req_" + hex.EncodeToString(b)
}

// isTextContent reports whether a content type is worth logging as text
func isTextContent(contentType string) bool {
	contentType = strings.ToLower(contentType)
	for _, textType := range []string{"application/json", "application/xml", "text/", "application/x-www-form-urlencoded"} {
		if strings.Contains(contentType, textType) {
			return true
		}
	}
	return false
}
