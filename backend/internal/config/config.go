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

	// Rate Limiting
	RateLimitEnabled bool
	RateLimitMax     int           // Maximum requests per window
	RateLimitWindow  time.Duration // Time window for rate limiting

	// Bcrypt Worker Pool
	BcryptPoolWorkers int           // Number of concurrent bcrypt workers (default: NumCPU)
	BcryptPoolQueue   int           // Max pending jobs (default: Workers * 10)
	BcryptPoolTimeout time.Duration // Default timeout for bcrypt operations

	// Graceful Shutdown
	DrainTimeout     time.Duration // Time to wait for connections to drain before forced shutdown
	DrainGracePeriod time.Duration // Time between reconnect signal and closing connections

	// Quotas
	Quotas *models.QuotaConfig

	// Logging
	LogLevel  string
	LogFormat string

	// LiveKit (Voice/Video)
	LiveKitAPIKey    string
	LiveKitAPISecret string
	LiveKitURL       string

	// OAuth Configuration
	OAuthGitHubClientID      string
	OAuthGitHubClientSecret  string
	OAuthGoogleClientID      string
	OAuthGoogleClientSecret  string
	OAuthDiscordClientID     string
	OAuthDiscordClientSecret string
	OAuthRedirectBase        string // Base URL for OAuth redirects (e.g., https://app.example.com)

	// AI Provider Configuration
	AIEncryptionKey     string // Key for encrypting API keys at rest
	AIDefaultProvider   string // Default provider type (openai, anthropic, ollama, etc.)
	AIDefaultModel      string // Default model ID
	AIOpenAIKey         string // Default OpenAI API key
	AIAnthropicKey      string // Default Anthropic API key
	AIOpenRouterKey     string // Default OpenRouter API key
	AIBedrockRegion     string // AWS Bedrock region
	AIBedrockAccessKey  string // AWS access key for Bedrock
	AIBedrockSecretKey  string // AWS secret key for Bedrock
	AIOllamaURL         string // Ollama server URL
	AIAllowUserOverride bool   // Allow users to use their own API keys

	// Stripe Configuration (Premium/Billing)
	StripeSecretKey     string // Stripe secret key
	StripeWebhookSecret string // Stripe webhook signing secret
	IsProduction        bool   // Whether running in production
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
		SecretKey:     getRequiredEnv("SECRET_KEY"), // No default - must be set for security
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

		// Rate Limiting (enabled by default, disable for testing with RATE_LIMIT_ENABLED=false)
		RateLimitEnabled: getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitMax:     getEnvInt("RATE_LIMIT_MAX", 10000),
		RateLimitWindow:  getEnvDuration("RATE_LIMIT_WINDOW", 60*time.Second),

		// Bcrypt Worker Pool (bounds concurrent CPU-intensive password operations)
		BcryptPoolWorkers: getEnvInt("BCRYPT_POOL_WORKERS", 0), // 0 = runtime.NumCPU()
		BcryptPoolQueue:   getEnvInt("BCRYPT_POOL_QUEUE", 0),   // 0 = Workers * 10
		BcryptPoolTimeout: getEnvDuration("BCRYPT_POOL_TIMEOUT", 5*time.Second),

		// Graceful Shutdown (connection draining for zero-downtime deploys)
		DrainTimeout:     getEnvDuration("DRAIN_TIMEOUT", 30*time.Second),     // Max time to wait for connections to drain
		DrainGracePeriod: getEnvDuration("DRAIN_GRACE_PERIOD", 5*time.Second), // Time between reconnect signal and forced close

		// Quotas
		Quotas: loadQuotaConfig(),

		// Logging
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),

		// LiveKit (Voice/Video)
		LiveKitAPIKey:    getEnv("LIVEKIT_API_KEY", ""),
		LiveKitAPISecret: getEnv("LIVEKIT_API_SECRET", ""),
		LiveKitURL:       getEnv("LIVEKIT_URL", ""),

		// OAuth (Social Login)
		OAuthGitHubClientID:      getEnv("OAUTH_GITHUB_CLIENT_ID", ""),
		OAuthGitHubClientSecret:  getEnv("OAUTH_GITHUB_CLIENT_SECRET", ""),
		OAuthGoogleClientID:      getEnv("OAUTH_GOOGLE_CLIENT_ID", ""),
		OAuthGoogleClientSecret:  getEnv("OAUTH_GOOGLE_CLIENT_SECRET", ""),
		OAuthDiscordClientID:     getEnv("OAUTH_DISCORD_CLIENT_ID", ""),
		OAuthDiscordClientSecret: getEnv("OAUTH_DISCORD_CLIENT_SECRET", ""),
		OAuthRedirectBase:        getEnv("OAUTH_REDIRECT_BASE", ""),

		// Stripe (Premium & Billing)
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		IsProduction:        getEnvBool("PRODUCTION", false),

		// AI Provider Configuration
		AIEncryptionKey:     getEnv("AI_ENCRYPTION_KEY", ""),   // Falls back to SECRET_KEY if empty
		AIDefaultProvider:   getEnv("AI_DEFAULT_PROVIDER", ""), // e.g., "openai", "anthropic", "ollama"
		AIDefaultModel:      getEnv("AI_DEFAULT_MODEL", ""),    // e.g., "gpt-4-turbo", "claude-3-opus"
		AIOpenAIKey:         getEnv("AI_OPENAI_KEY", ""),
		AIAnthropicKey:      getEnv("AI_ANTHROPIC_KEY", ""),
		AIOpenRouterKey:     getEnv("AI_OPENROUTER_KEY", ""),
		AIBedrockRegion:     getEnv("AI_BEDROCK_REGION", "us-east-1"),
		AIBedrockAccessKey:  getEnv("AI_BEDROCK_ACCESS_KEY", ""),
		AIBedrockSecretKey:  getEnv("AI_BEDROCK_SECRET_KEY", ""),
		AIOllamaURL:         getEnv("AI_OLLAMA_URL", "http://localhost:11434"),
		AIAllowUserOverride: getEnvBool("AI_ALLOW_USER_OVERRIDE", true),
	}

	// Use SECRET_KEY as fallback for AI encryption
	if cfg.AIEncryptionKey == "" {
		cfg.AIEncryptionKey = cfg.SecretKey
	}

	// Use PUBLIC_URL as fallback for OAuth redirect base
	if cfg.OAuthRedirectBase == "" {
		cfg.OAuthRedirectBase = cfg.PublicURL
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

func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("required environment variable " + key + " is not set")
	}
	return value
}
