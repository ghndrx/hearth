package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
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
}

// ServerServiceForUsersInterface defines the methods needed from ServerService
type ServerServiceForUsersInterface interface {
	GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error)
}

// ChannelServiceForUsersInterface defines the methods needed from ChannelService
type ChannelServiceForUsersInterface interface {
	GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error)
}

type UserHandler struct {
	userService    UserServiceInterface
	serverService  ServerServiceForUsersInterface
	channelService ChannelServiceForUsersInterface
}

func NewUserHandler(
	userService *services.UserService,
	serverService *services.ServerService,
	channelService *services.ChannelService,
) *UserHandler {
	return &UserHandler{
		userService:    userService,
		serverService:  serverService,
		channelService: channelService,
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

// ServerResponse represents a server in API responses
type ServerResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	IconURL     *string    `json:"icon_url,omitempty"`
	BannerURL   *string    `json:"banner_url,omitempty"`
	Description *string    `json:"description,omitempty"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	Features    []string   `json:"features"`
	CreatedAt   time.Time  `json:"created_at"`
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
	ID            uuid.UUID             `json:"id"`
	Type          models.ChannelType    `json:"type"`
	Recipients    []UserResponse        `json:"recipients"`
	LastMessageID *uuid.UUID            `json:"last_message_id,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
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

// RelationshipType defines the type of relationship
type RelationshipType int

const (
	RelationshipTypeFriend       RelationshipType = 1
	RelationshipTypeBlocked      RelationshipType = 2
	RelationshipTypePendingIn    RelationshipType = 3
	RelationshipTypePendingOut   RelationshipType = 4
)

// RelationshipResponse represents a relationship in API responses
type RelationshipResponse struct {
	ID   uuid.UUID        `json:"id"`
	Type RelationshipType `json:"type"`
	User UserResponse     `json:"user"`
}

// GetRelationships returns user's friends/blocked list
func (h *UserHandler) GetRelationships(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	// Get friends
	friends, err := h.userService.GetFriends(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get relationships",
		})
	}

	relationships := make([]RelationshipResponse, 0, len(friends))
	
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

	// TODO: Add blocked users and pending requests

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
		// For now, directly add as friend (skip request flow)
		// TODO: Implement proper friend request flow
		if err := h.userService.AddFriend(c.Context(), userID, targetID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to add friend",
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
