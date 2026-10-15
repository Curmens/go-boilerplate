package database

import (
	"context"
	"fmt"
	"time"

	"example.com/config"
	"example.com/migrations"
	"example.com/utils"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps gorm.DB with lifecycle, health and migration helpers
type DB struct {
	*gorm.DB
	config *config.DatabaseConfig
	logger *utils.Logger
}

// New opens a database connection, configures the pool and verifies connectivity
func New(cfg *config.DatabaseConfig, appLogger *utils.Logger) (*DB, error) {
	var dialector gorm.Dialector

	dsn := buildDSN(cfg)
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	gormConfig := &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true, // Skip implicit per-statement transactions; use explicit db.Transaction() where needed
		Logger: logger.New(
			&gormLoggerWriter{logger: appLogger},
			logger.Config{
				SlowThreshold:             cfg.SlowThreshold,
				LogLevel:                  logger.Warn,
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

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	dbInstance := &DB{
		DB:     db,
		config: cfg,
		logger: appLogger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dbInstance.Ping(ctx); err != nil {
		_ = sqlDB.Close()
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

// buildDSN constructs the database connection string. Sessions run in UTC.
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

// Migrate applies all pending migrations registered in migrations.All()
func (db *DB) Migrate() error {
	db.logger.Info("Starting database migrations", nil)

	err := migrations.Apply(db.DB, func(name string) {
		db.logger.Info("Migration applied", map[string]interface{}{"migration": name})
	})
	if err != nil {
		db.logger.Error("Migration failed", map[string]interface{}{"error": err.Error()})
		return err
	}

	db.logger.Info("Database migrations completed successfully", nil)
	return nil
}

// Health runs a ping and a trivial query with a timeout
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	var result int
	if err := db.DB.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	return nil
}

// GetStats returns database connection pool statistics
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

// gormLoggerWriter routes GORM log output through the application logger
type gormLoggerWriter struct {
	logger *utils.Logger
}

func (w *gormLoggerWriter) Printf(format string, args ...interface{}) {
	_ = w.logger.Warn(fmt.Sprintf(format, args...), map[string]interface{}{
		"component": "gorm",
	})
}
