// File: handlers/app_directory.go
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// AppDirectoryHandler handles app directory API endpoints
type AppDirectoryHandler struct {
	appService *services.AppDirectoryService
}

// NewAppDirectoryHandler creates a new app directory handler
func NewAppDirectoryHandler(appService *services.AppDirectoryService) *AppDirectoryHandler {
	return &AppDirectoryHandler{appService: appService}
}

// SetAppDirectoryHandler sets the app directory handler on the comprehensive handlers struct
// (called from main.go during initialization)
func (h *Handlers) SetAppDirectoryHandler(appService *services.AppDirectoryService) {
	h.AppDirectory = NewAppDirectoryHandler(appService)
}

// ListApps returns a list of apps with optional filtering
// @Summary List apps in the directory
// @Description Returns a list of approved apps, optionally filtered by category or search query
// @Tags Apps
// @Produce json
// @Param category query string false "Filter by category"
// @Param query query string false "Search query"
// @Param featured query bool false "Only show featured apps"
// @Param limit query int false "Number of results (default 20, max 100)"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} models.App
// @Failure 401 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps [get]
func (h *AppDirectoryHandler) ListApps(c *fiber.Ctx) error {
	req := &models.ListAppsRequest{
		Category: c.Query("category"),
		Query:    c.Query("query"),
		Featured: c.QueryBool("featured", false),
		Limit:    c.QueryInt("limit", 20),
		Offset:   c.QueryInt("offset", 0),
	}

	apps, err := h.appService.ListApps(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"apps":   apps,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

// GetApp returns a single app by ID
// @Summary Get app details
// @Description Returns detailed information about an app
// @Tags Apps
// @Produce json
// @Param id path string true "App ID"
// @Success 200 {object} models.App
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id} [get]
func (h *AppDirectoryHandler) GetApp(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	app, err := h.appService.GetApp(c.Context(), appID)
	if err != nil {
		if err == services.ErrAppNotFound {
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		}
		return err
	}

	return c.JSON(app)
}

// CreateApp creates a new app submission
// @Summary Submit a new app
// @Description Creates a new app for review (developers only)
// @Tags Apps
// @Accept json
// @Produce json
// @Param body body models.CreateAppRequest true "App data"
// @Success 201 {object} models.App
// @Failure 400 {object} HTTPError
// @Failure 401 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps [post]
func (h *AppDirectoryHandler) CreateApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	var req models.CreateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	// Basic validation
	if req.Name == "" || len(req.Name) < 2 || len(req.Name) > 100 {
		return fiber.NewError(fiber.StatusBadRequest, "name must be between 2 and 100 characters")
	}
	if req.Description == "" || len(req.Description) < 10 || len(req.Description) > 200 {
		return fiber.NewError(fiber.StatusBadRequest, "description must be between 10 and 200 characters")
	}
	if req.Category == "" {
		return fiber.NewError(fiber.StatusBadRequest, "category is required")
	}

	app, err := h.appService.SubmitApp(c.Context(), &req, userID)
	if err != nil {
		if err == services.ErrInvalidAppCategory {
			return fiber.NewError(fiber.StatusBadRequest, "invalid category")
		}
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(app)
}

// UpdateApp updates an existing app
// @Summary Update an app
// @Description Updates an existing app (developers only)
// @Tags Apps
// @Accept json
// @Produce json
// @Param id path string true "App ID"
// @Param body body models.UpdateAppRequest true "App update data"
// @Success 200 {object} models.App
// @Failure 400 {object} HTTPError
// @Failure 401 {object} HTTPError
// @Failure 403 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id} [patch]
func (h *AppDirectoryHandler) UpdateApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	var req models.UpdateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	app, err := h.appService.UpdateApp(c.Context(), appID, &req, userID)
	if err != nil {
		switch err {
		case services.ErrAppNotFound:
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		case services.ErrNotAppDeveloper:
			return fiber.NewError(fiber.StatusForbidden, "you are not a developer of this app")
		case services.ErrInvalidAppCategory:
			return fiber.NewError(fiber.StatusBadRequest, "invalid category")
		default:
			return err
		}
	}

	return c.JSON(app)
}

