package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"hearth/internal/models"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Host string
	Port int

	// Public URL for OAuth redirects, etc.
	PublicURL string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Storage
	StorageBackend   string // local, s3
	StorageEndpoint  string
	StorageBucket    string
	StorageAccessKey string
	StorageSecretKey string
	StorageRegion    string
	LocalStoragePath string

	// Auth
	SecretKey     string
	TokenExpiry   time.Duration
	RefreshExpiry time.Duration
	AuthProvider  string // native, fusionauth

	// FusionAuth
	FusionAuthHost          string
	FusionAuthApplicationID string
	FusionAuthClientID      string
	FusionAuthClientSecret  string
	FusionAuthAPIKey        string

	// Registration
	RegistrationEnabled bool
	InviteOnly          bool

	// Quotas
	Quotas *models.QuotaConfig

	// Logging
	LogLevel  string
	LogFormat string
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		// Server
		Host: getEnv("HOST", "0.0.0.0"),
		Port: getEnvInt("PORT", 8080),

		PublicURL: getEnv("PUBLIC_URL", "http://localhost:8080"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://hearth:hearth@localhost:5432/hearth?sslmode=disable"),

		// Redis
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		// Storage
		StorageBackend:   getEnv("STORAGE_BACKEND", "local"),
		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", ""),
		StorageBucket:    getEnv("STORAGE_BUCKET", "hearth"),
		StorageAccessKey: getEnv("STORAGE_ACCESS_KEY", ""),
		StorageSecretKey: getEnv("STORAGE_SECRET_KEY", ""),
		StorageRegion:    getEnv("STORAGE_REGION", "us-east-1"),
		LocalStoragePath: getEnv("LOCAL_STORAGE_PATH", "./data/uploads"),

		// Auth
		SecretKey:     getEnv("SECRET_KEY", "change-me-in-production"),
		TokenExpiry:   getEnvDuration("TOKEN_EXPIRY", 1*time.Hour),
		RefreshExpiry: getEnvDuration("REFRESH_EXPIRY", 30*24*time.Hour),
		AuthProvider:  getEnv("AUTH_PROVIDER", "native"),

		// FusionAuth
		FusionAuthHost:          getEnv("FUSIONAUTH_HOST", ""),
		FusionAuthApplicationID: getEnv("FUSIONAUTH_APPLICATION_ID", ""),
		FusionAuthClientID:      getEnv("FUSIONAUTH_CLIENT_ID", ""),
		FusionAuthClientSecret:  getEnv("FUSIONAUTH_CLIENT_SECRET", ""),
		FusionAuthAPIKey:        getEnv("FUSIONAUTH_API_KEY", ""),

		// Registration
		RegistrationEnabled: getEnvBool("REGISTRATION_ENABLED", true),
		InviteOnly:          getEnvBool("INVITE_ONLY", false),

		// Quotas
		Quotas: loadQuotaConfig(),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}

	return cfg
}

func loadQuotaConfig() *models.QuotaConfig {
	// Start with defaults
	cfg := models.DefaultQuotaConfig()

	// Override from environment
	if v := getEnvInt("QUOTA_USER_STORAGE_MB", 0); v != 0 {
		cfg.Storage.UserStorageMB = int64(v)
	}
	if v := getEnvInt("QUOTA_SERVER_STORAGE_MB", 0); v != 0 {
		cfg.Storage.ServerStorageMB = int64(v)
	}
	if v := getEnvInt("QUOTA_MAX_FILE_SIZE_MB", 0); v != 0 {
		cfg.Storage.MaxFileSizeMB = int64(v)
	}
	if v := getEnvInt("QUOTA_MESSAGE_RATE_LIMIT", 0); v != 0 {
		cfg.Messages.RateLimitMessages = v
	}
	if v := getEnvInt("QUOTA_MAX_SERVERS_OWNED", 0); v != 0 {
		cfg.Servers.MaxServersOwned = v
	}

	// Check for unlimited mode
	if getEnvBool("QUOTAS_UNLIMITED", false) {
		cfg = models.UnlimitedQuotaConfig()
	}

	return cfg
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		value = strings.ToLower(value)
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
