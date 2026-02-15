package models

import (
	"testing"
)

func TestDefaultQuotaConfig(t *testing.T) {
	cfg := DefaultQuotaConfig()

	if cfg == nil {
		t.Fatal("DefaultQuotaConfig returned nil")
	}

	// Verify storage defaults
	if !cfg.Storage.Enabled {
		t.Error("Storage should be enabled by default")
	}
	if cfg.Storage.UserStorageMB != 500 {
		t.Errorf("expected UserStorageMB 500, got %d", cfg.Storage.UserStorageMB)
	}
	if cfg.Storage.ServerStorageMB != 5000 {
		t.Errorf("expected ServerStorageMB 5000, got %d", cfg.Storage.ServerStorageMB)
	}
	if cfg.Storage.MaxFileSizeMB != 25 {
		t.Errorf("expected MaxFileSizeMB 25, got %d", cfg.Storage.MaxFileSizeMB)
	}
	if cfg.Storage.MaxAttachmentsPerMsg != 10 {
		t.Errorf("expected MaxAttachmentsPerMsg 10, got %d", cfg.Storage.MaxAttachmentsPerMsg)
	}

	// Verify blocked extensions
	if len(cfg.Storage.BlockedExtensions) == 0 {
		t.Error("expected some blocked extensions")
	}

	// Verify message defaults
	if cfg.Messages.RateLimitMessages != 5 {
		t.Errorf("expected RateLimitMessages 5, got %d", cfg.Messages.RateLimitMessages)
	}
	if cfg.Messages.MaxMessageLength != 2000 {
		t.Errorf("expected MaxMessageLength 2000, got %d", cfg.Messages.MaxMessageLength)
	}

	// Verify server defaults
	if cfg.Servers.MaxServersOwned != 10 {
		t.Errorf("expected MaxServersOwned 10, got %d", cfg.Servers.MaxServersOwned)
	}
	if cfg.Servers.MaxServersJoined != 100 {
		t.Errorf("expected MaxServersJoined 100, got %d", cfg.Servers.MaxServersJoined)
	}

	// Verify voice defaults
	if !cfg.Voice.Enabled {
		t.Error("Voice should be enabled by default")
	}
	if cfg.Voice.MaxBitrateKbps != 384 {
		t.Errorf("expected MaxBitrateKbps 384, got %d", cfg.Voice.MaxBitrateKbps)
	}

	// Verify API defaults
	if cfg.API.RequestsPerMinute != 60 {
		t.Errorf("expected RequestsPerMinute 60, got %d", cfg.API.RequestsPerMinute)
	}
}

func TestUnlimitedQuotaConfig(t *testing.T) {
	cfg := UnlimitedQuotaConfig()

	if cfg == nil {
		t.Fatal("UnlimitedQuotaConfig returned nil")
	}

	// All storage limits should be 0 (unlimited)
	if cfg.Storage.UserStorageMB != 0 {
		t.Errorf("expected unlimited UserStorageMB (0), got %d", cfg.Storage.UserStorageMB)
	}
	if cfg.Storage.ServerStorageMB != 0 {
		t.Errorf("expected unlimited ServerStorageMB (0), got %d", cfg.Storage.ServerStorageMB)
	}
	if cfg.Storage.MaxFileSizeMB != 0 {
		t.Errorf("expected unlimited MaxFileSizeMB (0), got %d", cfg.Storage.MaxFileSizeMB)
	}

	// All message limits should be 0
	if cfg.Messages.RateLimitMessages != 0 {
		t.Errorf("expected unlimited RateLimitMessages (0), got %d", cfg.Messages.RateLimitMessages)
	}
	if cfg.Messages.MaxMessageLength != 0 {
		t.Errorf("expected unlimited MaxMessageLength (0), got %d", cfg.Messages.MaxMessageLength)
	}

	// All server limits should be 0
	if cfg.Servers.MaxServersOwned != 0 {
		t.Errorf("expected unlimited MaxServersOwned (0), got %d", cfg.Servers.MaxServersOwned)
	}

	// All API limits should be 0
	if cfg.API.RequestsPerMinute != 0 {
		t.Errorf("expected unlimited RequestsPerMinute (0), got %d", cfg.API.RequestsPerMinute)
	}

	// Blocked extensions should be empty for unlimited
	if len(cfg.Storage.BlockedExtensions) != 0 {
		t.Error("expected no blocked extensions in unlimited config")
	}
}