// DeleteApp deletes an app
// @Summary Delete an app
// @Description Deletes an app (owner only, must have no installations)
// @Tags Apps
// @Param id path string true "App ID"
// @Success 204
// @Failure 401 {object} HTTPError
// @Failure 403 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 409 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id} [delete]
func (h *AppDirectoryHandler) DeleteApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	err = h.appService.DeleteApp(c.Context(), appID, userID)
	if err != nil {
		switch err {
		case services.ErrAppNotFound:
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		case services.ErrNotAppOwner:
			return fiber.NewError(fiber.StatusForbidden, "only the app owner can delete")
		case services.ErrCannotDeleteApp:
			return fiber.NewError(fiber.StatusConflict, "cannot delete app with active installations")
		default:
			return err
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// InstallApp installs an app on a server
// @Summary Install an app
// @Description Installs an app on a server (server admin only)
// @Tags Apps
// @Param id path string true "App ID"
// @Param serverId path string true "Server ID"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} HTTPError
// @Failure 401 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 409 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id}/install/{serverId} [post]
func (h *AppDirectoryHandler) InstallApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	err = h.appService.InstallApp(c.Context(), appID, serverID, userID)
	if err != nil {
		switch err {
		case services.ErrAppNotFound:
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		case services.ErrAppNotApproved:
			return fiber.NewError(fiber.StatusBadRequest, "app is not approved")
		case services.ErrAlreadyInstalled:
			return fiber.NewError(fiber.StatusConflict, "app is already installed")
		default:
			return err
		}
	}

	return c.JSON(fiber.Map{"message": "app installed successfully"})
}

// UninstallApp uninstalls an app from a server
// @Summary Uninstall an app
// @Description Removes an app from a server (server admin only)
// @Tags Apps
// @Param id path string true "App ID"
// @Param serverId path string true "Server ID"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} HTTPError
// @Failure 401 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id}/install/{serverId} [delete]
func (h *AppDirectoryHandler) UninstallApp(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	serverID, err := uuid.Parse(c.Params("serverId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid server id")
	}

	err = h.appService.UninstallApp(c.Context(), appID, serverID)
	if err != nil {
		switch err {
		case services.ErrNotInstalled:
			return fiber.NewError(fiber.StatusNotFound, "app is not installed on this server")
		default:
			return err
		}
	}

	return c.JSON(fiber.Map{"message": "app uninstalled successfully"})
}

// ListAppReviews returns all reviews for an app
// @Summary List app reviews
// @Description Returns all reviews for an app
// @Tags Apps
// @Produce json
// @Param id path string true "App ID"
// @Param limit query int false "Number of results (default 20)"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} models.AppReview
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id}/reviews [get]
func (h *AppDirectoryHandler) ListAppReviews(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	reviews, err := h.appService.ListAppReviews(c.Context(), appID, limit, offset)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"reviews": reviews,
		"limit":   limit,
		"offset":  offset,
	})
}

// CreateReview creates a review for an app
// @Summary Create a review
// @Description Creates a review for an app (authenticated user only, one review per app)
// @Tags Apps
// @Accept json
// @Produce json
// @Param id path string true "App ID"
// @Param body body models.CreateReviewRequest true "Review data"
// @Success 201 {object} models.AppReview
// @Failure 400 {object} HTTPError
// @Failure 401 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 409 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id}/reviews [post]
func (h *AppDirectoryHandler) CreateReview(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	var req models.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Rating < 1 || req.Rating > 5 {
		return fiber.NewError(fiber.StatusBadRequest, "rating must be between 1 and 5")
	}

	review, err := h.appService.CreateReview(c.Context(), appID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrAppNotFound:
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		case services.ErrAppNotApproved:
			return fiber.NewError(fiber.StatusBadRequest, "cannot review unapproved apps")
		case services.ErrAlreadyReviewed:
			return fiber.NewError(fiber.StatusConflict, "you have already reviewed this app")
		default:
			return err
		}
	}

	return c.Status(fiber.StatusCreated).JSON(review)
}

// UpdateReview updates an existing review
// @Summary Update a review
// @Description Updates the authenticated user's review for an app
// @Tags Apps
// @Accept json
// @Produce json
// @Param id path string true "App ID"
// @Param reviewId path string true "Review ID"
// @Param body body models.UpdateReviewRequest true "Review update data"
// @Success 200 {object} models.AppReview
// @Failure 400 {object} HTTPError
// @Failure 401 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id}/reviews/{reviewId} [patch]
func (h *AppDirectoryHandler) UpdateReview(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	reviewID, err := uuid.Parse(c.Params("reviewId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid review id")
	}

	var req models.UpdateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	review, err := h.appService.UpdateReview(c.Context(), appID, reviewID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrReviewNotFound:
			return fiber.NewError(fiber.StatusNotFound, "review not found")
		case services.ErrNotAppDeveloper:
			return fiber.NewError(fiber.StatusForbidden, "you can only update your own reviews")
		default:
			return err
		}
	}

	return c.JSON(review)
}

