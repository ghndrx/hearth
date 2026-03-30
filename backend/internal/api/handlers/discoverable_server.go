package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// DiscoverableServerHandler handles public server directory endpoints
type DiscoverableServerHandler struct {
	discoverableServerService *services.DiscoverableServerService
	serverService             *services.ServerService
}

// NewDiscoverableServerHandler creates a new discoverable server handler
func NewDiscoverableServerHandler(
	discoverableServerService *services.DiscoverableServerService,
	serverService *services.ServerService,
) *DiscoverableServerHandler {
	return &DiscoverableServerHandler{
		discoverableServerService: discoverableServerService,
		serverService:             serverService,
	}
}

// GetDiscoverableServers returns paginated discoverable servers with optional search and category filter
// GET /api/v1/servers/discover
func (h *DiscoverableServerHandler) GetDiscoverableServers(c *fiber.Ctx) error {
	filters := &models.DiscoverFilters{
		Query:    c.Query("q"),
		Category: models.ServerDiscoveryCategory(c.Query("category")),
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil {
			filters.Page = page
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil {
			filters.Limit = limit
		}
	}

	result, err := h.discoverableServerService.GetDiscoverableServers(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get servers",
		})
	}

	return c.JSON(result)
}

// GetFeaturedServers returns featured/recommended servers
// GET /api/v1/servers/discover/featured
func (h *DiscoverableServerHandler) GetFeaturedServers(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	servers, err := h.discoverableServerService.GetFeaturedServers(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get featured servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
	})
}

// GetServerDetail returns details about a discoverable server
// GET /api/v1/servers/:id
func (h *DiscoverableServerHandler) GetServerDetail(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	server, err := h.discoverableServerService.GetServerByID(c.Context(), id)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound || err == services.ErrServerNotPublic {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get server",
		})
	}

	// Get invite code
	detail, err := h.discoverableServerService.GetServerDetail(c.Context(), id)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound || err == services.ErrServerNotPublic {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get server details",
		})
	}

	// Don't expose internal fields
	response := fiber.Map{
		"id":           server.ID,
		"server_id":    server.ServerID,
		"name":         server.Name,
		"description": server.Description,
		"category":     server.Category,
		"icon_url":     server.IconURL,
		"banner_url":   server.BannerURL,
		"member_count": server.MemberCount,
		"is_verified":  server.IsVerified,
		"is_featured":  server.IsFeatured,
		"created_at":   server.CreatedAt,
	}

	if detail.InviteCode != nil {
		response["invite_code"] = detail.InviteCode
	}

	return c.JSON(response)
}

// JoinServer allows a user to join a discoverable server
// POST /api/v1/servers/:id/join
func (h *DiscoverableServerHandler) JoinServer(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	err = h.discoverableServerService.JoinServer(c.Context(), id, userID)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found",
			})
		}
		if err == services.ErrServerNotPublic {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found",
			})
		}
		if err == services.ErrAlreadyMember {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Already a member of this server",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to join server",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Successfully joined server",
		"server_id": id,
	})
}

// GetCategories returns all available discovery categories
// GET /api/v1/servers/categories
func (h *DiscoverableServerHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.discoverableServerService.GetCategories(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get categories",
		})
	}

	// If no categories exist yet, return default categories with zero counts
	if len(categories) == 0 {
		defaultCategories := []fiber.Map{
			{"name": "gaming", "slug": "gaming", "server_count": 0},
			{"name": "technology", "slug": "technology", "server_count": 0},
			{"name": "art", "slug": "art", "server_count": 0},
			{"name": "music", "slug": "music", "server_count": 0},
			{"name": "sports", "slug": "sports", "server_count": 0},
			{"name": "education", "slug": "education", "server_count": 0},
			{"name": "entertainment", "slug": "entertainment", "server_count": 0},
			{"name": "community", "slug": "community", "server_count": 0},
			{"name": "other", "slug": "other", "server_count": 0},
		}
		return c.JSON(fiber.Map{
			"categories": defaultCategories,
		})
	}

	return c.JSON(fiber.Map{
		"categories": categories,
	})
}
