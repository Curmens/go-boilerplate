package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	App      AppConfig      `json:"app"`
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Cache    CacheConfig    `json:"cache"`
	Redis    RedisConfig    `json:"redis"`
	JWT      JWTConfig      `json:"jwt"`
	Email    EmailConfig    `json:"email"`
	Logger   LoggerConfig   `json:"logger"`
	Storage  StorageConfig  `json:"storage"`
	Rate     RateConfig     `json:"rate"`
	Cors     CorsConfig     `json:"cors"`
}

type AppConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerConfig struct {
	Host            string        `json:"host"`
	Port            string        `json:"port"`
	Mode            string        `json:"mode"` // debug, release, test
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	MaxBodyBytes    int64         `json:"max_body_bytes"`
	TLS             TLSConfig     `json:"tls"`
	// TrustedProxies lists load balancer / reverse-proxy IPs or CIDRs allowed to
	// set X-Forwarded-For. Empty means trust no proxies (safest default).
	TrustedProxies []string `json:"trusted_proxies"`
	// InternalAPIKey protects service-to-service endpoints via the X-API-Key header.
	InternalAPIKey string `json:"-"`
}

type TLSConfig struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

type DatabaseConfig struct {
	// Enabled lets the app boot without a database (useful for a fresh clone).
	Enabled         bool          `json:"enabled"`
	AutoMigrate     bool          `json:"auto_migrate"`
	Driver          string        `json:"driver"`
	Host            string        `json:"host"`
	Port            string        `json:"port"`
	Name            string        `json:"name"`
	User            string        `json:"user"`
	Password        string        `json:"-"`
	SSLMode         string        `json:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
	SlowThreshold   time.Duration `json:"slow_threshold"`
}

type CacheConfig struct {
	UseRedis   bool          `json:"use_redis"`
	DefaultTTL time.Duration `json:"default_ttl"`
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"-"`
	DB       int    `json:"db"`
}

type JWTConfig struct {
	Secret          string        `json:"-"`
	AccessTokenTTL  time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl"`
	Issuer          string        `json:"issuer"`
}

type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"-"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
}

type LoggerConfig struct {
	Level         string `json:"level"`
	Format        string `json:"format"` // json, text
	FilePath      string `json:"file_path"`
	MaxSize       int    `json:"max_size"` // MB
	MaxFiles      int    `json:"max_files"`
	EnableConsole bool   `json:"enable_console"`
}

type StorageConfig struct {
	Driver    string    `json:"driver"` // local, s3, gcs
	LocalPath string    `json:"local_path"`
	S3        S3Config  `json:"s3"`
	GCS       GCSConfig `json:"gcs"`
}

