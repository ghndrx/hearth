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

	// Custom Status
	GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error)
	SetCustomStatus(ctx context.Context, userID uuid.UUID, req *models.UpdateStatusRequest) (*models.UserCustomStatus, error)
	ClearCustomStatus(ctx context.Context, userID uuid.UUID) error

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
// @Summary Get current user
// @Description Returns the current authenticated user's profile
// @Tags Users
// @Produce json
// @Success 200 {object} UserResponse "Current user profile"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "User not found"
// @Router /users/@me [get]
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	user, err := h.userService.GetUser(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// UpdateMe updates the current user
// @Summary Update current user
// @Description Updates the current user's profile information
// @Tags Users
// @Accept json
// @Produce json
// @Param body body struct{Username *string `json:"username"`; DisplayName *string `json:"display_name"`; AvatarURL *string `json:"avatar_url"`; BannerURL *string `json:"banner_url"`; Bio *string `json:"bio"`; AboutMe *string `json:"about_me"`; Pronouns *string `json:"pronouns"`; AccentColor *int `json:"accent_color"`; CustomStatus *string `json:"custom_status"`} true "User update data"
// @Success 200 {object} UserResponse "Updated user profile"
// @Failure 400 {object} fiber.Map "Invalid request or validation error"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 409 {object} fiber.Map "Username already taken"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me [patch]
func (h *UserHandler) UpdateMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		Username     *string `json:"username"`
		DisplayName  *string `json:"display_name"`
		AvatarURL    *string `json:"avatar_url"`
		BannerURL    *string `json:"banner_url"`
		Bio          *string `json:"bio"`
		AboutMe      *string `json:"about_me"`
		Pronouns     *string `json:"pronouns"`
		AccentColor  *int    `json:"accent_color"`
		CustomStatus *string `json:"custom_status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.Username != nil {
		if len(*req.Username) < 2 || len(*req.Username) > 32 {
			return ValidationError(c, "username", "must be between 2 and 32 characters")
		}
	}

	if req.DisplayName != nil && len(*req.DisplayName) > 32 {
		return ValidationError(c, "display_name", "must be 32 characters or less")
	}

	if req.Bio != nil && len(*req.Bio) > 190 {
		return ValidationError(c, "bio", "must be 190 characters or less")
	}

	if req.AboutMe != nil && len(*req.AboutMe) > 2000 {
		return ValidationError(c, "about_me", "must be 2000 characters or less")
	}

	if req.Pronouns != nil && len(*req.Pronouns) > 32 {
		return ValidationError(c, "pronouns", "must be 32 characters or less")
	}

	updates := &models.UserUpdate{
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		AvatarURL:    req.AvatarURL,
		BannerURL:    req.BannerURL,
		Bio:          req.Bio,
		AboutMe:      req.AboutMe,
		Pronouns:     req.Pronouns,
		AccentColor:  req.AccentColor,
		CustomStatus: req.CustomStatus,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
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
// @Summary Update user avatar
// @Description Uploads a new avatar image for the current user
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Avatar image file (JPEG, PNG, GIF, WebP, max 8MB)"
// @Success 200 {object} UserResponse "Updated user profile with new avatar"
// @Failure 400 {object} fiber.Map "Invalid file or file too large"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 501 {object} fiber.Map "File storage not configured"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/avatar [patch]
func (h *UserHandler) UpdateAvatar(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	if h.storageService == nil {
		return NotImplemented(c, "file storage not configured")
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return BadRequest(c, "avatar file required")
	}

	if file.Size > maxAvatarSize {
		return BadRequest(c, fmt.Sprintf("avatar must be smaller than %dMB", maxAvatarSize/1024/1024))
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedAvatarTypes[strings.ToLower(contentType)] {
		return BadRequest(c, "avatar must be a JPEG, PNG, GIF, or WebP image")
	}

	fileInfo, err := h.storageService.UploadFile(c.Context(), file, userID, "avatars")
	if err != nil {
		return InternalError(c, "failed to upload avatar")
	}

	updates := &models.UserUpdate{
		AvatarURL: &fileInfo.URL,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		_ = h.storageService.DeleteFile(c.Context(), fileInfo.Path)
		return HandleServiceError(c, err)
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// DeleteAvatar removes the current user's avatar
// @Summary Delete user avatar
// @Description Removes the current user's avatar image
// @Tags Users
// @Produce json
// @Success 200 {object} UserResponse "Updated user profile without avatar"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/avatar [delete]
func (h *UserHandler) DeleteAvatar(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var nilAvatar *string = nil
	updates := &models.UserUpdate{
		AvatarURL: nilAvatar,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// maxBannerSize is the maximum banner file size (8MB)
const maxBannerSize = 8 * 1024 * 1024

// UpdateBanner handles banner file upload for the current user
// @Summary Update user banner
// @Description Uploads a new banner image for the current user
// @Tags Users
// @Accept multipart/form-data
// @Produce json
// @Param banner formData file true "Banner image file (JPEG, PNG, GIF, WebP, max 8MB)"
// @Success 200 {object} UserResponse "Updated user profile with new banner"
// @Failure 400 {object} fiber.Map "Invalid file or file too large"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 501 {object} fiber.Map "File storage not configured"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/banner [patch]
func (h *UserHandler) UpdateBanner(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	if h.storageService == nil {
		return NotImplemented(c, "file storage not configured")
	}

	file, err := c.FormFile("banner")
	if err != nil {
		return BadRequest(c, "banner file required")
	}

	if file.Size > maxBannerSize {
		return BadRequest(c, fmt.Sprintf("banner must be smaller than %dMB", maxBannerSize/1024/1024))
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedAvatarTypes[strings.ToLower(contentType)] {
		return BadRequest(c, "banner must be a JPEG, PNG, GIF, or WebP image")
	}

	fileInfo, err := h.storageService.UploadFile(c.Context(), file, userID, "banners")
	if err != nil {
		return InternalError(c, "failed to upload banner")
	}

	updates := &models.UserUpdate{
		BannerURL: &fileInfo.URL,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		_ = h.storageService.DeleteFile(c.Context(), fileInfo.Path)
		return HandleServiceError(c, err)
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
		CustomStatus:  user.CustomStatus,
		Flags:         user.Flags,
		CreatedAt:     user.CreatedAt,
	})
}

// DeleteBanner removes the current user's banner
// @Summary Delete user banner
// @Description Removes the current user's banner image
// @Tags Users
// @Produce json
// @Success 200 {object} UserResponse "Updated user profile without banner"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/banner [delete]
func (h *UserHandler) DeleteBanner(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var nilBanner *string = nil
	updates := &models.UserUpdate{
		BannerURL: nilBanner,
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, updates)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		Email:         &user.Email,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
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
// @Summary Get user's servers
// @Description Returns a list of servers the current user is a member of
// @Tags Users
// @Produce json
// @Success 200 {array} ServerResponse "List of user's servers"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/guilds [get]
func (h *UserHandler) GetMyServers(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	servers, err := h.serverService.GetUserServers(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get servers")
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
	Name          string             `json:"name,omitempty"`
	OwnerID       *uuid.UUID         `json:"owner_id,omitempty"`
	Recipients    []UserResponse     `json:"recipients"`
	LastMessageID *uuid.UUID         `json:"last_message_id,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
}

