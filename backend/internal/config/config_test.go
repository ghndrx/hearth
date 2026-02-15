package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Clear any existing env vars that might affect the test
	oldVars := map[string]string{}
	keysToClean := []string{"HOST", "PORT", "DATABASE_URL", "LOG_LEVEL"}
	for _, k := range keysToClean {
		oldVars[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range oldVars {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := Load()

	if cfg == nil {
		t.Fatal("Load returned nil")
	}

	// Test defaults
	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected default Host '0.0.0.0', got '%s'", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default Port 8080, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("expected default LogFormat 'json', got '%s'", cfg.LogFormat)
	}
	if cfg.StorageBackend != "local" {
		t.Errorf("expected default StorageBackend 'local', got '%s'", cfg.StorageBackend)
	}
	if cfg.AuthProvider != "native" {
		t.Errorf("expected default AuthProvider 'native', got '%s'", cfg.AuthProvider)
	}
	if cfg.TokenExpiry != 1*time.Hour {
		t.Errorf("expected default TokenExpiry 1h, got %v", cfg.TokenExpiry)
	}
	if cfg.RefreshExpiry != 30*24*time.Hour {
		t.Errorf("expected default RefreshExpiry 30 days, got %v", cfg.RefreshExpiry)
	}
	if !cfg.RegistrationEnabled {
		t.Error("expected RegistrationEnabled true by default")
	}
	if cfg.InviteOnly {
		t.Error("expected InviteOnly false by default")
	}
	if cfg.Quotas == nil {
		t.Error("expected Quotas to be initialized")
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set test env vars
	os.Setenv("HOST", "127.0.0.1")
	os.Setenv("PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("REGISTRATION_ENABLED", "false")
	os.Setenv("INVITE_ONLY", "true")
	defer func() {
		os.Unsetenv("HOST")
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("REGISTRATION_ENABLED")
		os.Unsetenv("INVITE_ONLY")
	}()

	cfg := Load()

	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected Host '127.0.0.1', got '%s'", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected Port 9090, got %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", cfg.LogLevel)
	}
	if cfg.RegistrationEnabled {
		t.Error("expected RegistrationEnabled false")
	}
	if !cfg.InviteOnly {
		t.Error("expected InviteOnly true")
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{"returns default when not set", "TEST_EMPTY", "default", "", "default"},
		{"returns env value when set", "TEST_SET", "default", "custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}
			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%s, %s) = %s, expected %s", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{"returns default when not set", "TEST_INT_EMPTY", 42, "", 42},
		{"returns parsed int when valid", "TEST_INT_VALID", 42, "100", 100},
		{"returns default for invalid int", "TEST_INT_INVALID", 42, "not-a-number", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}
			result := getEnvInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvInt(%s, %d) = %d, expected %d", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{"returns default when not set", "TEST_BOOL_EMPTY", true, "", true},
		{"true string", "TEST_BOOL_TRUE", false, "true", true},
		{"TRUE uppercase", "TEST_BOOL_TRUE_UP", false, "TRUE", true},
		{"1 string", "TEST_BOOL_1", false, "1", true},
		{"yes string", "TEST_BOOL_YES", false, "yes", true},
		{"false string", "TEST_BOOL_FALSE", true, "false", false},
		{"0 string", "TEST_BOOL_0", true, "0", false},
		{"no string", "TEST_BOOL_NO", true, "no", false},
		{"invalid string defaults to false", "TEST_BOOL_INVALID", true, "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}
			result := getEnvBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvBool(%s, %v) = %v, expected %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{"returns default when not set", "TEST_DUR_EMPTY", time.Hour, "", time.Hour},
		{"parses valid duration", "TEST_DUR_VALID", time.Hour, "30m", 30 * time.Minute},
		{"parses hours", "TEST_DUR_HOURS", time.Minute, "2h", 2 * time.Hour},
		{"returns default for invalid", "TEST_DUR_INVALID", time.Hour, "not-a-duration", time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}
			result := getEnvDuration(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvDuration(%s, %v) = %v, expected %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestLoadQuotaConfig(t *testing.T) {
	// Test default quota config
	cfg := loadQuotaConfig()
	if cfg == nil {
		t.Fatal("loadQuotaConfig returned nil")
	}
	if cfg.Storage.UserStorageMB != 500 {
		t.Errorf("expected default UserStorageMB 500, got %d", cfg.Storage.UserStorageMB)
	}
}

func TestLoadQuotaConfigWithOverrides(t *testing.T) {
	os.Setenv("QUOTA_USER_STORAGE_MB", "1000")
	os.Setenv("QUOTA_SERVER_STORAGE_MB", "10000")
	os.Setenv("QUOTA_MAX_FILE_SIZE_MB", "50")
	defer func() {
		os.Unsetenv("QUOTA_USER_STORAGE_MB")
		os.Unsetenv("QUOTA_SERVER_STORAGE_MB")
		os.Unsetenv("QUOTA_MAX_FILE_SIZE_MB")
	}()

	cfg := loadQuotaConfig()

	if cfg.Storage.UserStorageMB != 1000 {
		t.Errorf("expected UserStorageMB 1000, got %d", cfg.Storage.UserStorageMB)
	}
	if cfg.Storage.ServerStorageMB != 10000 {
		t.Errorf("expected ServerStorageMB 10000, got %d", cfg.Storage.ServerStorageMB)
	}
	if cfg.Storage.MaxFileSizeMB != 50 {
		t.Errorf("expected MaxFileSizeMB 50, got %d", cfg.Storage.MaxFileSizeMB)
	}
}

func TestLoadQuotaConfigUnlimited(t *testing.T) {
	os.Setenv("QUOTAS_UNLIMITED", "true")
	defer os.Unsetenv("QUOTAS_UNLIMITED")

	cfg := loadQuotaConfig()

	// Should be unlimited config
	if cfg.Storage.UserStorageMB != 0 {
		t.Errorf("expected unlimited UserStorageMB (0), got %d", cfg.Storage.UserStorageMB)
	}
	if cfg.Messages.RateLimitMessages != 0 {
		t.Errorf("expected unlimited RateLimitMessages (0), got %d", cfg.Messages.RateLimitMessages)
	}
}
