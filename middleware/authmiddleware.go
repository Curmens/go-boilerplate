package middleware

import (
	"errors"
	"net/http"
	"strings"

	"example.com/utils"

	"github.com/gin-gonic/gin"
)

// Context keys set by AuthMiddleware
const (
	ClaimsKey = "claims"
	UserIDKey = "userID"
)

// AuthCookieName is the cookie checked when no Authorization header is sent
const AuthCookieName = "token"

// AuthMiddleware validates the access token (Bearer header first, then cookie)
// and stores its claims in the context.
func AuthMiddleware(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			utils.AbortWithError(c, http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Authorization token required")
			return
		}

		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			message := "Invalid token"
			if errors.Is(err, utils.ErrExpiredToken) {
				message = "Token has expired"
			}
			utils.AbortWithError(c, http.StatusUnauthorized, utils.ErrCodeUnauthorized, message)
			return
		}

		c.Set(ClaimsKey, claims)
		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

// OptionalAuthMiddleware sets claims when a valid token is present but never rejects
func OptionalAuthMiddleware(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token := extractToken(c); token != "" {
			if claims, err := jwtManager.ValidateAccessToken(token); err == nil {
				c.Set(ClaimsKey, claims)
				c.Set(UserIDKey, claims.UserID)
			}
		}
		c.Next()
	}
}

// RoleMiddleware allows the request when the token's role is one of roles.
// Must run after AuthMiddleware.
func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := GetClaims(c)
		if !ok {
			utils.AbortWithError(c, http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Authentication required")
			return
		}

		for _, role := range roles {
			if claims.Role == role {
				c.Next()
				return
			}
		}

		utils.AbortWithError(c, http.StatusForbidden, utils.ErrCodeForbidden, "Insufficient permissions")
	}
}

// GetClaims returns the authenticated token claims, if any
func GetClaims(c *gin.Context) (*utils.CustomClaims, bool) {
	value, exists := c.Get(ClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := value.(*utils.CustomClaims)
	return claims, ok
}

// extractToken reads a Bearer token from the Authorization header, falling back to the auth cookie
func extractToken(c *gin.Context) string {
	if parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}

	if token, err := c.Cookie(AuthCookieName); err == nil {
		return token
	}
	return ""
}

// SetAuthCookie sets the HttpOnly authentication cookie
func SetAuthCookie(c *gin.Context, token string, maxAge int, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AuthCookieName, token, maxAge, "/", "", secure, true)
}

// ClearAuthCookie clears the authentication cookie
func ClearAuthCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AuthCookieName, "", -1, "/", "", secure, true)
}
