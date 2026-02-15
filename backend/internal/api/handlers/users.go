package handlers

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
	"hearth/internal/storage"
)

// UserServiceInterface defines the methods needed from UserService
type UserServiceInterface interface {
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, updates *models.UserUpdate) (*models.User, error)
	GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error)
	AddFriend(ctx context.Context, userID, friendID uuid.UUID) error
	RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error
	BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error
	UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error
	
	// Friend requests
	SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error
	GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error)
	GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error)
	AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error
	DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error
	GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error)
	
	// Profile enhancements (UX-003) - optional, check via type assertion
	// GetMutualFriends(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.User, int, error)
	// GetRecentActivity(ctx context.Context, requesterID, targetID uuid.UUID) (*services.RecentActivityInfo, error)
}

// ServerServiceForUsersInterface defines the methods needed from ServerService
type ServerServiceForUsersInterface interface {
	GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error)
	// GetMutualServersLimited is optional for profile enhancements (UX-003)
}

// MutualServersService is an optional interface for getting mutual servers
type MutualServersService interface {
	GetMutualServersLimited(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.Server, int, error)
}

// MutualFriendsService is an optional interface for getting mutual friends
type MutualFriendsService interface {
	GetMutualFriends(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.User, int, error)
}

// RecentActivityService is an optional interface for getting recent activity
type RecentActivityService interface {
	GetRecentActivity(ctx context.Context, requesterID, targetID uuid.UUID) (*services.RecentActivityInfo, error)
}

// SharedChannelsService is an optional interface for getting shared channels
type SharedChannelsService interface {
	GetSharedChannelsWithServerNames(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]services.SharedChannelInfo, int, error)
}

// ChannelServiceForUsersInterface defines the methods needed from ChannelService
type ChannelServiceForUsersInterface interface {
	GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)
	GetOrCreateDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error)
	CreateGroupDM(ctx context.Context, ownerID uuid.UUID, name string, recipientIDs []uuid.UUID) (*models.Channel, error)
}

// StorageServiceInterface defines the methods needed for file storage
type StorageServiceInterface interface {
	UploadFile(ctx context.Context, file *multipart.FileHeader, uploaderID uuid.UUID, category string) (*storage.FileInfo, error)
	DeleteFile(ctx context.Context, path string) error
}

type UserHandler struct {
	userService    UserServiceInterface
	serverService  ServerServiceForUsersInterface
	channelService ChannelServiceForUsersInterface
	storageService StorageServiceInterface
}

func NewUserHandler(
	userService UserServiceInterface,
	serverService ServerServiceForUsersInterface,
	channelService ChannelServiceForUsersInterface,
) *UserHandler {
	return &UserHandler{
		userService:    userService,
		serverService:  serverService,
		channelService: channelService,
	}
}

// NewUserHandlerWithStorage creates a user handler with storage support for avatar uploads
func NewUserHandlerWithStorage(
	userService UserServiceInterface,
	serverService ServerServiceForUsersInterface,
	channelService ChannelServiceForUsersInterface,
	storageService StorageServiceInterface,
) *UserHandler {
	return &UserHandler{
		userService:    userService,
		serverService:  serverService,
		channelService: channelService,
		storageService: storageService,
	}
}

