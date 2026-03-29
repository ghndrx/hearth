package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// DiscoveryHandler handles discovery API endpoints
type DiscoveryHandler struct {
	discovery *services.DiscoveryService
	server    *services.ServerService
}

// NewDiscoveryHandler creates a new discovery handler
func NewDiscoveryHandler(discovery *services.DiscoveryService, server *services.ServerService) *DiscoveryHandler {
	return &DiscoveryHandler{
		discovery: discovery,
		server:    server,
	}
}

// GetFeaturedServers returns featured servers
// GET /api/v1/discovery/featured
func (h *DiscoveryHandler) GetFeaturedServers(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	servers, err := h.discovery.GetFeaturedServers(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get featured servers",
		})
	}

	return c.JSON(servers)
}

// GetCategories returns all discovery categories
// GET /api/v1/discovery/categories
func (h *DiscoveryHandler) GetCategories(c *fiber.Ctx) error {
	categories, err := h.discovery.GetCategories(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get categories",
		})
	}

	return c.JSON(categories)
}

// SearchServers searches servers with filters
// GET /api/v1/discovery/search
func (h *DiscoveryHandler) SearchServers(c *fiber.Ctx) error {
	filters := &models.DiscoveryFilters{
		Query:     c.Query("q"),
		Category:  models.ServerCategory(c.Query("category")),
		Region:    c.Query("region"),
		Language:  c.Query("language"),
		SortBy:    c.Query("sort", "members"),
		SortOrder: c.Query("order", "desc"),
	}

	if minMembers := c.Query("min_members"); minMembers != "" {
		filters.MinMembers, _ = strconv.Atoi(minMembers)
	}
	if maxMembers := c.Query("max_members"); maxMembers != "" {
		filters.MaxMembers, _ = strconv.Atoi(maxMembers)
	}
	if featured := c.Query("featured"); featured != "" {
		b := featured == "true"
		filters.Featured = &b
	}
	if limit := c.Query("limit"); limit != "" {
		filters.Limit, _ = strconv.Atoi(limit)
	} else {
		filters.Limit = 25
	}
	if offset := c.Query("offset"); offset != "" {
		filters.Offset, _ = strconv.Atoi(offset)
	}

	servers, total, err := h.discovery.SearchServers(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
		"total":   total,
		"limit":   filters.Limit,
		"offset":  filters.Offset,
	})
}

// GetRecommendations returns personalized recommendations
// GET /api/v1/discovery/recommendations
func (h *DiscoveryHandler) GetRecommendations(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	servers, err := h.discovery.GetRecommendations(c.Context(), userID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get recommendations",
		})
	}

	return c.JSON(servers)
}

// GetServersByCategory returns servers for a category
// GET /api/v1/discovery/categories/:slug
func (h *DiscoveryHandler) GetServersByCategory(c *fiber.Ctx) error {
	category := c.Params("slug")
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	servers, total, err := h.discovery.GetServersByCategory(c.Context(), category, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetServerListing returns discovery listing for a server
// GET /api/v1/discovery/servers/:serverId
func (h *DiscoveryHandler) GetServerListing(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	listing, err := h.discovery.GetDiscoveryListingWithDetails(c.Context(), serverID)
	if err != nil {
		if err == services.ErrDiscoveryListingNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found in discovery",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get listing",
		})
	}

	return c.JSON(listing)
}

// SubmitForDiscovery submits a server for discovery
// POST /api/v1/servers/:serverId/listing
func (h *DiscoveryHandler) SubmitForDiscovery(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	var req models.SubmitDiscoveryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.ShortDescription == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Short description is required",
		})
	}
	if len(req.Categories) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one category is required",
		})
	}

	err = h.discovery.SubmitForDiscovery(c.Context(), serverID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrServerNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found",
			})
		case services.ErrDiscoveryNotOwner:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only server owner can submit for discovery",
			})
		case services.ErrDiscoveryListingExists:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Server already has a discovery listing",
			})
		case services.ErrInvalidCategory:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid category provided",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to submit for discovery",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Server submitted for discovery",
		"status":  "pending",
	})
}

// UpdateListing updates a discovery listing
// PATCH /api/v1/servers/:serverId/listing
func (h *DiscoveryHandler) UpdateListing(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	userID := c.Locals("userID").(uuid.UUID)

	var req models.UpdateDiscoveryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err = h.discovery.UpdateListing(c.Context(), serverID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrServerNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found",
			})
		case services.ErrDiscoveryNotOwner:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Only server owner can update listing",
			})
		case services.ErrDiscoveryListingNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Discovery listing not found",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update listing",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "Listing updated successfully",
	})
}

