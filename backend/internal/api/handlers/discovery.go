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
	discovery                 *services.DiscoveryService
	discoverableServerService *services.DiscoverableServerService
	server                    *services.ServerService
}

// NewDiscoveryHandler creates a new discovery handler
func NewDiscoveryHandler(discovery *services.DiscoveryService, discoverableServerService *services.DiscoverableServerService, server *services.ServerService) *DiscoveryHandler {
	return &DiscoveryHandler{
		discovery:                 discovery,
		discoverableServerService: discoverableServerService,
		server:                    server,
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
		if v, err := strconv.Atoi(minMembers); err == nil {
			filters.MinMembers = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid min_members parameter",
			})
		}
	}
	if maxMembers := c.Query("max_members"); maxMembers != "" {
		if v, err := strconv.Atoi(maxMembers); err == nil {
			filters.MaxMembers = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid max_members parameter",
			})
		}
	}
	if featured := c.Query("featured"); featured != "" {
		b := featured == "true"
		filters.Featured = &b
	}
	if limit := c.Query("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			filters.Limit = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid limit parameter",
			})
		}
	} else {
		filters.Limit = 25
	}
	if offset := c.Query("offset"); offset != "" {
		if v, err := strconv.Atoi(offset); err == nil {
			filters.Offset = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid offset parameter",
			})
		}
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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
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

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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

// requireAdmin checks if the authenticated user has admin privileges
func (h *DiscoveryHandler) requireAdmin(c *fiber.Ctx) error {
	isAdmin, ok := c.Locals("isAdmin").(bool)
	if !ok || !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}
	return nil
}

// ApproveListing approves a listing (admin)
// POST /api/v1/admin/discovery/:listingId/approve
func (h *DiscoveryHandler) ApproveListing(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	listingID, err := uuid.Parse(c.Params("listingId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	listingID, err := uuid.Parse(c.Params("listingId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid listing ID",
		})
	}

	adminID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
	if err := h.requireAdmin(c); err != nil {
		return err
	}

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

// GetPublicServers returns public server directory with search, category filter, and pagination
// GET /api/v1/servers (public - no auth required)
func (h *DiscoveryHandler) GetPublicServers(c *fiber.Ctx) error {
	filters := &models.DiscoveryFilters{
		Query:     c.Query("q"),
		Category:  models.ServerCategory(c.Query("category")),
		SortBy:    c.Query("sort", "popular"), // popular, new, active
		SortOrder: c.Query("order", "desc"),
	}

	// Parse pagination
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	filters.Limit = limit
	filters.Offset = offset

	// Validate sort parameter - map to internal sort values
	switch filters.SortBy {
	case "popular":
		filters.SortBy = "members" // Sort by member count
	case "new":
		filters.SortBy = "newest" // Sort by creation date
	case "active":
		filters.SortBy = "engagement" // Sort by engagement score
	default:
		filters.SortBy = "members"
	}

	// Validate category if provided
	if filters.Category != "" {
		validCategories := map[models.ServerCategory]bool{
			models.CategoryGaming:        true,
			models.CategoryMusic:         true,
			models.CategoryTechnology:    true,
			models.CategoryArt:           true,
			models.CategoryEducation:     true,
			models.CategoryScience:       true,
			models.CategoryEntertainment: true,
			models.CategorySocial:        true,
			models.CategorySports:        true,
			models.CategoryAnime:         true,
			models.CategoryFashion:       true,
			models.CategoryFood:          true,
			models.CategoryBusiness:      true,
			models.CategoryLanguage:      true,
		}
		if !validCategories[filters.Category] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid category",
			})
		}
	}

	servers, total, err := h.discovery.SearchServers(c.Context(), filters)
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

// GetPublicServer returns a single server from the public directory
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
		if v, err := strconv.Atoi(minMembers); err == nil {
			filters.MinMembers = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid min_members parameter",
			})
		}
	}
	if maxMembers := c.Query("max_members"); maxMembers != "" {
		if v, err := strconv.Atoi(maxMembers); err == nil {
			filters.MaxMembers = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid max_members parameter",
			})
		}
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
		if v, err := strconv.Atoi(limit); err == nil {
			filters.Limit = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid limit parameter",
			})
		}
	} else {
		filters.Limit = 25
	}
	if offset := c.Query("offset"); offset != "" {
		if v, err := strconv.Atoi(offset); err == nil {
			filters.Offset = v
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid offset parameter",
			})
		}
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

