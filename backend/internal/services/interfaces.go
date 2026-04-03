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

// ComponentRateLimiter defines rate limiting for component interactions
type ComponentRateLimiter interface {
	// CheckComponentInteraction checks rate limit for component interactions
	CheckComponentInteraction(ctx context.Context, userID uuid.UUID) error
	// CheckModalSubmit checks rate limit for modal submissions
	CheckModalSubmit(ctx context.Context, userID uuid.UUID) error
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
	UpdateForumConfig(ctx context.Context, channelID uuid.UUID, configJSON []byte) error

	// DM recipient management
	AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error
	RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error
	CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error)

	// Bulk operations
	BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error

	// Permission overrides
	GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error)
	UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error
	DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error
}

// RoleRepository defines role data access
type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdatePositions(ctx context.Context, serverID uuid.UUID, positions map[uuid.UUID]int) error

	// Member role operations
	AddRoleToMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error
	RemoveRoleFromMember(ctx context.Context, serverID, userID, roleID uuid.UUID) error
	GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error)

	// Permission helpers
	GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error)
	GetDefaultRole(ctx context.Context, serverID uuid.UUID) (*models.Role, error)
}

// WebhookRepository defines webhook data access
type WebhookRepository interface {
	Create(ctx context.Context, webhook *models.Webhook) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error)
	GetByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.Webhook, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Webhook, error)
	Update(ctx context.Context, webhook *models.Webhook) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error)
}

// MessageSender defines message sending operations (used by WebhookService)
type MessageSender interface {
	SendMessage(ctx context.Context, authorID, channelID uuid.UUID, content string, attachments []*models.Attachment, replyTo *uuid.UUID, stickerID *uuid.UUID) (*models.Message, error)
}

// E2EERepository defines E2EE key management data access
type E2EERepository interface {
	// Device registration and management
	RegisterDevice(ctx context.Context, userID uuid.UUID, req *models.KeyUploadRequest) (*models.DeviceKey, error)
	GetUserDevices(ctx context.Context, userID uuid.UUID) ([]*models.E2EEDeviceInfo, error)
	DeleteDevice(ctx context.Context, userID uuid.UUID, deviceID string) error
	UpdateLastSeen(ctx context.Context, userID uuid.UUID, deviceID string) error

	// PreKey bundle operations
	GetPreKeyBundle(ctx context.Context, targetUserID uuid.UUID, targetDeviceID string, requestingUserID uuid.UUID) (*models.E2EEPreKeyBundle, error)
	GetPreKeyCount(ctx context.Context, userID uuid.UUID, deviceID string) (*models.PreKeyCount, error)
	UploadPreKeys(ctx context.Context, userID uuid.UUID, deviceID string, prekeys []models.PreKeyData) error
}

// StorageRepository defines storage usage data access
type StorageRepository interface {
	// GetUserStorageUsage returns the total bytes used by a user
	GetUserStorageUsage(ctx context.Context, userID uuid.UUID) (int64, error)
	// GetServerStorageUsage returns the total bytes used by a server
	GetServerStorageUsage(ctx context.Context, serverID uuid.UUID) (int64, error)
	// UpdateUserStorage adds to a user's storage usage (after successful upload)
	UpdateUserStorage(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID, bytesDelta int64, fileCountDelta int) error
	// DecrementUserStorage removes from a user's storage usage (after successful delete)
	DecrementUserStorage(ctx context.Context, userID uuid.UUID, serverID *uuid.UUID, bytesDelta int64, fileCountDelta int) error
}

// StickerRepository defines sticker data access
type StickerRepository interface {
	// CRUD operations
	Create(ctx context.Context, sticker *models.Sticker) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Sticker, error)
	Update(ctx context.Context, sticker *models.Sticker) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Query operations
	GetByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Sticker, error)
	GetGlobal(ctx context.Context) ([]*models.Sticker, error)
	GetAvailable(ctx context.Context, serverID *uuid.UUID) ([]*models.Sticker, error)
	Search(ctx context.Context, query string, serverID *uuid.UUID) ([]*models.Sticker, error)

	// Sticker Pack operations
	CreatePack(ctx context.Context, pack *models.StickerPack) error
	GetPackByID(ctx context.Context, id uuid.UUID) (*models.StickerPack, error)
	UpdatePack(ctx context.Context, pack *models.StickerPack) error
	DeletePack(ctx context.Context, id uuid.UUID) error
	GetPacksByServer(ctx context.Context, serverID uuid.UUID) ([]*models.StickerPack, error)
	GetGlobalPacks(ctx context.Context) ([]*models.StickerPack, error)
	GetPacksByTier(ctx context.Context, tier models.StickerPackTier) ([]*models.StickerPack, error)
	GetAvailablePacks(ctx context.Context, serverID *uuid.UUID, userTier models.StickerPackTier) ([]*models.StickerPack, error)

	// Pack-Sticker relationship operations
	AddStickerToPack(ctx context.Context, packID, stickerID uuid.UUID, position int, isDefault bool) error
	RemoveStickerFromPack(ctx context.Context, packID, stickerID uuid.UUID) error
	GetStickersInPack(ctx context.Context, packID uuid.UUID) ([]*models.Sticker, error)
	GetPacksContainingSticker(ctx context.Context, stickerID uuid.UUID) ([]*models.StickerPack, error)
}