// GetMe returns the current user
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// UpdateMe updates the current user
func (h *UserHandler) UpdateMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		Username     *string `json:"username"`
		AvatarURL    *string `json:"avatar_url"`
		BannerURL    *string `json:"banner_url"`
		Bio          *string `json:"bio"`
		CustomStatus *string `json:"custom_status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate username if provided
	if req.Username != nil {
		if len(*req.Username) < 2 || len(*req.Username) > 32 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "username must be between 2 and 32 characters",
			})
		}
	}

	// Validate bio if provided
	if req.Bio != nil && len(*req.Bio) > 190 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "bio must be 190 characters or less",
		})
	}

	updates := &models.UserUpdate{
		Username:     req.Username,
		AvatarURL:    req.AvatarURL,
		BannerURL:    req.BannerURL,
		Bio:          req.Bio,
		CustomStatus: req.CustomStatus,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		if err == services.ErrUsernameTaken {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "username already taken",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update user",
		})
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// allowedAvatarTypes defines allowed content types for avatar uploads
var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// maxAvatarSize is the maximum avatar file size (8MB)
const maxAvatarSize = 8 * 1024 * 1024

// UpdateAvatar handles avatar file upload for the current user
func (h *UserHandler) UpdateAvatar(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Check if storage service is available
	if h.storageService == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"error": "file storage not configured",
		})
	}

	// Parse multipart form
	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "avatar file required",
		})
	}

	// Validate file size
	if file.Size > maxAvatarSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("avatar must be smaller than %dMB", maxAvatarSize/1024/1024),
		})
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if !allowedAvatarTypes[strings.ToLower(contentType)] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "avatar must be a JPEG, PNG, GIF, or WebP image",
		})
	}

	// Upload file
	fileInfo, err := h.storageService.UploadFile(c.Context(), file, userID, "avatars")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to upload avatar",
		})
	}

	// Update user with new avatar URL
	updates := &models.UserUpdate{
		AvatarURL: &fileInfo.URL,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		// Attempt to clean up uploaded file on failure
		_ = h.storageService.DeleteFile(c.Context(), fileInfo.Path)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update avatar",
		})
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// DeleteAvatar removes the current user's avatar
func (h *UserHandler) DeleteAvatar(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Set avatar to nil to remove it
	var nilAvatar *string = nil
	updates := &models.UserUpdate{
		AvatarURL: nilAvatar,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to remove avatar",
		})
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// ServerResponse represents a server in API responses
type ServerResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	IconURL     *string   `json:"icon_url,omitempty"`
	BannerURL   *string   `json:"banner_url,omitempty"`
	Description *string   `json:"description,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Features    []string  `json:"features"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetMyServers returns servers the user is a member of
func (h *UserHandler) GetMyServers(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	servers, err := h.serverService.GetUserServers(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get servers",
		})
	}

	response := make([]ServerResponse, len(servers))
	for i, s := range servers {
		features := s.Features
		if features == nil {
			features = []string{}
		}
		response[i] = ServerResponse{
			ID:          s.ID,
			Name:        s.Name,
			IconURL:     s.IconURL,
			BannerURL:   s.BannerURL,
			Description: s.Description,
			OwnerID:     s.OwnerID,
			Features:    features,
			CreatedAt:   s.CreatedAt,
		}
	}

	return c.JSON(response)
}

