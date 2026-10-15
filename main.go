package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"example.com/cache"
	"example.com/config"
	"example.com/database"
	"example.com/routes"
	"example.com/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Apply Gin mode after the env file is loaded
	gin.SetMode(cfg.Server.Mode)

	appLogger, err := utils.NewLogger(cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer appLogger.Close()

	var db *database.DB
	if cfg.Database.Enabled {
		db, err = database.New(&cfg.Database, appLogger)
		if err != nil {
			return err
		}
		defer db.Close()

		if cfg.Database.AutoMigrate {
			if err := db.Migrate(); err != nil {
				return err
			}
		}
	} else {
		_ = appLogger.Warn("Database disabled (DB_ENABLED=false)", nil)
	}

	cacheStore, err := cache.NewStore(cfg, appLogger)
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	defer cacheStore.Close()

	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" && cfg.IsDebug() {
		// Config validation only allows this in debug mode. Tokens won't survive restarts.
		jwtSecret = randomHex(32)
		_ = appLogger.Warn("JWT_SECRET not set; using an ephemeral secret for this debug session", nil)
	}
	jwtManager, err := utils.NewJWTManager(utils.JWTConfig{
		Secret:          jwtSecret,
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
		Issuer:          cfg.JWT.Issuer,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize JWT manager: %w", err)
	}

	router, err := routes.SetupRouter(routes.RouterConfig{
		Config:     cfg,
		Logger:     appLogger,
		DB:         db,
		Cache:      cacheStore,
		JWTManager: jwtManager,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Server.Address(),
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		_ = appLogger.Info("Starting server", map[string]interface{}{
			"address": srv.Addr,
			"mode":    gin.Mode(),
			"tls":     cfg.Server.TLS.Enabled,
			"version": cfg.App.Version,
		})

		var err error
		if cfg.Server.TLS.Enabled {
			err = srv.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
	case <-ctx.Done():
	}

	_ = appLogger.Info("Shutting down server...", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	_ = appLogger.Info("Server stopped", nil)
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
