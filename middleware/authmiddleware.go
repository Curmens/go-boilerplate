package middleware

import (
	"strings"

	"example.com/utils"
	"github.com/gin-gonic/gin"
)

// AuthConfig holds authentication middleware configuration
type AuthConfig struct {
	JWTManager    *utils.JWTManager
	SkipPaths     []string
	TokenLookup   string // "header:Authorization" or "cookie:token"
	TokenHeadName string // "Bearer"
}

// AuthMiddleware creates an authentication middleware with JWT validation
func AuthMiddleware(config AuthConfig) gin.HandlerFunc {
	// Set defaults
	if config.TokenLookup == "" {
		config.TokenLookup = "header:Authorization"
	}
	if config.TokenHeadName == "" {
		config.TokenHeadName = "Bearer"
	}

	skipMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		// Check if path should be skipped
		if skipMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Extract token
		token := extractToken(c, config)
		if token == "" {
			utils.Unauthorized(c, "Authentication token is required")
			c.Abort()
			return
		}

		// Validate token
		claims, err := config.JWTManager.ValidateAccessToken(token)
		if err != nil {
			switch err {
			case utils.ErrExpiredToken:
				utils.Unauthorized(c, "Token has expired")
			case utils.ErrInvalidToken, utils.ErrInvalidClaims:
				utils.Unauthorized(c, "Invalid token")
			default:
				utils.Unauthorized(c, "Authentication failed")
			}
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

// extractToken extracts the JWT token from the request
func extractToken(c *gin.Context, config AuthConfig) string {
	parts := strings.Split(config.TokenLookup, ":")
	if len(parts) != 2 {
		return ""
	}

	switch parts[0] {
	case "header":
		return extractTokenFromHeader(c, parts[1], config.TokenHeadName)
	case "cookie":
		return extractTokenFromCookie(c, parts[1])
	default:
		return ""
	}
}

// extractTokenFromHeader extracts token from Authorization header
func extractTokenFromHeader(c *gin.Context, headerName, tokenHead string) string {
	authHeader := c.GetHeader(headerName)
	if authHeader == "" {
		return ""
	}

	// Check for Bearer prefix
	prefix := tokenHead + " "
	if len(authHeader) > len(prefix) && strings.EqualFold(authHeader[:len(prefix)], prefix) {
		return authHeader[len(prefix):]
	}

	return ""
}

// extractTokenFromCookie extracts token from a cookie
func extractTokenFromCookie(c *gin.Context, cookieName string) string {
	token, err := c.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return token
}

// RequireRole creates a middleware that checks for specific roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists {
			utils.Forbidden(c, "Role information not found")
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok {
			utils.Forbidden(c, "Invalid role information")
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		for _, r := range roles {
			if strings.EqualFold(role, r) {
				c.Next()
				return
			}
		}

		utils.Forbidden(c, "Insufficient permissions")
		c.Abort()
	}
}

// RequireAdmin is a shorthand for RequireRole("admin", "super_admin")
func RequireAdmin() gin.HandlerFunc {
	return RequireRole("admin", "super_admin")
}

// OptionalAuth extracts user info if token is present but doesn't require it
func OptionalAuth(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c, config)
		if token == "" {
			c.Next()
			return
		}

		claims, err := config.JWTManager.ValidateAccessToken(token)
		if err == nil {
			c.Set("userID", claims.UserID)
			c.Set("userEmail", claims.Email)
			c.Set("userRole", claims.Role)
			c.Set("claims", claims)
		}

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

// GetUserEmail extracts user email from context
func GetUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("userEmail")
	if !exists {
		return "", false
	}
	e, ok := email.(string)
	return e, ok
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("userRole")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}

// GetClaims extracts full claims from context
func GetClaims(c *gin.Context) (*utils.CustomClaims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}
	cc, ok := claims.(*utils.CustomClaims)
	return cc, ok
}
