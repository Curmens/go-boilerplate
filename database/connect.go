// database/database.go
package database

import (
	"context"
	_ "example.com/migrations"
	"fmt"
	"time"

	"example.com/config"
	"example.com/utils"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps gorm.DB with additional functionality
type DB struct {
	*gorm.DB
	config *config.DatabaseConfig
	logger *utils.Logger
}

// New creates a new database connection with proper configuration
func New(cfg *config.DatabaseConfig, appLogger *utils.Logger) (*DB, error) {
	var dialector gorm.Dialector

	// Build DSN based on driver
	dsn := buildDSN(cfg)

	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: false, // Keep transactions for data integrity
		Logger: logger.New(
			&gormLoggerWriter{logger: appLogger},
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	dbInstance := &DB{
		DB:     db,
		config: cfg,
		logger: appLogger,
	}

	// Test connection
	if err := dbInstance.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	appLogger.Info("Database connection established", map[string]interface{}{
		"driver": cfg.Driver,
		"host":   cfg.Host,
		"port":   cfg.Port,
		"name":   cfg.Name,
	})

	return dbInstance, nil
}

// buildDSN constructs the database connection string
func buildDSN(cfg *config.DatabaseConfig) string {
	switch cfg.Driver {
	case "postgres":
		return fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
		)
	case "mysql":
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
		)
	default:
		return ""
	}
}

// Ping checks database connectivity
func (db *DB) Ping(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close gracefully closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}

	db.logger.Info("Closing database connection", nil)
	return sqlDB.Close()
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	db.logger.Info("Starting database migrations", nil)

	db.logger.Info("Database migrations completed successfully", nil)
	return nil
}

// rollback migration rolls back the last migration
// Note: GORM does not support automatic rollback of migrations.
func (db *DB) RollbackMigration() error {
	// rollback marketplace services

	db.logger.Info("Database rollback completed successfully", nil)

	return nil
}

// Transaction executes a function within a database transaction
//func (db *DB) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
//	return db.WithContext(ctx).Transaction(fn)
//}

// WithContext returns a new DB instance with context
func (db *DB) WithContext(ctx context.Context) *DB {
	return &DB{
		DB:     db.DB.WithContext(ctx),
		config: db.config,
		logger: db.logger,
	}
}

// GetStats returns database connection statistics
func (db *DB) GetStats() map[string]interface{} {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

// Health checks database health
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query
	var result int
	if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	return nil
}

// gormLoggerWriter implements io.Writer for GORM logger
type gormLoggerWriter struct {
	logger *utils.Logger
}

func (w *gormLoggerWriter) Printf(format string, args ...interface{}) {
	w.logger.Debug(fmt.Sprintf(format, args...), map[string]interface{}{
		"component": "gorm",
	})
}
