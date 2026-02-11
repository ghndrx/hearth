package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// CacheService defines caching operations
type CacheService interface {
	// Users
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	SetUser(ctx context.Context, user *models.User, ttl time.Duration) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	
	// Servers
	GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error)
	SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error
	DeleteServer(ctx context.Context, id uuid.UUID) error
	
	// Channels
	GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error
	DeleteChannel(ctx context.Context, id uuid.UUID) error
	
	// Generic
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// EventBus defines event publishing
type EventBus interface {
	Publish(event string, data interface{})
	Subscribe(event string, handler func(data interface{}))
	Unsubscribe(event string, handler func(data interface{}))
}

// RateLimiter defines rate limiting operations
type RateLimiter interface {
	Check(ctx context.Context, userID, channelID uuid.UUID) error
	CheckSlowmode(ctx context.Context, userID, channelID uuid.UUID, seconds int) error
	Reset(ctx context.Context, userID, channelID uuid.UUID) error
}

// E2EEService defines E2EE operations
type E2EEService interface {
	// Validate that a payload is properly formatted encrypted content
	ValidateEncryptedPayload(payload string) bool
	
	// Key management for DMs
	GetPreKeys(ctx context.Context, userID uuid.UUID) (*models.PreKeyBundle, error)
	UploadPreKeys(ctx context.Context, userID uuid.UUID, bundle *models.PreKeyBundle) error
	
	// Group key management (MLS)
	CreateGroup(ctx context.Context, channelID uuid.UUID, memberIDs []uuid.UUID) error
	AddGroupMember(ctx context.Context, channelID, userID uuid.UUID) error
	RemoveGroupMember(ctx context.Context, channelID, userID uuid.UUID) error
}

// ChannelRepository defines channel data access
type ChannelRepository interface {
	Create(ctx context.Context, channel *models.Channel) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error)
	Update(ctx context.Context, channel *models.Channel) error
	Delete(ctx context.Context, id uuid.UUID) error
	
	// Queries
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error)
	GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error)
	GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)
	
	// Updates
	UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error
}

// RoleRepository defines role data access
type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	
	// Member roles
	AddToMember(ctx context.Context, memberID, roleID uuid.UUID) error
	RemoveFromMember(ctx context.Context, memberID, roleID uuid.UUID) error
}

// QuotaService defines quota checking
type QuotaService struct {
	config       *models.QuotaConfig
	serverRepo   ServerRepository
	userRepo     UserRepository
	roleRepo     RoleRepository
}

// NewQuotaService creates a new quota service
func NewQuotaService(config *models.QuotaConfig, serverRepo ServerRepository, userRepo UserRepository, roleRepo RoleRepository) *QuotaService {
	return &QuotaService{
		config:     config,
		serverRepo: serverRepo,
		userRepo:   userRepo,
		roleRepo:   roleRepo,
	}
}

// EffectiveLimits for quota checks
type EffectiveLimits struct {
	MaxMessageLength int
	MaxServersOwned  int
	MaxServersJoined int
	StorageMB        int64
	MaxFileSizeMB    int64
}

// GetEffectiveLimits calculates effective limits for a user
func (s *QuotaService) GetEffectiveLimits(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID) (*EffectiveLimits, error) {
	// Start with instance defaults
	limits := &EffectiveLimits{
		MaxMessageLength: s.config.Messages.MaxMessageLength,
		MaxServersOwned:  s.config.Servers.MaxServersOwned,
		MaxServersJoined: s.config.Servers.MaxServersJoined,
		StorageMB:        s.config.Storage.UserStorageMB,
		MaxFileSizeMB:    s.config.Storage.MaxFileSizeMB,
	}
	
	// TODO: Apply server, role, and user overrides
	
	return limits, nil
}

// CheckStorageQuota checks if a file upload is allowed
func (s *QuotaService) CheckStorageQuota(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID, fileSizeBytes int64) error {
	limits, err := s.GetEffectiveLimits(ctx, userID, serverID)
	if err != nil {
		return err
	}
	
	// Check file size limit
	if limits.MaxFileSizeMB > 0 {
		maxBytes := limits.MaxFileSizeMB * 1024 * 1024
		if fileSizeBytes > maxBytes {
			return models.NewFileTooLargeError(fileSizeBytes/(1024*1024), limits.MaxFileSizeMB)
		}
	}
	
	// TODO: Check total storage usage
	
	return nil
}