// GET /api/v1/servers/:id (public - no auth required)
func (h *DiscoveryHandler) GetPublicServer(c *fiber.Ctx) error {
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	listing, err := h.discovery.GetDiscoveryListingWithDetails(c.Context(), serverID)
	if err != nil {
		if err == services.ErrDiscoveryListingNotFound || err == services.ErrServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get server",
		})
	}

	// Check if listing is approved and visible
	listingCheck, listingErr := h.discovery.GetServerListing(c.Context(), serverID)
	if listingErr != nil || listingCheck == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Server not found",
		})
	}
	if listingCheck.ApprovalStatus != models.ApprovalApproved || !listingCheck.IsListed {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Server not found",
		})
	}

	return c.JSON(listing)
}

// GetPublicCategories returns all public discovery categories
// GET /api/v1/servers/categories (public - no auth required)
func (h *DiscoveryHandler) GetPublicCategories(c *fiber.Ctx) error {
	categories, err := h.discovery.GetCategories(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get categories",
		})
	}

	return c.JSON(fiber.Map{
		"categories": categories,
	})
}

// GetDiscoverableServers returns paginated discoverable servers with optional search and category filter
// GET /api/v1/servers/discover
func (h *DiscoveryHandler) GetDiscoverableServers(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetFeaturedServersDS(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetTrendingServersDS(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetRecommendedServers(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) SearchServersEnhancedDS(c *fiber.Ctx) error {
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

// GetDiscovery is the main public server directory endpoint listing servers that have opted into being discoverable
// GET /api/v1/discovery
func (h *DiscoveryHandler) GetDiscovery(c *fiber.Ctx) error {
	featuredLimit := c.QueryInt("featured_limit", 5)
	trendingLimit := c.QueryInt("trending_limit", 10)
	recommendedLimit := c.QueryInt("recommended_limit", 10)

	// Parse pagination for main server listing
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 25)
	if limit < 1 || limit > 100 {
		limit = 25
	}

	// Parse filters for server listing
	filters := &models.DiscoverFilters{
		Query:    c.Query("q"),
		Category: models.ServerDiscoveryCategory(c.Query("category")),
		Page:     page,
		Limit:    limit,
	}

	// User is optional for home page
	var userID uuid.UUID
	if id, ok := c.Locals("userID").(uuid.UUID); ok {
		userID = id
	}

	// Fetch home page data and paginated servers concurrently
	homePage, err := h.discoverableServerService.GetDiscoveryHomePage(c.Context(), userID, featuredLimit, trendingLimit, recommendedLimit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get discovery page",
		})
	}

	// Get paginated server listings
	serversResult, err := h.discoverableServerService.GetDiscoverableServers(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get servers",
		})
	}

	return c.JSON(fiber.Map{
		"featured":     homePage.Featured,
		"trending":     homePage.Trending,
		"recommended":  homePage.Recommended,
		"categories":   homePage.Categories,
		"popular_tags": homePage.PopularTags,
		"stats":        homePage.Stats,
		"servers":      serversResult.Servers,
		"total":        serversResult.Total,
		"page":         serversResult.Page,
		"limit":        serversResult.Limit,
		"total_pages":  serversResult.TotalPages,
	})
}

