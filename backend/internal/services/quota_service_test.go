package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"hearth/internal/models"
)

func TestNewQuotaService(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)

	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
}

func TestQuotaService_GetEffectiveLimits(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	limits, err := service.GetEffectiveLimits(ctx, userID, nil)

	assert.NoError(t, err)
	assert.NotNil(t, limits)
	assert.Equal(t, 4000, limits.MaxMessageLength)
	assert.Equal(t, 10, limits.MaxServersOwned)
	assert.Equal(t, 100, limits.MaxServersJoined)
	assert.Equal(t, int64(100), limits.StorageMB)
	assert.Equal(t, int64(25), limits.MaxFileSizeMB)
}

func TestQuotaService_GetEffectiveLimitsWithServer(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	limits, err := service.GetEffectiveLimits(ctx, userID, &serverID)

	assert.NoError(t, err)
	assert.NotNil(t, limits)
	// Server context should eventually modify limits, but for now returns defaults
	assert.Equal(t, 4000, limits.MaxMessageLength)
}

func TestQuotaService_CheckStorageQuota_Success(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	// 10MB file should be allowed (under 25MB limit)
	err := service.CheckStorageQuota(ctx, userID, nil, 10*1024*1024)

	assert.NoError(t, err)
}

func TestQuotaService_CheckStorageQuota_FileTooLarge(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	// 30MB file should exceed 25MB limit
	err := service.CheckStorageQuota(ctx, userID, nil, 30*1024*1024)

	assert.Error(t, err)
	// Should be a QuotaError (file_too_large type)
	quotaErr, ok := err.(*models.QuotaError)
	assert.True(t, ok, "expected QuotaError")
	assert.Equal(t, "file_too_large", quotaErr.Type)
}

func TestQuotaService_CheckStorageQuota_ExactLimit(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	// Exactly 25MB file should be allowed (limit is maxBytes = 25 * 1024 * 1024)
	err := service.CheckStorageQuota(ctx, userID, nil, 25*1024*1024)

	assert.NoError(t, err)
}

func TestQuotaService_CheckStorageQuota_JustOverLimit(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	// 1 byte over limit
	err := service.CheckStorageQuota(ctx, userID, nil, 25*1024*1024+1)

	assert.Error(t, err)
}

func TestQuotaService_CheckStorageQuota_ZeroLimit(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 0, // No limit
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	// Any size should be allowed when limit is 0
	err := service.CheckStorageQuota(ctx, userID, nil, 1000*1024*1024)

	assert.NoError(t, err)
}

func TestQuotaService_CheckStorageQuota_WithServer(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	// Test with server context
	err := service.CheckStorageQuota(ctx, userID, &serverID, 10*1024*1024)

	assert.NoError(t, err)
}

func TestEffectiveLimits_Structure(t *testing.T) {
	limits := &EffectiveLimits{
		MaxMessageLength: 4000,
		MaxServersOwned:  10,
		MaxServersJoined: 100,
		StorageMB:        500,
		MaxFileSizeMB:    50,
	}

	assert.Equal(t, 4000, limits.MaxMessageLength)
	assert.Equal(t, 10, limits.MaxServersOwned)
	assert.Equal(t, 100, limits.MaxServersJoined)
	assert.Equal(t, int64(500), limits.StorageMB)
	assert.Equal(t, int64(50), limits.MaxFileSizeMB)
}

func TestQuotaService_ZeroFileSizeAllowed(t *testing.T) {
	config := &models.QuotaConfig{
		Messages: models.MessageQuotaConfig{
			MaxMessageLength: 4000,
		},
		Servers: models.ServerQuotaConfig{
			MaxServersOwned:  10,
			MaxServersJoined: 100,
		},
		Storage: models.StorageQuotaConfig{
			UserStorageMB: 100,
			MaxFileSizeMB: 25,
		},
	}

	service := NewQuotaService(config, nil, nil, nil)
	ctx := context.Background()
	userID := uuid.New()

	// Zero byte file should always be allowed
	err := service.CheckStorageQuota(ctx, userID, nil, 0)

	assert.NoError(t, err)
}