// ReportServer reports a server in discovery
// POST /api/v1/discovery/report
func (h *DiscoveryHandler) ReportServer(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var body struct {
		ServerID string `json:"server_id"`
		Reason   string `json:"reason"`
		Details  string `json:"details"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if body.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Report reason is required",
		})
	}

	serverID, err := uuid.Parse(body.ServerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	reportReq := &models.ReportServerRequest{
		Reason:  body.Reason,
		Details: body.Details,
	}

	err = h.discovery.ReportServer(c.Context(), serverID, userID, reportReq)
	if err != nil {
		if err == services.ErrDiscoveryListingNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found in discovery",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to submit report",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Report submitted successfully",
	})
}

// ApproveListing approves a listing (admin)
// POST /api/v1/admin/discovery/:listingId/approve
func (h *DiscoveryHandler) ApproveListing(c *fiber.Ctx) error {
	listingID, err := uuid.Parse(c.Params("listingId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	adminID := c.Locals("userID").(uuid.UUID)

	err = h.discovery.ApproveListing(c.Context(), listingID, adminID)
	if err != nil {
		if err == services.ErrDiscoveryListingNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Listing not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to approve listing",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Listing approved",
	})
}

// RejectListing rejects a listing (admin)
// POST /api/v1/admin/discovery/:listingId/reject
func (h *DiscoveryHandler) RejectListing(c *fiber.Ctx) error {
	listingID, err := uuid.Parse(c.Params("listingId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	adminID := c.Locals("userID").(uuid.UUID)

	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err = h.discovery.RejectListing(c.Context(), listingID, adminID, body.Reason)
	if err != nil {
		if err == services.ErrDiscoveryListingNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Listing not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reject listing",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Listing rejected",
	})
}

// SetFeatured marks a server as featured (admin)
// POST /api/v1/admin/discovery/:listingId/featured
func (h *DiscoveryHandler) SetFeatured(c *fiber.Ctx) error {
	listingID, err := uuid.Parse(c.Params("listingId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	var body struct {
		Featured bool `json:"featured"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err = h.discovery.SetFeatured(c.Context(), listingID, body.Featured)
	if err != nil {
		if err == services.ErrDiscoveryListingNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Listing not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update featured status",
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Featured status updated",
		"is_featured": body.Featured,
	})
}

// GetTrendingServers returns trending servers
// GET /api/v1/discovery/trending
func (h *DiscoveryHandler) GetTrendingServers(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	servers, err := h.discovery.GetTrendingServers(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get trending servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
		"limit":   limit,
	})
}

// GetDiscoveryStats returns discovery statistics
// GET /api/v1/discovery/stats
func (h *DiscoveryHandler) GetDiscoveryStats(c *fiber.Ctx) error {
	stats, err := h.discovery.GetDiscoveryStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get discovery stats",
		})
	}

	return c.JSON(stats)
}

// GetPopularTags returns popular discovery tags
// GET /api/v1/discovery/tags
func (h *DiscoveryHandler) GetPopularTags(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	tags, err := h.discovery.GetPopularTags(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get popular tags",
		})
	}

	return c.JSON(fiber.Map{
		"tags":  tags,
		"limit": limit,
	})
}

// GetServersByTags returns servers matching specific tags
// GET /api/v1/discovery/tags/servers
func (h *DiscoveryHandler) GetServersByTags(c *fiber.Ctx) error {
	tagsParam := c.Query("tags")
	if tagsParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tags parameter is required",
		})
	}

	// Parse comma-separated tags
	tags := strings.Split(tagsParam, ",")
	for i, tag := range tags {
		tags[i] = strings.TrimSpace(tag)
	}

	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	servers, total, err := h.discovery.GetServersByTags(c.Context(), tags, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get servers by tags",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetDiscoveryPage returns the full discovery page data
// GET /api/v1/discovery/page
func (h *DiscoveryHandler) GetDiscoveryPage(c *fiber.Ctx) error {
	featuredLimit, _ := strconv.Atoi(c.Query("featured_limit", "5"))
	trendingLimit, _ := strconv.Atoi(c.Query("trending_limit", "10"))

	var userID uuid.UUID
	if id, ok := c.Locals("userID").(uuid.UUID); ok {
		userID = id
	}

	page, err := h.discovery.GetDiscoveryPage(c.Context(), userID, featuredLimit, trendingLimit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get discovery page",
		})
	}

	return c.JSON(page)
}

// SearchServersEnhanced searches servers with enhanced filters
// GET /api/v1/discovery/search
func (h *DiscoveryHandler) SearchServersEnhanced(c *fiber.Ctx) error {
	filters := &models.DiscoveryFilters{
		Query:     c.Query("q"),
		Category:  models.ServerCategory(c.Query("category")),
		Region:    c.Query("region"),
		Language:  c.Query("language"),
		SortBy:    c.Query("sort", "members"),
		SortOrder: c.Query("order", "desc"),
	}

	if minMembers := c.Query("min_members"); minMembers != "" {
		filters.MinMembers, _ = strconv.Atoi(minMembers)
	}
	if maxMembers := c.Query("max_members"); maxMembers != "" {
		filters.MaxMembers, _ = strconv.Atoi(maxMembers)
	}
	if featured := c.Query("featured"); featured != "" {
		b := featured == "true"
		filters.Featured = &b
	}
	if tags := c.Query("tags"); tags != "" {
		// Parse comma-separated tags
		tagList := strings.Split(tags, ",")
		for i, tag := range tagList {
			tagList[i] = strings.TrimSpace(tag)
		}
		filters.Tags = tagList
	}
	if onlineOnly := c.Query("online_only"); onlineOnly == "true" {
		filters.OnlineOnly = true
	}
	if limit := c.Query("limit"); limit != "" {
		filters.Limit, _ = strconv.Atoi(limit)
	} else {
		filters.Limit = 25
	}
	if offset := c.Query("offset"); offset != "" {
		filters.Offset, _ = strconv.Atoi(offset)
	}

	servers, total, err := h.discovery.SearchServersEnhanced(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
		"total":   total,
		"limit":   filters.Limit,
		"offset":  filters.Offset,
	})
}