// DeleteReview deletes a review
// @Summary Delete a review
// @Description Deletes the authenticated user's review for an app
// @Tags Apps
// @Param id path string true "App ID"
// @Param reviewId path string true "Review ID"
// @Success 204
// @Failure 401 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/apps/{id}/reviews/{reviewId} [delete]
func (h *AppDirectoryHandler) DeleteReview(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	reviewID, err := uuid.Parse(c.Params("reviewId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid review id")
	}

	err = h.appService.DeleteReview(c.Context(), appID, reviewID, userID)
	if err != nil {
		switch err {
		case services.ErrReviewNotFound:
			return fiber.NewError(fiber.StatusNotFound, "review not found")
		case services.ErrNotAppDeveloper:
			return fiber.NewError(fiber.StatusForbidden, "you can only delete your own reviews")
		default:
			return err
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ListDeveloperApps returns all apps for the authenticated developer
// @Summary List my apps
// @Description Returns all apps owned by the authenticated developer
// @Tags Developer
// @Produce json
// @Success 200 {array} models.App
// @Failure 401 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/developer/apps [get]
func (h *AppDirectoryHandler) ListDeveloperApps(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	apps, err := h.appService.ListDeveloperApps(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"apps": apps})
}

// GetDeveloperAnalytics returns analytics for the authenticated developer
// @Summary Get developer analytics
// @Description Returns analytics and insights for the developer's apps
// @Tags Developer
// @Produce json
// @Success 200 {object} models.AppDeveloperAnalytics
// @Failure 401 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/developer/analytics [get]
func (h *AppDirectoryHandler) GetDeveloperAnalytics(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	analytics, err := h.appService.GetDeveloperAnalytics(c.Context(), userID)
	if err != nil {
		return err
	}

	return c.JSON(analytics)
}

// ListCategories returns all app categories
// @Summary List app categories
// @Description Returns all available app categories
// @Tags Apps
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/apps/categories [get]
func (h *AppDirectoryHandler) ListCategories(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"categories": models.AppCategoryNames})
}

// GetMyReviewForApp returns the authenticated user's review for an app
// @Summary Get my review for an app
// @Description Returns the authenticated user's review for a specific app, if any
// @Tags Apps
// @Produce json
// @Param id path string true "App ID"
// @Success 200 {object} models.AppReview
// @Failure 401 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Router /api/v1/apps/{id}/reviews/@me [get]
func (h *AppDirectoryHandler) GetMyReviewForApp(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	appID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	review, err := h.appService.GetUserReviewForApp(c.Context(), appID, userID)
	if err != nil {
		return err
	}

	if review == nil {
		return fiber.NewError(fiber.StatusNotFound, "you have not reviewed this app")
	}

	return c.JSON(review)
}

// ApproveApp approves a pending app (admin only)
// @Summary Approve an app
// @Description Approves a pending app for listing in the directory
// @Tags Admin
// @Param appId path string true "App ID"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/admin/apps/{appId}/approve [post]
func (h *AppDirectoryHandler) ApproveApp(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	err = h.appService.ApproveApp(c.Context(), appID)
	if err != nil {
		switch err {
		case services.ErrAppNotFound:
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		case services.ErrInvalidStatus:
			return fiber.NewError(fiber.StatusBadRequest, "app is not in pending status")
		default:
			return err
		}
	}

	return c.JSON(fiber.Map{"message": "app approved"})
}

// RejectApp rejects a pending app (admin only)
// @Summary Reject an app
// @Description Rejects a pending app
// @Tags Admin
// @Param appId path string true "App ID"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/admin/apps/{appId}/reject [post]
func (h *AppDirectoryHandler) RejectApp(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	err = h.appService.RejectApp(c.Context(), appID, "")
	if err != nil {
		switch err {
		case services.ErrAppNotFound:
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		case services.ErrInvalidStatus:
			return fiber.NewError(fiber.StatusBadRequest, "app is not in pending status")
		default:
			return err
		}
	}

	return c.JSON(fiber.Map{"message": "app rejected"})
}

// SuspendApp suspends an approved app (admin only)
// @Summary Suspend an app
// @Description Suspends an approved app (makes it unavailable)
// @Tags Admin
// @Param appId path string true "App ID"
// @Success 200 {object} fiber.Map
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/admin/apps/{appId}/suspend [post]
func (h *AppDirectoryHandler) SuspendApp(c *fiber.Ctx) error {
	appID, err := uuid.Parse(c.Params("appId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid app id")
	}

	err = h.appService.SuspendApp(c.Context(), appID)
	if err != nil {
		switch err {
		case services.ErrAppNotFound:
			return fiber.NewError(fiber.StatusNotFound, "app not found")
		case services.ErrInvalidStatus:
			return fiber.NewError(fiber.StatusBadRequest, "app is not in approved status")
		default:
			return err
		}
	}

	return c.JSON(fiber.Map{"message": "app suspended"})
}