// GetMyDMs returns the user's DM channels
// @Summary Get user's DMs
// @Description Returns a list of the current user's DM channels
// @Tags Users
// @Produce json
// @Success 200 {array} DMChannelResponse "List of DM channels"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/channels [get]
func (h *UserHandler) GetMyDMs(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	channels, err := h.channelService.GetUserDMs(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get DMs")
	}

	response := make([]DMChannelResponse, len(channels))
	for i, ch := range channels {
		recipients := make([]UserResponse, 0, len(ch.Recipients))
		for _, recipientID := range ch.Recipients {
			if recipientID == userID {
				continue
			}
			user, err := h.userService.GetUser(c.Context(), recipientID)
			if err != nil {
				continue
			}
			recipients = append(recipients, *toUserResponse(user))
		}
		response[i] = DMChannelResponse{
			ID:            ch.ID,
			Type:          ch.Type,
			Recipients:    recipients,
			LastMessageID: ch.LastMessageID,
			CreatedAt:     ch.CreatedAt,
		}
	}

	return c.JSON(response)
}

// CreateDM creates or retrieves a DM channel with another user
// @Summary Create DM channel
// @Description Creates or retrieves a DM channel with another user
// @Tags Users
// @Accept json
// @Produce json
// @Param body body models.CreateDMRequest true "DM creation request"
// @Success 200 {object} DMChannelResponse "DM channel"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/channels [post]
func (h *UserHandler) CreateDM(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.CreateDMRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if req.RecipientID == "" {
		return BadRequest(c, "recipient_id is required")
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		return InvalidUUID(c, "recipient_id")
	}

	if recipientID == userID {
		return BadRequest(c, "cannot create DM with yourself")
	}

	channel, err := h.channelService.GetOrCreateDM(c.Context(), userID, recipientID)
	if err != nil {
		return InternalError(c, "failed to create DM channel")
	}

	recipientUser, err := h.userService.GetUser(c.Context(), recipientID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Recipients:    []UserResponse{*toUserResponse(recipientUser)},
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// CreateGroupDM creates a new group DM channel
// @Summary Create group DM
// @Description Creates a new group DM channel with multiple users
// @Tags Users
// @Accept json
// @Produce json
// @Param body body models.CreateGroupDMRequest true "Group DM creation request"
// @Success 201 {object} DMChannelResponse "Group DM channel created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/channels/group [post]
func (h *UserHandler) CreateGroupDM(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.CreateGroupDMRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	if len(req.RecipientIDs) == 0 {
		return BadRequest(c, "at least one recipient is required")
	}

	if len(req.RecipientIDs) > 9 {
		return BadRequest(c, "group DM can have at most 10 members")
	}

	recipientIDs := make([]uuid.UUID, 0, len(req.RecipientIDs))
	for _, idStr := range req.RecipientIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return BadRequest(c, fmt.Sprintf("invalid recipient_id: %s", idStr))
		}
		if id == userID {
			continue
		}
		recipientIDs = append(recipientIDs, id)
	}

	if len(recipientIDs) == 0 {
		return BadRequest(c, "at least one other recipient is required")
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	channel, err := h.channelService.CreateGroupDM(c.Context(), userID, name, recipientIDs)
	if err != nil {
		return InternalError(c, "failed to create group DM")
	}

	recipients := make([]UserResponse, 0, len(channel.Recipients))
	for _, recipientID := range channel.Recipients {
		user, err := h.userService.GetUser(c.Context(), recipientID)
		if err != nil {
			continue
		}
		recipients = append(recipients, *toUserResponse(user))
	}

	return c.Status(fiber.StatusCreated).JSON(DMChannelResponse{
		ID:            channel.ID,
		Type:          channel.Type,
		Recipients:    recipients,
		LastMessageID: channel.LastMessageID,
		CreatedAt:     channel.CreatedAt,
	})
}

// GetUser returns a user by ID
// @Summary Get user by ID
// @Description Returns a user's public profile by their ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} UserResponse "User profile"
// @Failure 400 {object} fiber.Map "Invalid user ID"
// @Failure 404 {object} fiber.Map "User not found"
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return InvalidUUID(c, "user id")
	}

	user, err := h.userService.GetUser(c.Context(), id)
	if err != nil {
		return HandleServiceError(c, err)
	}

	// Public profile - don't include email
	return c.JSON(UserResponse{
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Discriminator: user.Discriminator,
		AvatarURL:     user.AvatarURL,
		BannerURL:     user.BannerURL,
		Bio:           user.Bio,
		AboutMe:       user.AboutMe,
		Pronouns:      user.Pronouns,
		AccentColor:   user.AccentColor,
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
// @Summary Get user profile
// @Description Returns an enhanced user profile including mutual servers, shared channels, mutual friends, and recent activity
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} UserProfileResponse "Enhanced user profile"
// @Failure 400 {object} fiber.Map "Invalid user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "User not found"
// @Router /users/{id}/profile [get]
func (h *UserHandler) GetUserProfile(c *fiber.Ctx) error {
	requesterID := c.Locals("userID").(uuid.UUID)

	idParam := c.Params("id")
	targetID, err := uuid.Parse(idParam)
	if err != nil {
		return InvalidUUID(c, "user id")
	}

	user, err := h.userService.GetUser(c.Context(), targetID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	response := UserProfileResponse{
		User: UserResponse{
			ID:            user.ID,
			Username:      user.Username,
			DisplayName:   user.DisplayName,
			Discriminator: user.Discriminator,
			AvatarURL:     user.AvatarURL,
			BannerURL:     user.BannerURL,
			Bio:           user.Bio,
			AboutMe:       user.AboutMe,
			Pronouns:      user.Pronouns,
			AccentColor:   user.AccentColor,
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
// @Summary Get relationships
// @Description Returns the current user's friends, blocked users, and pending friend requests
// @Tags Users
// @Produce json
// @Success 200 {array} RelationshipResponse "List of relationships"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/relationships [get]
func (h *UserHandler) GetRelationships(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	friends, err := h.userService.GetFriends(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get relationships")
	}

	incoming, err := h.userService.GetIncomingFriendRequests(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get incoming requests")
	}

	outgoing, err := h.userService.GetOutgoingFriendRequests(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get outgoing requests")
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
// @Summary Create relationship
// @Description Sends a friend request or blocks a user
// @Tags Users
// @Accept json
// @Produce json
// @Param body body struct{UserID uuid.UUID `json:"user_id"`; Type RelationshipType `json:"type"`; Username string `json:"username"`} true "Relationship creation request"
// @Success 204 "Relationship created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 403 {object} fiber.Map "Cannot send friend request"
// @Failure 404 {object} fiber.Map "User not found"
// @Failure 409 {object} fiber.Map "Already friends or request pending"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/relationships [post]
func (h *UserHandler) CreateRelationship(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req struct {
		UserID   uuid.UUID        `json:"user_id"`
		Type     RelationshipType `json:"type"`
		Username string           `json:"username"`
	}

	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	targetID := req.UserID
	if req.Username != "" && targetID == uuid.Nil {
		targetUser, err := h.userService.GetUserByUsername(c.Context(), req.Username)
		if err != nil {
			return NotFound(c, "user not found")
		}
		targetID = targetUser.ID
	}

	if targetID == uuid.Nil {
		return BadRequest(c, "user_id or username required")
	}

	if targetID == userID {
		return BadRequest(c, "cannot create relationship with yourself")
	}

	switch req.Type {
	case RelationshipTypeFriend:
		if err := h.userService.SendFriendRequest(c.Context(), userID, targetID); err != nil {
			if strings.Contains(err.Error(), "already friends") {
				return Conflict(c, "already friends")
			}
			if strings.Contains(err.Error(), "already sent") {
				return Conflict(c, "friend request already sent")
			}
			if strings.Contains(err.Error(), "blocked") || strings.Contains(err.Error(), "cannot send friend request") {
				return Forbidden(c, "cannot send friend request to this user")
			}
			return InternalError(c, "failed to send friend request")
		}
	case RelationshipTypeBlocked:
		if err := h.userService.BlockUser(c.Context(), userID, targetID); err != nil {
			return InternalError(c, "failed to block user")
		}
	default:
		return BadRequest(c, "invalid relationship type")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AcceptFriendRequest accepts a pending friend request
// @Summary Accept friend request
// @Description Accepts a pending friend request from another user
// @Tags Users
// @Param id path string true "User ID of the request sender"
// @Success 204 "Friend request accepted"
// @Failure 400 {object} fiber.Map "Invalid user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 404 {object} fiber.Map "No pending friend request found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/relationships/{id} [put]
func (h *UserHandler) AcceptFriendRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	senderParam := c.Params("id")
	senderID, err := uuid.Parse(senderParam)
	if err != nil {
		return InvalidUUID(c, "user id")
	}

	if err := h.userService.AcceptFriendRequest(c.Context(), userID, senderID); err != nil {
		if strings.Contains(err.Error(), "no pending") {
			return NotFound(c, "no pending friend request from this user")
		}
		return InternalError(c, "failed to accept friend request")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *UserHandler) DeclineFriendRequest(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	otherParam := c.Params("id")
	otherID, err := uuid.Parse(otherParam)
	if err != nil {
		return InvalidUUID(c, "user id")
	}

	if err := h.userService.DeclineFriendRequest(c.Context(), userID, otherID); err != nil {
		if strings.Contains(err.Error(), "no pending") {
			return NotFound(c, "no pending friend request")
		}
		return InternalError(c, "failed to decline friend request")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetFriends returns the user's friends list
// @Summary Get friends list
// @Description Returns the current user's friends list
// @Tags Users
// @Produce json
// @Success 200 {array} UserResponse "List of friends"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/friends [get]
func (h *UserHandler) GetFriends(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	friends, err := h.userService.GetFriends(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get friends")
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
// @Summary Get pending friend requests
// @Description Returns the current user's pending friend requests (both incoming and outgoing)
// @Tags Users
// @Produce json
// @Success 200 {object} fiber.Map "Object with incoming and outgoing arrays"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/relationships/pending [get]
func (h *UserHandler) GetPendingFriendRequests(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	incoming, err := h.userService.GetIncomingFriendRequests(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get incoming requests")
	}

	outgoing, err := h.userService.GetOutgoingFriendRequests(c.Context(), userID)
	if err != nil {
		return InternalError(c, "failed to get outgoing requests")
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
// @Summary Delete relationship
// @Description Removes a friend or unblocks a user
// @Tags Users
// @Param id path string true "User ID"
// @Success 204 "Relationship removed"
// @Failure 400 {object} fiber.Map "Invalid user ID"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/@me/relationships/{id} [delete]
func (h *UserHandler) DeleteRelationship(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	targetParam := c.Params("id")
	targetID, err := uuid.Parse(targetParam)
	if err != nil {
		return InvalidUUID(c, "user id")
	}

	if err := h.userService.RemoveFriend(c.Context(), userID, targetID); err != nil {
		if err := h.userService.UnblockUser(c.Context(), userID, targetID); err != nil {
			return InternalError(c, "failed to remove relationship")
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetMyStatus returns the current user's custom status
// @Summary Get custom status
// @Description Returns the current user's rich custom status with emoji and expiration
// @Tags Users
// @Produce json
// @Success 200 {object} models.UserCustomStatus "Custom status"
// @Success 204 "No custom status set"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Router /users/@me/status [get]
func (h *UserHandler) GetMyStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	status, err := h.userService.GetCustomStatus(c.Context(), userID)
	if err != nil {
		return HandleServiceError(c, err)
	}

	if status == nil {
		return c.SendStatus(fiber.StatusNoContent)
	}

	// Check if status has expired
	if status.ClearAfter != nil && status.ClearAfter.Before(time.Now()) {
		_ = h.userService.ClearCustomStatus(c.Context(), userID)
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.JSON(status)
}

// UpdateMyStatus sets or clears the current user's custom status
// @Summary Set custom status
// @Description Sets the current user's rich custom status with emoji and optional expiration
// @Tags Users
// @Accept json
// @Produce json
// @Param body body models.UpdateStatusRequest true "Status data"
// @Success 200 {object} models.UserCustomStatus "Updated custom status"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Router /users/@me/status [put]
func (h *UserHandler) UpdateMyStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req models.UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return ParseError(c, err)
	}

	// Validate custom text length
	if req.CustomText != nil && len(*req.CustomText) > 128 {
		return ValidationError(c, "custom_text", "must be 128 characters or less")
	}

	// Validate emoji length
	if req.Emoji != nil && len(*req.Emoji) > 64 {
		return ValidationError(c, "emoji", "must be 64 characters or less")
	}

	// If everything is nil/empty, clear the status
	if req.CustomText == nil && req.Emoji == nil && req.EmojiID == nil && req.EmojiName == nil {
		if err := h.userService.ClearCustomStatus(c.Context(), userID); err != nil {
			return InternalError(c, "failed to clear status")
		}
		return c.SendStatus(fiber.StatusNoContent)
	}

	status, err := h.userService.SetCustomStatus(c.Context(), userID, &req)
	if err != nil {
		return HandleServiceError(c, err)
	}

	return c.JSON(status)
}

// DeleteMyStatus clears the current user's custom status
// @Summary Clear custom status
// @Description Removes the current user's custom status
// @Tags Users
// @Success 204 "Status cleared"
// @Failure 401 {object} fiber.Map "Unauthorized"
// @Router /users/@me/status [delete]
func (h *UserHandler) DeleteMyStatus(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	if err := h.userService.ClearCustomStatus(c.Context(), userID); err != nil {
		return InternalError(c, "failed to clear status")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