func TestIsUnlimited(t *testing.T) {
	tests := []struct {
		value    int64
		expected bool
	}{
		{0, true},
		{-1, true},
		{-100, true},
		{1, false},
		{100, false},
		{500, false},
	}

	for _, tt := range tests {
		result := IsUnlimited(tt.value)
		if result != tt.expected {
			t.Errorf("IsUnlimited(%d) = %v, expected %v", tt.value, result, tt.expected)
		}
	}
}

func TestNewStorageQuotaError(t *testing.T) {
	err := NewStorageQuotaError(400, 500, 150)

	if err == nil {
		t.Fatal("NewStorageQuotaError returned nil")
	}
	if err.Type != "quota_exceeded" {
		t.Errorf("expected type 'quota_exceeded', got '%s'", err.Type)
	}
	if err.Message == "" {
		t.Error("expected non-empty message")
	}
	if err.UpgradeURL != "/settings/premium" {
		t.Errorf("expected upgrade URL '/settings/premium', got '%s'", err.UpgradeURL)
	}
	if err.Details == nil {
		t.Fatal("expected details")
	}
	if err.Details["used_mb"] != int64(400) {
		t.Errorf("expected used_mb 400, got %v", err.Details["used_mb"])
	}
	if err.Details["limit_mb"] != int64(500) {
		t.Errorf("expected limit_mb 500, got %v", err.Details["limit_mb"])
	}
	if err.Details["file_size_mb"] != int64(150) {
		t.Errorf("expected file_size_mb 150, got %v", err.Details["file_size_mb"])
	}
	if err.Details["would_be_mb"] != int64(550) {
		t.Errorf("expected would_be_mb 550, got %v", err.Details["would_be_mb"])
	}

	// Test Error() method
	errStr := err.Error()
	if errStr != err.Message {
		t.Errorf("Error() should return Message")
	}
}

func TestNewRateLimitError(t *testing.T) {
	err := NewRateLimitError(5, 10, 60, 3)

	if err == nil {
		t.Fatal("NewRateLimitError returned nil")
	}
	if err.Type != "rate_limited" {
		t.Errorf("expected type 'rate_limited', got '%s'", err.Type)
	}
	if err.RetryAfter != 5 {
		t.Errorf("expected RetryAfter 5, got %d", err.RetryAfter)
	}
	if err.Details == nil {
		t.Fatal("expected details")
	}
	if err.Details["limit"] != 10 {
		t.Errorf("expected limit 10, got %v", err.Details["limit"])
	}
	if err.Details["window_seconds"] != 60 {
		t.Errorf("expected window_seconds 60, got %v", err.Details["window_seconds"])
	}
	if err.Details["slowmode_seconds"] != 3 {
		t.Errorf("expected slowmode_seconds 3, got %v", err.Details["slowmode_seconds"])
	}
}

func TestNewFileTooLargeError(t *testing.T) {
	err := NewFileTooLargeError(100, 25)

	if err == nil {
		t.Fatal("NewFileTooLargeError returned nil")
	}
	if err.Type != "file_too_large" {
		t.Errorf("expected type 'file_too_large', got '%s'", err.Type)
	}
	if err.Details == nil {
		t.Fatal("expected details")
	}
	if err.Details["file_size_mb"] != int64(100) {
		t.Errorf("expected file_size_mb 100, got %v", err.Details["file_size_mb"])
	}
	if err.Details["max_size_mb"] != int64(25) {
		t.Errorf("expected max_size_mb 25, got %v", err.Details["max_size_mb"])
	}
}

func TestQuotaErrorInterface(t *testing.T) {
	var err error = NewStorageQuotaError(100, 50, 75)

	// Should implement error interface
	if err.Error() == "" {
		t.Error("Error() should return non-empty string")
	}
}
