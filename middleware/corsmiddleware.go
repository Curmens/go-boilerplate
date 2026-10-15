package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"example.com/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware handles CORS preflight and regular requests.
// An AllowedOrigins entry of "*" allows any origin; with credentials enabled the
// request origin is echoed back, since browsers reject "*" alongside credentials.
func CORSMiddleware(cfg config.CorsConfig) gin.HandlerFunc {
	allowAll := false
	allowed := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowed[origin] = true
	}

	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	exposed := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		h := c.Writer.Header()
		h.Add("Vary", "Origin")

		if !allowAll && !allowed[origin] {
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		if allowAll && !cfg.AllowCredentials {
			h.Set("Access-Control-Allow-Origin", "*")
		} else {
			h.Set("Access-Control-Allow-Origin", origin)
		}
		if cfg.AllowCredentials {
			h.Set("Access-Control-Allow-Credentials", "true")
		}
		if exposed != "" {
			h.Set("Access-Control-Expose-Headers", exposed)
		}

		// Preflight request
		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			h.Set("Access-Control-Allow-Methods", methods)
			h.Set("Access-Control-Allow-Headers", headers)
			h.Set("Access-Control-Max-Age", maxAge)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
