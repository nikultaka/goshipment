package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	NATS     NATSConfig
	OAuth    OAuthConfig
	Storage  StorageConfig
	GA4      GA4Config
	SendGrid SendGridConfig
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Name        string
	Env         string
	Port        string
	Version     string
	LogLevel    string
	TrustedProxies []string
	AllowOrigins   []string
	RateLimitRPS   int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration with Sentinel support
type RedisConfig struct {
	Addresses     []string
	MasterName    string
	Password      string
	DB            int
	PoolSize      int
	MinIdleConns  int
	UseSentinel   bool
}

// NATSConfig holds NATS JetStream configuration
type NATSConfig struct {
	URLs     []string
	Token    string
	StreamName string
}

// OAuthConfig holds OAuth2 server configuration
type OAuthConfig struct {
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	JWTSecret          string
	JWTIssuer          string
	JWTAlgorithm       string // "HS256" or "RS256"
	JWTKeyID           string // Key ID for RS256
	JWTPrivateKeyPath  string // Path to RSA private key
	JWTPublicKeyPath   string // Path to RSA public key
}


// StorageConfig holds file storage configuration
type StorageConfig struct {
	Type       string // local, s3
	BasePath   string
	S3Bucket   string
	S3Region   string
	S3AccessKey string
	S3SecretKey string
	MaxFileSize int64
}

// GA4Config holds Google Analytics 4 configuration
type GA4Config struct {
	MeasurementID string
	APISecret     string
	Enabled       bool
}

// SendGridConfig holds SendGrid email configuration
type SendGridConfig struct {
	APIKey       string
	FromEmail    string
	FromName     string
	ReplyToEmail string
	Enabled      bool
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (not in production)
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "Go API System"),
			Env:         getEnv("APP_ENV", "development"),
			Port:        getEnv("APP_PORT", "8080"),
			Version:     getEnv("APP_VERSION", "v1"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
			TrustedProxies: getEnvSlice("TRUSTED_PROXIES", []string{"127.0.0.1"}),
			AllowOrigins:   getEnvSlice("ALLOW_ORIGINS", []string{"http://localhost:3000"}),
			RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 100),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "goapi"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 5)) * time.Minute,
		},
		Redis: RedisConfig{
			Addresses:    getEnvSlice("REDIS_ADDRESSES", []string{"localhost:6379"}),
			MasterName:   getEnv("REDIS_MASTER_NAME", "mymaster"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
			UseSentinel:  getEnvBool("REDIS_USE_SENTINEL", false),
		},
		NATS: NATSConfig{
			URLs:       getEnvSlice("NATS_URLS", []string{"nats://localhost:4222"}),
			Token:      getEnv("NATS_TOKEN", ""),
			StreamName: getEnv("NATS_STREAM_NAME", "NOTIFICATIONS"),
		},
		OAuth: OAuthConfig{
			AccessTokenExpiry:  time.Duration(getEnvInt("OAUTH_ACCESS_TOKEN_EXPIRY", 3600)) * time.Second,
			RefreshTokenExpiry: time.Duration(getEnvInt("OAUTH_REFRESH_TOKEN_EXPIRY", 2592000)) * time.Second,
			JWTSecret:          getEnv("JWT_SECRET", ""),
			JWTIssuer:          getEnv("JWT_ISSUER", "goapi"),
			JWTAlgorithm:       getEnv("JWT_ALGORITHM", "HS256"),
			JWTKeyID:           getEnv("JWT_KID", "default-key-2025"),
			JWTPrivateKeyPath:  getEnv("JWT_PRIVATE_KEY_PATH", ""),
			JWTPublicKeyPath:   getEnv("JWT_PUBLIC_KEY_PATH", ""),
		},
		Storage: StorageConfig{
			Type:        getEnv("STORAGE_TYPE", "local"),
			BasePath:    getEnv("STORAGE_BASE_PATH", "./uploads"),
			S3Bucket:    getEnv("S3_BUCKET", ""),
			S3Region:    getEnv("S3_REGION", "us-east-1"),
			S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
			S3SecretKey: getEnv("S3_SECRET_KEY", ""),
			MaxFileSize: int64(getEnvInt("MAX_FILE_SIZE", 10485760)), // 10MB default
		},
		GA4: GA4Config{
			MeasurementID: getEnv("GA4_MEASUREMENT_ID", ""),
			APISecret:     getEnv("GA4_API_SECRET", ""),
			Enabled:       getEnvBool("GA4_ENABLED", false),
		},
		SendGrid: SendGridConfig{
			APIKey:       getEnv("SENDGRID_API_KEY", ""),
			FromEmail:    getEnv("SENDGRID_FROM_EMAIL", "noreply@goshipment.com"),
			FromName:     getEnv("SENDGRID_FROM_NAME", "GoShipment"),
			ReplyToEmail: getEnv("SENDGRID_REPLY_TO_EMAIL", "support@goshipment.com"),
			Enabled:      getEnvBool("SENDGRID_ENABLED", false),
		},
	}

	// Validate critical configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if critical configuration values are set
func (c *Config) Validate() error {
	if c.App.Env == "production" {
		if c.OAuth.JWTSecret == "" {
			return fmt.Errorf("JWT_SECRET is required in production")
		}
		if c.Database.Password == "" {
			return fmt.Errorf("DB_PASSWORD is required in production")
		}
	}
	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// Helper functions

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

func getEnvSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		// Simple comma-separated parsing
		result := []string{}
		for _, v := range splitString(val, ",") {
			if trimmed := trimSpace(v); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultVal
}

func splitString(s, sep string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if string(c) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