// GetDiscoveryHomePage returns the full discovery home page data
// GET /api/v1/servers/discover/home
func (h *DiscoveryHandler) GetDiscoveryHomePage(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetCategoriesWithStats(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetPopularTagsDS(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetDiscoveryStatsDS(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetSearchSuggestions(c *fiber.Ctx) error {
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
func (h *DiscoveryHandler) GetServerDetail(c *fiber.Ctx) error {
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
		"description":  server.Description,
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
func (h *DiscoveryHandler) JoinServer(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

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
		"message":   "Successfully joined server",
		"server_id": id,
	})
}

// RegisterServer registers a server for discovery (requires auth, server owner only)
// POST /api/v1/servers/:serverId/discover
func (h *DiscoveryHandler) RegisterServer(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	serverIDStr := c.Params("serverId")
	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	var req models.RegisterServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if req.Category == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category is required",
		})
	}

	ds, err := h.discoverableServerService.RegisterServer(c.Context(), serverID, userID, &req)
	if err != nil {
		if err == services.ErrServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Server not found"})
		}
		if err == services.ErrNotServerOwner {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the server owner can register for discovery"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(ds)
}

// UpdateRegisteredServer updates a server's discovery listing (requires auth, server owner only)
// PATCH /api/v1/servers/discover/:id
func (h *DiscoveryHandler) UpdateRegisteredServer(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	var req models.UpdateDiscoverableServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	ds, err := h.discoverableServerService.UpdateRegisteredServer(c.Context(), id, userID, &req)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Listing not found"})
		}
		if err == services.ErrNotServerOwner {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the server owner can update the listing"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(ds)
}

// DeleteRegisteredServer removes a server from discovery (requires auth, server owner only)
// DELETE /api/v1/servers/discover/:id
func (h *DiscoveryHandler) DeleteRegisteredServer(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	err = h.discoverableServerService.DeleteRegisteredServer(c.Context(), id, userID)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Listing not found"})
		}
		if err == services.ErrNotServerOwner {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Only the server owner can remove the listing"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to remove listing"})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// GetCategories returns all available discovery categories
// GET /api/v1/servers/categories
func (h *DiscoveryHandler) GetCategoriesDS(c *fiber.Ctx) error {
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

// GetDirectory is the public server directory endpoint with search, categories, and pagination
// GET /api/v1/directory
func (h *DiscoveryHandler) GetDirectory(c *fiber.Ctx) error {
	// Parse pagination
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 25)
	if limit < 1 || limit > 100 {
		limit = 25
	}

	// Parse filters
	filters := &models.DiscoverFilters{
		Query:    c.Query("q"),
		Category: models.ServerDiscoveryCategory(c.Query("category")),
		Page:     page,
		Limit:    limit,
	}

	// Validate category if provided
	if filters.Category != "" && !models.IsValidCategory(string(filters.Category)) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category. Valid categories: gaming, technology, art, music, sports, education, entertainment, community, other",
		})
	}

	// Parse sort options
	sort := c.Query("sort", "popular")
	sortOrder := c.Query("order", "desc")

	// Get paginated server listings
	serversResult, err := h.discoverableServerService.GetDiscoverableServers(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get server directory",
		})
	}

	// Get categories for browsing
	categories, catErr := h.discoverableServerService.GetCategories(c.Context())
	if catErr != nil {
		categories = []*models.CategoryInfo{}
	}

	// Get featured servers
	featured, featErr := h.discoverableServerService.GetFeaturedServers(c.Context(), 5)
	if featErr != nil {
		featured = []*models.DiscoverableFeaturedServer{}
	}

	return c.JSON(fiber.Map{
		"servers":     serversResult.Servers,
		"total":       serversResult.Total,
		"page":        serversResult.Page,
		"limit":       serversResult.Limit,
		"total_pages": serversResult.TotalPages,
		"categories":  categories,
		"featured":    featured,
		"sort":        sort,
		"order":       sortOrder,
	})
}

// AdminApproveServer approves a server for the public directory (admin only)
// POST /api/v1/admin/directory/:id/approve
func (h *DiscoveryHandler) AdminApproveServer(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	err = h.discoverableServerService.SetServerPublicStatus(c.Context(), id, true)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found in directory",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to approve server",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Server approved for public directory",
		"is_public": true,
	})
}

// AdminRejectServer removes a server from the public directory (admin only)
// POST /api/v1/admin/directory/:id/reject
func (h *DiscoveryHandler) AdminRejectServer(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	err = h.discoverableServerService.SetServerPublicStatus(c.Context(), id, false)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found in directory",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reject server",
		})
	}

	return c.JSON(fiber.Map{
		"message":   "Server removed from public directory",
		"is_public": false,
	})
}

// AdminFeatureServer marks/unmarks a server as featured (admin only)
// POST /api/v1/admin/directory/:id/feature
func (h *DiscoveryHandler) AdminFeatureServer(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
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

	server, err := h.discoverableServerService.GetServerByID(c.Context(), id)
	if err != nil {
		if err == services.ErrDiscoverableServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Server not found in directory",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get server",
		})
	}

	server.IsFeatured = body.Featured
	if err := h.discoverableServerService.SetServerPublicStatus(c.Context(), id, server.IsPublic); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update featured status",
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Featured status updated",
		"is_featured": body.Featured,
	})
}

// TrackDiscoveryActivity records a discovery activity event
// POST /api/v1/directory/:id/track
func (h *DiscoveryHandler) TrackDiscoveryActivity(c *fiber.Ctx) error {
	idStr := c.Params("id")
	serverID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	var body struct {
		ActivityType string `json:"activity_type"`
		Source       string `json:"source"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var userID *uuid.UUID
	if id, ok := c.Locals("userID").(uuid.UUID); ok {
		userID = &id
	}

	err = h.discoverableServerService.TrackActivity(c.Context(), serverID, userID, body.ActivityType, body.Source)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to track activity",
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
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