type S3Config struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"-"`
	SecretAccessKey string `json:"-"`
	Endpoint        string `json:"endpoint"` // For S3-compatible services
}

type GCSConfig struct {
	ProjectID       string `json:"project_id"`
	Bucket          string `json:"bucket"`
	CredentialsPath string `json:"credentials_path"`
}

// RateConfig configures the global per-IP fixed-window rate limiter.
type RateConfig struct {
	Enabled bool          `json:"enabled"`
	Limit   int           `json:"limit"`  // Requests allowed per window
	Window  time.Duration `json:"window"` // Window length
}

type CorsConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// Load loads configuration from environment variables and an env file.
// ENV_FILE selects which file to read (e.g. ENV_FILE=.prod.env); it defaults
// to .env. Variables already present in the environment always win, so the
// file never overrides an explicit export.
func Load() (*Config, error) {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}

	if err := godotenv.Load(envFile); err != nil {
		// The env file is optional.
		fmt.Printf("Warning: %s not found: %v\n", envFile, err)
	}

	config := &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "go-boilerplate"),
			Version: getEnv("APP_VERSION", "0.1.0"),
		},
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnv("SERVER_PORT", getEnv("PORT", "8080")),
			Mode:            getEnv("GIN_MODE", "release"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getDurationEnv("SERVER_IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),
			MaxBodyBytes:    int64(getIntEnv("SERVER_MAX_BODY_BYTES", 4*1024*1024)),
			TLS: TLSConfig{
				Enabled:  getBoolEnv("TLS_ENABLED", false),
				CertFile: getEnv("TLS_CERT_FILE", ""),
				KeyFile:  getEnv("TLS_KEY_FILE", ""),
			},
			TrustedProxies: getSliceEnv("TRUSTED_PROXIES", []string{}),
			InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
		},
		Database: DatabaseConfig{
			Enabled:         getBoolEnv("DB_ENABLED", true),
			AutoMigrate:     getBoolEnv("DB_AUTO_MIGRATE", false),
			Driver:          getEnv("DB_DRIVER", "postgres"),
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "app"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getDurationEnv("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
			SlowThreshold:   getDurationEnv("DB_SLOW_THRESHOLD", 200*time.Millisecond),
		},
		Cache: CacheConfig{
			UseRedis:   getBoolEnv("USE_REDIS", false),
			DefaultTTL: getDurationEnv("CACHE_DEFAULT_TTL", 60*time.Second),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", ""),
			AccessTokenTTL:  getDurationEnv("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getDurationEnv("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
			Issuer:          getEnv("JWT_ISSUER", "go-boilerplate"),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", ""),
			SMTPPort:     getIntEnv("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", ""),
			FromName:     getEnv("FROM_NAME", "App"),
		},
		Logger: LoggerConfig{
			Level:         getEnv("LOG_LEVEL", "info"),
			Format:        getEnv("LOG_FORMAT", "json"),
			FilePath:      getEnv("LOG_FILE_PATH", "./logs"),
			MaxSize:       getIntEnv("LOG_MAX_SIZE", 100),
			MaxFiles:      getIntEnv("LOG_MAX_FILES", 7),
			EnableConsole: getBoolEnv("LOG_ENABLE_CONSOLE", true),
		},
		Storage: StorageConfig{
			Driver:    getEnv("STORAGE_DRIVER", "local"),
			LocalPath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),
			S3: S3Config{
				Region:          getEnv("AWS_REGION", ""),
				Bucket:          getEnv("AWS_S3_BUCKET", ""),
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
				Endpoint:        getEnv("AWS_S3_ENDPOINT", ""),
			},
			GCS: GCSConfig{
				ProjectID:       getEnv("GCS_PROJECT_ID", ""),
				Bucket:          getEnv("GCS_BUCKET", ""),
				CredentialsPath: getEnv("GCS_CREDENTIALS_PATH", ""),
			},
		},
		Rate: RateConfig{
			Enabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
			Limit:   getIntEnv("RATE_LIMIT_LIMIT", 300),
			Window:  getDurationEnv("RATE_LIMIT_WINDOW", time.Minute),
		},
		Cors: CorsConfig{
			AllowedOrigins:   getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
			AllowedMethods:   getSliceEnv("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getSliceEnv("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Authorization", "X-Request-ID", "X-API-Key"}),
			ExposedHeaders:   getSliceEnv("CORS_EXPOSED_HEADERS", []string{"X-Request-ID"}),
			AllowCredentials: getBoolEnv("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           getIntEnv("CORS_MAX_AGE", 86400),
		},
	}

	return config, config.Validate()
}

// IsDebug reports whether the server runs in gin debug mode.
func (c *Config) IsDebug() bool {
	return c.Server.Mode == "debug"
}

// Validate validates the configuration. Debug mode relaxes the secret checks
// so a fresh clone can boot without any configuration.
func (c *Config) Validate() error {
	switch c.Server.Mode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("GIN_MODE must be one of debug, release, test (got %q)", c.Server.Mode)
	}

	if c.IsDebug() {
		return nil
	}

	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be set and at least 32 characters outside debug mode")
	}

	if c.Database.Enabled && c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD must be set outside debug mode")
	}

	if c.Server.TLS.Enabled && (c.Server.TLS.CertFile == "" || c.Server.TLS.KeyFile == "") {
		return fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE must be set when TLS_ENABLED=true")
	}

	return nil
}

// Address returns the host:port the HTTP server listens on.
func (s *ServerConfig) Address() string {
	return s.Host + ":" + s.Port
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