// DMChannelResponse represents a DM channel in API responses
type DMChannelResponse struct {
	ID            uuid.UUID          `json:"id"`
	Type          models.ChannelType `json:"type"`
	Recipients    []UserResponse     `json:"recipients"`
	LastMessageID *uuid.UUID         `json:"last_message_id,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

// GetMyDMs returns the user's DM channels
func (h *UserHandler) GetMyDMs(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channels, err := h.channelService.GetUserDMs(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get DMs",
		})
	}

	response := make([]DMChannelResponse, len(channels))
	for i, ch := range channels {
		// TODO: Load recipients for each DM
		response[i] = DMChannelResponse{
			ID:            ch.ID,
			Type:          ch.Type,
			Recipients:    []UserResponse{},
			LastMessageID: ch.LastMessageID,
			CreatedAt:     ch.CreatedAt,
		}
	}

	return c.JSON(response)
}

// CreateDM creates or retrieves a DM channel with another user
func (h *UserHandler) CreateDM(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.CreateDMRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.RecipientID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "recipient_id is required",
		})
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid recipient_id",
		})
	}

	if recipientID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot create DM with yourself",
		})
	}

	channel, err := h.channelService.GetOrCreateDM(c.Context(), userID, recipientID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create DM channel",
		})
	}

	return c.Status(fiber.StatusOK).JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Recipients:    []UserResponse{}, // TODO: Load recipient users
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// CreateGroupDM creates a new group DM channel
func (h *UserHandler) CreateGroupDM(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.CreateGroupDMRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if len(req.RecipientIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "at least one recipient is required",
		})
	}

	if len(req.RecipientIDs) > 9 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "group DM can have at most 10 members",
		})
	}

	// Parse recipient UUIDs
	recipientIDs := make([]uuid.UUID, 0, len(req.RecipientIDs))
	for _, idStr := range req.RecipientIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("invalid recipient_id: %s", idStr),
			})
		}
		if id == userID {
			continue // Skip self, owner is automatically added
		}
		recipientIDs = append(recipientIDs, id)
	}

	if len(recipientIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "at least one other recipient is required",
		})
	}

	// Use provided name or generate default
	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	channel, err := h.channelService.CreateGroupDM(c.Context(), userID, name, recipientIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create group DM",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Recipients:    []UserResponse{}, // TODO: Load recipient users
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// GetUser returns a user by ID
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	user, err := h.userService.GetUser(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	// Public profile - don't include email
	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		Discriminator: user.Discriminator,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// MutualServerResponse represents a mutual server in API responses
type MutualServerResponse struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	IconURL *string   `json:"icon_url,omitempty"`
}

// SharedChannelResponse represents a shared channel in API responses
type SharedChannelResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	ServerID   uuid.UUID `json:"server_id"`
	ServerName string    `json:"server_name"`
	ServerIcon *string   `json:"server_icon,omitempty"`
}

// RecentActivityResponse represents recent activity in API responses
type RecentActivityResponse struct {
	LastMessageAt   *time.Time `json:"last_message_at,omitempty"`
	ServerName      *string    `json:"server_name,omitempty"`
	ChannelName     *string    `json:"channel_name,omitempty"`
	MessageCount24h int        `json:"message_count_24h"`
}

// MutualFriendResponse represents a mutual friend in API responses
type MutualFriendResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
}

// UserProfileResponse represents enhanced user profile data
type UserProfileResponse struct {
	User           UserResponse            `json:"user"`
	MutualServers  []MutualServerResponse  `json:"mutual_servers"`
	SharedChannels []SharedChannelResponse `json:"shared_channels"`
	MutualFriends  []MutualFriendResponse  `json:"mutual_friends"`
	RecentActivity *RecentActivityResponse `json:"recent_activity,omitempty"`
	TotalMutual    struct {
		Servers  int `json:"servers"`
		Channels int `json:"channels"`
		Friends  int `json:"friends"`
	} `json:"total_mutual"`
}

// GetUserProfile returns enhanced user profile with mutual servers, shared channels, etc.
func (h *UserHandler) GetUserProfile(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)
	
	idParam := c.Params("id")
	targetID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	// Get target user
	user, err := h.userService.GetUser(c.Context(), targetID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	response := UserProfileResponse{
		User: UserResponse{
			ID:            user.ID,
			Username:      user.Username,
			Discriminator: user.Discriminator,
			AvatarURL:     user.AvatarURL,
			BannerURL:     user.BannerURL,
			Bio:           user.Bio,
			Flags:         user.Flags,
			CreatedAt:     user.CreatedAt,
		},
		MutualServers:  []MutualServerResponse{},
		SharedChannels: []SharedChannelResponse{},
		MutualFriends:  []MutualFriendResponse{},
	}

	// If viewing own profile, return basic info only (no "mutual" concept)
	if requesterID == targetID {
		return c.JSON(response)
	}

	// Get mutual servers (limit to 10 for popout)
	if svc, ok := h.serverService.(MutualServersService); ok {
		servers, total, err := svc.GetMutualServersLimited(c.Context(), requesterID, targetID, 10)
		if err == nil {
			response.TotalMutual.Servers = total
			for _, s := range servers {
				response.MutualServers = append(response.MutualServers, MutualServerResponse{
					ID:      s.ID,
					Name:    s.Name,
					IconURL: s.IconURL,
				})
			}
		}
	}

	// Get shared channels (limit to 10 for popout)
	if svc, ok := h.channelService.(SharedChannelsService); ok {
		channels, total, err := svc.GetSharedChannelsWithServerNames(c.Context(), requesterID, targetID, 10)
		if err == nil {
			response.TotalMutual.Channels = total
			for _, ch := range channels {
				if ch.ServerID != nil {
					response.SharedChannels = append(response.SharedChannels, SharedChannelResponse{
						ID:         ch.ID,
						Name:       ch.Name,
						ServerID:   *ch.ServerID,
						ServerName: ch.ServerName,
						ServerIcon: ch.ServerIcon,
					})
				}
			}
		}
	}

	// Get mutual friends (limit to 10 for popout)
	if svc, ok := h.userService.(MutualFriendsService); ok {
		friends, total, err := svc.GetMutualFriends(c.Context(), requesterID, targetID, 10)
		if err == nil {
			response.TotalMutual.Friends = total
			for _, f := range friends {
				response.MutualFriends = append(response.MutualFriends, MutualFriendResponse{
					ID:        f.ID,
					Username:  f.Username,
					AvatarURL: f.AvatarURL,
				})
			}
		}
	}

	// Get recent activity
	if svc, ok := h.userService.(RecentActivityService); ok {
		activity, err := svc.GetRecentActivity(c.Context(), requesterID, targetID)
		if err == nil && activity != nil {
			response.RecentActivity = &RecentActivityResponse{
				LastMessageAt:   activity.LastMessageAt,
				ServerName:      activity.ServerName,
				ChannelName:     activity.ChannelName,
				MessageCount24h: activity.MessageCount24h,
			}
		}
	}

	return c.JSON(response)
}

// RelationshipType defines the type of relationship
type RelationshipType int

const (
	RelationshipTypeFriend     RelationshipType = 1
	RelationshipTypeBlocked    RelationshipType = 2
	RelationshipTypePendingIn  RelationshipType = 3
	RelationshipTypePendingOut RelationshipType = 4
)

// RelationshipResponse represents a relationship in API responses
type RelationshipResponse struct {
	ID   uuid.UUID        `json:"id"`
	Type RelationshipType `json:"type"`
	User UserResponse     `json:"user"`
}

// GetRelationships returns user's friends/blocked list and pending requests
func (h *UserHandler) GetRelationships(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Get friends
	friends, err := h.userService.GetFriends(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get relationships",
		})
	}

	// Get incoming friend requests
	incoming, err := h.userService.GetIncomingFriendRequests(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get incoming requests",
		})
	}

	// Get outgoing friend requests
	outgoing, err := h.userService.GetOutgoingFriendRequests(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get outgoing requests",
		})
	}

	relationships := make([]RelationshipResponse, 0, len(friends)+len(incoming)+len(outgoing))

	for _, friend := range friends {
		relationships = append(relationships, RelationshipResponse{
			ID:   friend.ID,
			Type: RelationshipTypeFriend,
			User: UserResponse{
				ID:            friend.ID,
				Username:      friend.Username,
				Discriminator: friend.Discriminator,
				AvatarURL:     friend.AvatarURL,
				Flags:         friend.Flags,
			},
		})
	}

	for _, user := range incoming {
		relationships = append(relationships, RelationshipResponse{
			ID:   user.ID,
			Type: RelationshipTypePendingIn,
			User: UserResponse{
				ID:            user.ID,
				Username:      user.Username,
				Discriminator: user.Discriminator,
				AvatarURL:     user.AvatarURL,
				Flags:         user.Flags,
			},
		})
	}

	for _, user := range outgoing {
		relationships = append(relationships, RelationshipResponse{
			ID:   user.ID,
			Type: RelationshipTypePendingOut,
			User: UserResponse{
				ID:            user.ID,
				Username:      user.Username,
				Discriminator: user.Discriminator,
				AvatarURL:     user.AvatarURL,
				Flags:         user.Flags,
			},
		})
	}

	return c.JSON(relationships)
}

// CreateRelationship creates a friend request or block
func (h *UserHandler) CreateRelationship(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		UserID   uuid.UUID        `json:"user_id"`
		Type     RelationshipType `json:"type"`
		Username string           `json:"username"` // Alternative to user_id
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Resolve username to user_id if provided
	targetID := req.UserID
	if req.Username != "" && targetID == uuid.Nil {
		targetUser, err := h.userService.GetUserByUsername(c.Context(), req.Username)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "user not found",
			})
		}
		targetID = targetUser.ID
	}

	if targetID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id or username required",
		})
	}

	if targetID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot create relationship with yourself",
		})
	}

	switch req.Type {
	case RelationshipTypeFriend:
		// Send a friend request (pending state)
		if err := h.userService.SendFriendRequest(c.Context(), userID, targetID); err != nil {
			if strings.Contains(err.Error(), "already friends") {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "already friends",
				})
			}
			if strings.Contains(err.Error(), "already sent") {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "friend request already sent",
				})
			}
			if strings.Contains(err.Error(), "blocked") || strings.Contains(err.Error(), "cannot send friend request") {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "cannot send friend request to this user",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to send friend request",
			})
		}
	case RelationshipTypeBlocked:
		if err := h.userService.BlockUser(c.Context(), userID, targetID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to block user",
			})
		}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid relationship type",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AcceptFriendRequest accepts a pending friend request
func (h *UserHandler) AcceptFriendRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	senderParam := c.Params("id")
	senderID, err := uuid.Parse(senderParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.userService.AcceptFriendRequest(c.Context(), userID, senderID); err != nil {
		if strings.Contains(err.Error(), "no pending") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "no pending friend request from this user",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to accept friend request",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// DeclineFriendRequest declines or cancels a pending friend request
func (h *UserHandler) DeclineFriendRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	otherParam := c.Params("id")
	otherID, err := uuid.Parse(otherParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	if err := h.userService.DeclineFriendRequest(c.Context(), userID, otherID); err != nil {
		if strings.Contains(err.Error(), "no pending") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "no pending friend request",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to decline friend request",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetFriends returns the user's friends list
func (h *UserHandler) GetFriends(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	friends, err := h.userService.GetFriends(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get friends",
		})
	}

	response := make([]UserResponse, len(friends))
	for i, friend := range friends {
		response[i] = UserResponse{
			ID:            friend.ID,
			Username:      friend.Username,
			Discriminator: friend.Discriminator,
			AvatarURL:     friend.AvatarURL,
			Flags:         friend.Flags,
			CreatedAt:     friend.CreatedAt,
		}
	}

	return c.JSON(response)
}

// GetPendingFriendRequests returns pending friend requests (both incoming and outgoing)
func (h *UserHandler) GetPendingFriendRequests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Get incoming friend requests
	incoming, err := h.userService.GetIncomingFriendRequests(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get incoming requests",
		})
	}

	// Get outgoing friend requests
	outgoing, err := h.userService.GetOutgoingFriendRequests(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get outgoing requests",
		})
	}

	incomingResponse := make([]UserResponse, len(incoming))
	for i, user := range incoming {
		incomingResponse[i] = UserResponse{
			ID:            user.ID,
			Username:      user.Username,
			Discriminator: user.Discriminator,
			AvatarURL:     user.AvatarURL,
			Flags:         user.Flags,
		}
	}

	outgoingResponse := make([]UserResponse, len(outgoing))
	for i, user := range outgoing {
		outgoingResponse[i] = UserResponse{
			ID:            user.ID,
			Username:      user.Username,
			Discriminator: user.Discriminator,
			AvatarURL:     user.AvatarURL,
			Flags:         user.Flags,
		}
	}

	return c.JSON(fiber.Map{
		"incoming": incomingResponse,
		"outgoing": outgoingResponse,
	})
}

// DeleteRelationship removes a relationship
func (h *UserHandler) DeleteRelationship(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	targetParam := c.Params("id")
	targetID, err := uuid.Parse(targetParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user id",
		})
	}

	// Try to remove friend first
	if err := h.userService.RemoveFriend(c.Context(), userID, targetID); err != nil {
		// If not a friend, try to unblock
		if err := h.userService.UnblockUser(c.Context(), userID, targetID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to remove relationship",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