type QuotaService struct {
	config      *models.QuotaConfig
	serverRepo  ServerRepository
	userRepo    UserRepository
	roleRepo    RoleRepository
	storageRepo StorageRepository
}

// NewQuotaService creates a new quota service
func NewQuotaService(config *models.QuotaConfig, serverRepo ServerRepository, userRepo UserRepository, roleRepo RoleRepository, storageRepo StorageRepository) *QuotaService {
	return &QuotaService{
		config:      config,
		serverRepo:  serverRepo,
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		storageRepo: storageRepo,
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

	// Apply server overrides if serverID is provided and serverRepo is available
	if serverID != nil && s.serverRepo != nil {
		if server, err := s.serverRepo.GetByID(ctx, *serverID); err == nil {
			// Server-specific limits would be applied here when ServerQuotas are stored
			// For now, server ownership boosts MaxServersOwned
			if server.OwnerID == userID {
				// Owner gets unlimited servers (0 means unlimited in our config)
				limits.MaxServersOwned = 0
			}
		}
	}

	// Apply role overrides if roleRepo is available
	if serverID != nil && s.roleRepo != nil {
		if roles, err := s.roleRepo.GetMemberRoles(ctx, *serverID, userID); err == nil {
			for _, role := range roles {
				// Higher position roles get boosted limits
				// Role position is relative to the server
				if role.Position > 0 {
					// Premium role boost: increase storage
					if limits.StorageMB > 0 {
						limits.StorageMB += int64(role.Position) * 100 // 100MB per role level
					}
				}
			}
		}
	}

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

	// Apply user-specific boosts (premium users get 2x storage)
	effectiveStorageMB := limits.StorageMB
	if s.userRepo != nil {
		if user, err := s.userRepo.GetByID(ctx, userID); err == nil {
			if user.Flags&models.UserFlagPremium != 0 {
				effectiveStorageMB *= 2
			}
		}
	}

	// Check total storage usage if storage limit is set and storage repo is available
	if effectiveStorageMB > 0 && s.storageRepo != nil {
		usedBytes, err := s.storageRepo.GetUserStorageUsage(ctx, userID)
		if err != nil {
			return err
		}
		usedMB := usedBytes / (1024 * 1024)
		limitMB := effectiveStorageMB
		if usedMB+fileSizeBytes/(1024*1024) > limitMB {
			return models.NewStorageQuotaError(usedMB, limitMB, fileSizeBytes/(1024*1024))
		}
	}

	return nil
}

// VoiceServiceInterface defines the interface for LiveKit voice operations
type VoiceServiceInterface interface {
	IsConfigured() bool
	GenerateToken(ctx context.Context, userID uuid.UUID, channelID uuid.UUID, userName, displayName, avatarURL string) (*VoiceTokenResponse, error)
	GetRoomParticipants(ctx context.Context, channelID uuid.UUID) ([]Participant, error)
	DisconnectParticipant(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	MuteParticipant(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, muted bool) error
}

// UserServiceInterface defines the interface for user operations needed by voice handler
type UserServiceInterface interface {
	GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

// ChannelServiceInterface defines the interface for channel operations needed by voice handler
type ChannelServiceInterface interface {
	GetChannel(ctx context.Context, channelID uuid.UUID) (*models.Channel, error)
	GetServerChannels(ctx context.Context, serverID uuid.UUID, requesterID uuid.UUID) ([]*models.Channel, error)
}

// VoicePermissionServiceInterface defines the interface for permission operations needed by voice handler
type VoicePermissionServiceInterface interface {
	RequirePermission(ctx context.Context, serverID, userID uuid.UUID, permission int64) error
}
