package middleware

import (
	"crypto/subtle"
	"net/http"

	"example.com/utils"

	"github.com/gin-gonic/gin"
)

// APIKeyMiddleware rejects requests that do not supply the correct X-API-Key header.
// If the configured key is empty the server is misconfigured — all requests are rejected.
func APIKeyMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := c.GetHeader("X-API-Key")
		if key == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(key)) != 1 {
			utils.AbortWithError(c, http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Invalid or missing API key")
			return
		}
		c.Next()
	}
}
