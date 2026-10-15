package routes

import (
	"fmt"
	"net/http"

	"example.com/cache"
	"example.com/config"
	"example.com/controllers"
	"example.com/database"
	"example.com/middleware"
	"example.com/utils"
	"example.com/utils/metrics"

	"github.com/gin-gonic/gin"
)

// RouterConfig holds dependencies for route setup
type RouterConfig struct {
	Config     *config.Config
	Logger     *utils.Logger
	DB         *database.DB // nil when the database is disabled
	Cache      cache.Store
	JWTManager *utils.JWTManager
}

// SetupRouter creates and configures the Gin router
func SetupRouter(deps RouterConfig) (*gin.Engine, error) {
	cfg := deps.Config
	r := gin.New()

	// Only these proxies may set X-Forwarded-For; empty means ClientIP() is the direct peer.
	if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		return nil, fmt.Errorf("invalid TRUSTED_PROXIES: %w", err)
	}

	utils.InitValidator()

	// Order matters: metrics first for accurate timings, request ID before anything that logs.
	r.Use(metrics.MetricsMiddleware())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware(deps.Logger))
	r.Use(middleware.LoggerMiddleware(middleware.MiddlewareConfig{
		Logger:    deps.Logger,
		SkipPaths: []string{"/health", "/livez", "/readyz", "/metrics", "/metrics/prometheus", "/ping"},
	}))
	r.Use(middleware.CORSMiddleware(cfg.Cors))
	r.Use(middleware.CacheTraceMiddleware())
	r.Use(middleware.ErrorLoggingMiddleware(deps.Logger))
	r.Use(middleware.MaxBodySizeMiddleware(cfg.Server.MaxBodyBytes))
	if cfg.Rate.Enabled {
		r.Use(middleware.RateLimitMiddleware(cfg.Rate.Limit, cfg.Rate.Window))
	}
	if gin.Mode() == gin.DebugMode {
		r.Use(middleware.BodyLoggingMiddleware(deps.Logger, 32*1024))
	}

	r.NoRoute(func(c *gin.Context) {
		utils.NotFound(c, "Route not found")
	})
	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		utils.Error(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	})

	setupSystemRoutes(r, deps)
	setupAPIRoutes(r, deps)

	return r, nil
}

// setupSystemRoutes registers health, metrics and documentation endpoints
func setupSystemRoutes(r *gin.Engine, deps RouterConfig) {
	health := &controllers.HealthController{DB: deps.DB, Version: deps.Config.App.Version}

	r.GET("/ping", controllers.Ping)
	r.GET("/health", health.Health)
	r.GET("/livez", health.Liveness)
	r.GET("/readyz", health.Readiness)

	// Internal/dev-only endpoints — not registered in release mode.
	// In production, expose Prometheus metrics to your scraper via network-level
	// controls (firewall, internal load balancer) instead.
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/metrics", health.Metrics)
		r.GET("/metrics/prometheus", metrics.PrometheusHandler())
		r.GET("/openapi.yaml", controllers.OpenAPISpecHandler())
		r.GET("/docs", controllers.SwaggerUIHandler(deps.Config.App.Name, "/openapi.yaml"))
	}

	// Service-to-service endpoints guarded by X-API-Key
	internal := r.Group("/internal", middleware.APIKeyMiddleware(deps.Config.Server.InternalAPIKey))
	internal.GET("/metrics/prometheus", metrics.PrometheusHandler())
}

// setupAPIRoutes registers versioned application routes
func setupAPIRoutes(r *gin.Engine, deps RouterConfig) {
	api := r.Group("/api/v1")

	// Public routes go on api directly. Give sensitive endpoints (login,
	// register, ...) a tighter limit than the global one:
	//   auth := api.Group("/auth", middleware.RateLimitMiddleware(10, time.Minute))

	// Authenticated routes
	protected := api.Group("", middleware.AuthMiddleware(deps.JWTManager))
	protected.GET("/me", controllers.Me)

	// Role-restricted routes:
	//   admin := protected.Group("/admin", middleware.RoleMiddleware("admin"))
}
