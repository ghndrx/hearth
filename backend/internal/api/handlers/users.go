package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	
	"hearth/internal/services"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
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
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Banner:      user.Banner,
		Bio:         user.Bio,
		Pronouns:    user.Pronouns,
		Bot:         user.Bot,
		CreatedAt:   user.CreatedAt,
	})
}

// UpdateMe updates the current user
func (h *UserHandler) UpdateMe(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	
	var req struct {
		Username    *string `json:"username"`
		DisplayName *string `json:"display_name"`
		Avatar      *string `json:"avatar"`
		Banner      *string `json:"banner"`
		Bio         *string `json:"bio"`
		Pronouns    *string `json:"pronouns"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	
	// TODO: Implement update
	_ = userID
	
	return c.JSON(fiber.Map{"status": "updated"})
}

// GetMyServers returns servers the user is a member of
func (h *UserHandler) GetMyServers(c *fiber.Ctx) error {
	// userID := c.Locals("userID").(uuid.UUID)
	
	// TODO: Get user's servers
	
	return c.JSON([]interface{}{})
}

// GetMyDMs returns the user's DM channels
func (h *UserHandler) GetMyDMs(c *fiber.Ctx) error {
	// userID := c.Locals("userID").(uuid.UUID)
	
	// TODO: Get user's DM channels
	
	return c.JSON([]interface{}{})
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
	
	return c.JSON(UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Avatar:      user.Avatar,
		Bot:         user.Bot,
		CreatedAt:   user.CreatedAt,
	})
}

// GetRelationships returns user's friends/blocked list
func (h *UserHandler) GetRelationships(c *fiber.Ctx) error {
	// userID := c.Locals("userID").(uuid.UUID)
	
	// TODO: Get relationships
	
	return c.JSON([]interface{}{})
}

// CreateRelationship creates a friend request or block
func (h *UserHandler) CreateRelationship(c *fiber.Ctx) error {
	// userID := c.Locals("userID").(uuid.UUID)
	
	var req struct {
		UserID uuid.UUID `json:"user_id"`
		Type   int       `json:"type"` // 1=friend, 2=block
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	
	// TODO: Create relationship
	
	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteRelationship removes a relationship
func (h *UserHandler) DeleteRelationship(c *fiber.Ctx) error {
	// userID := c.Locals("userID").(uuid.UUID)
	// targetID := c.Params("id")
	
	// TODO: Delete relationship
	
	return c.SendStatus(fiber.StatusNoContent)
}
