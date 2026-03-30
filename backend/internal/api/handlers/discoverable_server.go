package handlers

import (
	"strconv"
	"strings"

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

// GetTrendingServers returns trending servers
// GET /api/v1/servers/discover/trending
func (h *DiscoverableServerHandler) GetTrendingServers(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	servers, err := h.discoverableServerService.GetTrendingServers(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get trending servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
	})
}

// GetRecommendedServers returns personalized recommendations for authenticated user
// GET /api/v1/servers/discover/recommended
func (h *DiscoverableServerHandler) GetRecommendedServers(c *fiber.Ctx) error {
	// Check if user is authenticated
	userIDValue := c.Locals("userID")
	if userIDValue == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required for recommendations",
		})
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user authentication",
		})
	}

	limit := c.QueryInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	servers, err := h.discoverableServerService.GetRecommendedServers(c.Context(), userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get recommendations",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
	})
}

// SearchServersEnhanced performs enhanced search on servers
// GET /api/v1/servers/discover/search
func (h *DiscoverableServerHandler) SearchServersEnhanced(c *fiber.Ctx) error {
	req := &models.DiscoverySearchRequest{
		Query:     c.Query("q"),
		Category:  models.ServerDiscoveryCategory(c.Query("category")),
		SortBy:    c.Query("sort", "popular"),
		SortOrder: c.Query("order", "desc"),
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, _ := strconv.Atoi(pageStr)
		req.Page = page
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, _ := strconv.Atoi(limitStr)
		req.Limit = limit
	}

	// Parse comma-separated categories
	if categories := c.Query("categories"); categories != "" {
		catList := parseCommaSeparated(categories)
		req.Categories = make([]models.ServerDiscoveryCategory, len(catList))
		for i, cat := range catList {
			req.Categories[i] = models.ServerDiscoveryCategory(cat)
		}
	}

	// Parse comma-separated tags
	if tags := c.Query("tags"); tags != "" {
		req.Tags = parseCommaSeparated(tags)
	}

	result, err := h.discoverableServerService.SearchServersEnhanced(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search servers",
		})
	}

	return c.JSON(result)
}

// GetDiscoveryHomePage returns the full discovery home page data
// GET /api/v1/servers/discover/home
func (h *DiscoverableServerHandler) GetDiscoveryHomePage(c *fiber.Ctx) error {
	featuredLimit := c.QueryInt("featured_limit", 5)
	trendingLimit := c.QueryInt("trending_limit", 10)
	recommendedLimit := c.QueryInt("recommended_limit", 10)

	// User is optional for home page
	var userID uuid.UUID
	if id, ok := c.Locals("userID").(uuid.UUID); ok {
		userID = id
	}

	page, err := h.discoverableServerService.GetDiscoveryHomePage(c.Context(), userID, featuredLimit, trendingLimit, recommendedLimit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get discovery page",
		})
	}

	return c.JSON(page)
}

// GetCategoriesWithStats returns categories with statistics
// GET /api/v1/servers/discover/categories/stats
func (h *DiscoverableServerHandler) GetCategoriesWithStats(c *fiber.Ctx) error {
	categories, err := h.discoverableServerService.GetCategoriesWithStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get categories",
		})
	}

	return c.JSON(fiber.Map{
		"categories": categories,
	})
}

// GetPopularTags returns popular discovery tags
// GET /api/v1/servers/discover/tags
func (h *DiscoverableServerHandler) GetPopularTags(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)

	tags, err := h.discoverableServerService.GetPopularTags(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get tags",
		})
	}

	return c.JSON(fiber.Map{
		"tags": tags,
	})
}

// GetDiscoveryStats returns overall discovery statistics
// GET /api/v1/servers/discover/stats
func (h *DiscoverableServerHandler) GetDiscoveryStats(c *fiber.Ctx) error {
	stats, err := h.discoverableServerService.GetDiscoveryStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get stats",
		})
	}

	return c.JSON(stats)
}

// GetSearchSuggestions returns search suggestions
// GET /api/v1/servers/discover/suggestions
func (h *DiscoverableServerHandler) GetSearchSuggestions(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(fiber.Map{
			"suggestions": []interface{}{},
		})
	}

	limit := c.QueryInt("limit", 10)

	suggestions, err := h.discoverableServerService.GetSearchSuggestions(c.Context(), query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get suggestions",
		})
	}

	return c.JSON(fiber.Map{
		"suggestions": suggestions,
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
		"message":  "Successfully joined server",
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

// Helper function to parse comma-separated values
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
