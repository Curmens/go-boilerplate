package routes

import (
	"example.com/controllers"
	"example.com/middleware"
	"example.com/utils"

	"github.com/gin-gonic/gin"
)

// RouterConfig holds dependencies for route setup
type RouterConfig struct {
	Logger     *utils.Logger
	JWTManager *utils.JWTManager
}

// SetupRouter creates and configures the Gin router
func SetupRouter(config RouterConfig) *gin.Engine {
	// Create Gin router
	r := gin.New()

	// Add middleware in order of execution
	r.Use(middleware.RequestIDMiddleware())

	// CORS middleware should be early in the chain
	r.Use(middleware.CORSMiddlewareCustom(middleware.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With", "X-CSRF-Token", "X-Request-ID"},
	}))

	// Recovery middleware should be early in the chain
	r.Use(middleware.RecoveryMiddleware(config.Logger))

	// Main logging middleware
	r.Use(middleware.LoggerMiddleware(middleware.MiddlewareConfig{
		Logger:            config.Logger,
		SkipPaths:         []string{"/health", "/healthz", "/ready", "/metrics", "/ping"},
		EnableBodyLogging: gin.Mode() == gin.DebugMode,
		MaxBodySize:       32 * 1024, // 32KB
	}))

	// Error logging middleware
	r.Use(middleware.ErrorLoggingMiddleware(config.Logger))

	// Body logging middleware for development
	if gin.Mode() == gin.DebugMode {
		r.Use(middleware.BodyLoggingMiddleware(config.Logger, 1024)) // 1KB limit for dev
	}

	// Setup routes
	setupHealthRoutes(r)
	setupPublicRoutes(r, config)
	setupAuthenticatedRoutes(r, config)

	return r
}

// setupHealthRoutes configures health check endpoints
func setupHealthRoutes(router *gin.Engine) {
	// Health check endpoints
	router.GET("/health", controllers.HealthHandler())
	router.GET("/healthz", controllers.LivenessHandler()) // Kubernetes liveness
	router.GET("/ready", controllers.ReadinessHandler())  // Kubernetes readiness
	router.GET("/metrics", controllers.MetricsHandler())
	router.GET("/ping", controllers.Ping)
}

// setupPublicRoutes configures public routes that don't require authentication
func setupPublicRoutes(router *gin.Engine, config RouterConfig) {
	// API v1 group
	api := router.Group("/api/v1")
	{
		// Authentication routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/login", loginHandler())
			auth.POST("/register", registerHandler())
			auth.POST("/refresh", refreshHandler())
			auth.POST("/forgot-password", forgotPasswordHandler())
			auth.POST("/reset-password", resetPasswordHandler())
			auth.POST("/verify-email", verifyEmailHandler())

			// OAuth routes
			oauth := auth.Group("/oauth")
			{
				google := oauth.Group("/google")
				{
					google.GET("/login", googleLoginHandler())
					google.GET("/callback", googleCallbackHandler())
				}
				apple := oauth.Group("/apple")
				{
					apple.GET("/login", appleLoginHandler())
					apple.POST("/callback", appleCallbackHandler())
				}
			}
		}
	}
}

// setupAuthenticatedRoutes configures routes that require authentication
func setupAuthenticatedRoutes(router *gin.Engine, config RouterConfig) {
	// API v1 authenticated group
	api := router.Group("/api/v1")

	// Apply auth middleware
	authConfig := middleware.AuthConfig{
		JWTManager: config.JWTManager,
		SkipPaths:  []string{},
	}
	api.Use(middleware.AuthMiddleware(authConfig))

	{
		// User routes
		users := api.Group("/users")
		{
			users.GET("/me", getMeHandler())
			users.PUT("/me", updateMeHandler())
			users.DELETE("/me", deleteMeHandler())
			users.PUT("/me/password", changePasswordHandler())
		}

		// Admin routes
		admin := api.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			// User management
			admin.GET("/users", listUsersHandler())
			admin.GET("/users/:id", getUserHandler())
			admin.POST("/users", createUserHandler())
			admin.PUT("/users/:id", updateUserHandler())
			admin.DELETE("/users/:id", deleteUserHandler())

			// Role management
			admin.GET("/roles", listRolesHandler())
			admin.POST("/roles", createRoleHandler())
			admin.PUT("/roles/:id", updateRoleHandler())
			admin.DELETE("/roles/:id", deleteRoleHandler())
		}
	}
}

// Placeholder handlers - these should be implemented in controllers
func loginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "login not implemented"})
	}
}

func registerHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "register not implemented"})
	}
}

func refreshHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "refresh not implemented"})
	}
}

func forgotPasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "forgot-password not implemented"})
	}
}

func resetPasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "reset-password not implemented"})
	}
}

func verifyEmailHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "verify-email not implemented"})
	}
}

func googleLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "google oauth login not implemented"})
	}
}

func googleCallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "google oauth callback not implemented"})
	}
}

func appleLoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "apple oauth login not implemented"})
	}
}

func appleCallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "apple oauth callback not implemented"})
	}
}

func getMeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := middleware.GetUserID(c)
		utils.Success(c, gin.H{"user_id": userID, "message": "get me not implemented"})
	}
}

func updateMeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "update me not implemented"})
	}
}

func deleteMeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "delete me not implemented"})
	}
}

func changePasswordHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "change password not implemented"})
	}
}

func listUsersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "list users not implemented"})
	}
}

func getUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "get user not implemented"})
	}
}

func createUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "create user not implemented"})
	}
}

func updateUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "update user not implemented"})
	}
}

func deleteUserHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "delete user not implemented"})
	}
}

func listRolesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "list roles not implemented"})
	}
}

func createRoleHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "create role not implemented"})
	}
}

func updateRoleHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "update role not implemented"})
	}
}

func deleteRoleHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.Success(c, gin.H{"message": "delete role not implemented"})
	}
}
