package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// Mocks

type mockContentSafetyService struct {
	createContentFilterFunc    func(ctx context.Context, serverID, userID uuid.UUID, req *models.CreateContentFilterRequest) (*models.ContentFilter, error)
	getContentFilterFunc      func(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error)
	getServerFiltersFunc      func(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error)
	updateContentFilterFunc   func(ctx context.Context, id uuid.UUID, req *models.UpdateContentFilterRequest) (*models.ContentFilter, error)
	deleteContentFilterFunc   func(ctx context.Context, id uuid.UUID) error
	createAgeVerifyFunc       func(ctx context.Context, serverID, userID uuid.UUID, req *models.CreateAgeVerificationRequest) (*models.AgeVerificationSetting, error)
	getAgeVerifyFunc          func(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error)
	updateAgeVerifyFunc       func(ctx context.Context, serverID uuid.UUID, req *models.UpdateAgeVerificationRequest) (*models.AgeVerificationSetting, error)
	deleteAgeVerifyFunc       func(ctx context.Context, id uuid.UUID) error
	getUserPrefsFunc          func(ctx context.Context, userID uuid.UUID) (*models.UserContentPreference, error)
	updateUserPrefsFunc       func(ctx context.Context, userID uuid.UUID, req *models.UpdateUserContentPreferenceRequest) (*models.UserContentPreference, error)
	getServerSafetyFunc       func(ctx context.Context, serverID uuid.UUID) (*models.ContentSafetySettings, error)
	scanContentFunc           func(ctx context.Context, serverID, channelID, userID uuid.UUID, userRoles []uuid.UUID, content string) (*models.ContentScanResult, error)
}

type mockContentSafetyServerService struct {
	getMemberFunc func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
}

func setupContentSafetyTestApp(svcMock *mockContentSafetyService, serverSvcMock *mockContentSafetyServerService) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID")
		if userID != "" {
			uid, err := uuid.Parse(userID)
			if err == nil {
				c.Locals("userID", uid)
			}
		}
		return c.Next()
	})

	// List content filters
	app.Get("/servers/:id/content-filters", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if serverSvcMock.getMemberFunc != nil {
			_, err = serverSvcMock.getMemberFunc(c.Context(), serverID, userID)
			if err != nil {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not a member"})
			}
		}
		filters, err := svcMock.getServerFiltersFunc(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.JSON(filters)
	})

	// Create content filter
	app.Post("/servers/:id/content-filters", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var req models.CreateContentFilterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}
		filter, err := svcMock.createContentFilterFunc(c.Context(), serverID, userID, &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.Status(fiber.StatusCreated).JSON(filter)
	})

	// Get content filter
	app.Get("/content-filters/:id", func(c *fiber.Ctx) error {
		filterID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid filter id"})
		}
		filter, err := svcMock.getContentFilterFunc(c.Context(), filterID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		if filter == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.JSON(filter)
	})

	// Update content filter
	app.Patch("/content-filters/:id", func(c *fiber.Ctx) error {
		filterID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid filter id"})
		}
		var req models.UpdateContentFilterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}
		filter, err := svcMock.updateContentFilterFunc(c.Context(), filterID, &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.JSON(filter)
	})

	// Delete content filter
	app.Delete("/content-filters/:id", func(c *fiber.Ctx) error {
		filterID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid filter id"})
		}
		err = svcMock.deleteContentFilterFunc(c.Context(), filterID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Get age verification
	app.Get("/servers/:id/age-verification", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		settings, err := svcMock.getAgeVerifyFunc(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		if settings == nil {
			return c.JSON(fiber.Map{"server_id": serverID, "enabled": false})
		}
		return c.JSON(settings)
	})

	// Create age verification
	app.Put("/servers/:id/age-verification", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var req models.CreateAgeVerificationRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}
		settings, err := svcMock.createAgeVerifyFunc(c.Context(), serverID, userID, &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.Status(fiber.StatusCreated).JSON(settings)
	})

	// Get user content preferences
	app.Get("/users/@me/content-preferences", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		prefs, err := svcMock.getUserPrefsFunc(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.JSON(prefs)
	})

	// Update user content preferences
	app.Put("/users/@me/content-preferences", func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		var req models.UpdateUserContentPreferenceRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}
		prefs, err := svcMock.updateUserPrefsFunc(c.Context(), userID, &req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.JSON(prefs)
	})

	// Get server safety settings
	app.Get("/servers/:id/content-safety", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		settings, err := svcMock.getServerSafetyFunc(c.Context(), serverID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.JSON(settings)
	})

	// Test content scan
	app.Post("/servers/:id/content-safety/test", func(c *fiber.Ctx) error {
		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid server id"})
		}
		var req struct {
			Content string `json:"content"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
		}
		userID, _ := c.Locals("userID").(uuid.UUID)
		result, err := svcMock.scanContentFunc(c.Context(), serverID, serverID, userID, nil, req.Content)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		return c.JSON(result)
	})

	return app
}

func TestContentSafetyHandler_ListContentFilters(t *testing.T) {
	svcMock := &mockContentSafetyService{
		getServerFiltersFunc: func(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error) {
			return []*models.ContentFilter{
				{
					ID:        uuid.New(),
					ServerID:  serverID,
					Name:      "Test Filter",
					Type:      models.FilterTypeNSFW,
					Enabled:   true,
					Threshold: models.NSFWThresholdMedium,
					Action:    models.FilterActionBlock,
				},
			}, nil
		},
	}
	serverSvcMock := &mockContentSafetyServerService{
		getMemberFunc: func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
			return &models.Member{UserID: userID}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, serverSvcMock)

	serverID := uuid.New()
	userID := uuid.New()

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/content-filters", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var filters []*models.ContentFilter
	err = json.NewDecoder(resp.Body).Decode(&filters)
	assert.NoError(t, err)
	assert.Len(t, filters, 1)
	assert.Equal(t, "Test Filter", filters[0].Name)
}

func TestContentSafetyHandler_CreateContentFilter(t *testing.T) {
	svcMock := &mockContentSafetyService{
		createContentFilterFunc: func(ctx context.Context, serverID, userID uuid.UUID, req *models.CreateContentFilterRequest) (*models.ContentFilter, error) {
			return &models.ContentFilter{
				ID:        uuid.New(),
				ServerID:  serverID,
				Name:      req.Name,
				Type:      req.Type,
				Enabled:   true,
				Threshold: req.Threshold,
				Action:    req.Action,
			}, nil
		},
	}
	serverSvcMock := &mockContentSafetyServerService{
		getMemberFunc: func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
			return &models.Member{UserID: userID}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, serverSvcMock)

	serverID := uuid.New()
	userID := uuid.New()

	reqBody := models.CreateContentFilterRequest{
		Name:   "New Filter",
		Type:   models.FilterTypeNSFW,
		Action: models.FilterActionBlock,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/content-filters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var filter models.ContentFilter
	err = json.NewDecoder(resp.Body).Decode(&filter)
	assert.NoError(t, err)
	assert.Equal(t, "New Filter", filter.Name)
}

func TestContentSafetyHandler_GetContentFilter(t *testing.T) {
	filterID := uuid.New()
	svcMock := &mockContentSafetyService{
		getContentFilterFunc: func(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error) {
			if id == filterID {
				return &models.ContentFilter{
					ID:        filterID,
					Name:      "Test Filter",
					Type:      models.FilterTypeNSFW,
					Enabled:   true,
					Threshold: models.NSFWThresholdMedium,
					Action:    models.FilterActionBlock,
				}, nil
			}
			return nil, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	req := httptest.NewRequest("GET", "/content-filters/"+filterID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var filter models.ContentFilter
	err = json.NewDecoder(resp.Body).Decode(&filter)
	assert.NoError(t, err)
	assert.Equal(t, filterID, filter.ID)
}

func TestContentSafetyHandler_GetContentFilter_NotFound(t *testing.T) {
	svcMock := &mockContentSafetyService{
		getContentFilterFunc: func(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error) {
			return nil, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	req := httptest.NewRequest("GET", "/content-filters/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestContentSafetyHandler_UpdateContentFilter(t *testing.T) {
	filterID := uuid.New()
	svcMock := &mockContentSafetyService{
		updateContentFilterFunc: func(ctx context.Context, id uuid.UUID, req *models.UpdateContentFilterRequest) (*models.ContentFilter, error) {
			return &models.ContentFilter{
				ID:        id,
				Name:      "Updated Name",
				Type:      models.FilterTypeNSFW,
				Enabled:   *req.Enabled,
				Threshold: *req.Threshold,
				Action:    models.FilterActionBlock,
			}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	enabled := true
	threshold := models.NSFWThresholdHigh
	reqBody := models.UpdateContentFilterRequest{
		Name:      func() *string { s := "Updated Name"; return &s }(),
		Enabled:   &enabled,
		Threshold: &threshold,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PATCH", "/content-filters/"+filterID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var filter models.ContentFilter
	err = json.NewDecoder(resp.Body).Decode(&filter)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", filter.Name)
	assert.True(t, filter.Enabled)
}

func TestContentSafetyHandler_DeleteContentFilter(t *testing.T) {
	filterID := uuid.New()
	svcMock := &mockContentSafetyService{
		deleteContentFilterFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	req := httptest.NewRequest("DELETE", "/content-filters/"+filterID.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestContentSafetyHandler_GetAgeVerification(t *testing.T) {
	serverID := uuid.New()
	svcMock := &mockContentSafetyService{
		getAgeVerifyFunc: func(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error) {
			return &models.AgeVerificationSetting{
				ID:               uuid.New(),
				ServerID:         serverID,
				Enabled:          true,
				RequiredAge:      18,
				VerificationType: "automatic",
			}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/age-verification", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var settings models.AgeVerificationSetting
	err = json.NewDecoder(resp.Body).Decode(&settings)
	assert.NoError(t, err)
	assert.True(t, settings.Enabled)
	assert.Equal(t, 18, settings.RequiredAge)
}

func TestContentSafetyHandler_GetAgeVerification_NotEnabled(t *testing.T) {
	serverID := uuid.New()
	svcMock := &mockContentSafetyService{
		getAgeVerifyFunc: func(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error) {
			return nil, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/age-verification", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, serverID.String(), result["server_id"])
	assert.Equal(t, false, result["enabled"])
}

func TestContentSafetyHandler_CreateAgeVerification(t *testing.T) {
	svcMock := &mockContentSafetyService{
		createAgeVerifyFunc: func(ctx context.Context, serverID, userID uuid.UUID, req *models.CreateAgeVerificationRequest) (*models.AgeVerificationSetting, error) {
			return &models.AgeVerificationSetting{
				ID:               uuid.New(),
				ServerID:         serverID,
				Enabled:          req.Enabled,
				RequiredAge:      req.RequiredAge,
				VerificationType: req.VerificationType,
			}, nil
		},
	}
	serverSvcMock := &mockContentSafetyServerService{
		getMemberFunc: func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
			return &models.Member{UserID: userID}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, serverSvcMock)

	serverID := uuid.New()
	userID := uuid.New()

	reqBody := models.CreateAgeVerificationRequest{
		Enabled:          true,
		RequiredAge:      18,
		VerificationType: "automatic",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/servers/"+serverID.String()+"/age-verification", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var settings models.AgeVerificationSetting
	err = json.NewDecoder(resp.Body).Decode(&settings)
	assert.NoError(t, err)
	assert.True(t, settings.Enabled)
	assert.Equal(t, 18, settings.RequiredAge)
}

func TestContentSafetyHandler_GetUserContentPreferences(t *testing.T) {
	userID := uuid.New()
	svcMock := &mockContentSafetyService{
		getUserPrefsFunc: func(ctx context.Context, uid uuid.UUID) (*models.UserContentPreference, error) {
			if uid == userID {
				return &models.UserContentPreference{
					ID:                 uuid.New(),
					UserID:             userID,
					NSFWFilterLevel:    models.NSFWThresholdMedium,
					HideNSFWContent:    true,
					HideExplicitContent: false,
					AutoCollapseNSFW:   false,
				}, nil
			}
			return nil, errors.New("user not found")
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	req := httptest.NewRequest("GET", "/users/@me/content-preferences", nil)
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var prefs models.UserContentPreference
	err = json.NewDecoder(resp.Body).Decode(&prefs)
	assert.NoError(t, err)
	assert.Equal(t, userID, prefs.UserID)
	assert.Equal(t, models.NSFWThresholdMedium, prefs.NSFWFilterLevel)
	assert.True(t, prefs.HideNSFWContent)
}

func TestContentSafetyHandler_UpdateUserContentPreferences(t *testing.T) {
	userID := uuid.New()
	svcMock := &mockContentSafetyService{
		updateUserPrefsFunc: func(ctx context.Context, uid uuid.UUID, req *models.UpdateUserContentPreferenceRequest) (*models.UserContentPreference, error) {
			return &models.UserContentPreference{
				ID:                 uuid.New(),
				UserID:             uid,
				NSFWFilterLevel:    *req.NSFWFilterLevel,
				HideNSFWContent:    *req.HideNSFWContent,
				HideExplicitContent: false,
				AutoCollapseNSFW:   false,
			}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	nsfwLevel := models.NSFWThresholdHigh
	hideNSFW := false
	reqBody := models.UpdateUserContentPreferenceRequest{
		NSFWFilterLevel: &nsfwLevel,
		HideNSFWContent: &hideNSFW,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PUT", "/users/@me/content-preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var prefs models.UserContentPreference
	err = json.NewDecoder(resp.Body).Decode(&prefs)
	assert.NoError(t, err)
	assert.Equal(t, models.NSFWThresholdHigh, prefs.NSFWFilterLevel)
	assert.False(t, prefs.HideNSFWContent)
}

func TestContentSafetyHandler_GetServerSafetySettings(t *testing.T) {
	serverID := uuid.New()
	svcMock := &mockContentSafetyService{
		getServerSafetyFunc: func(ctx context.Context, sid uuid.UUID) (*models.ContentSafetySettings, error) {
			return &models.ContentSafetySettings{
				ServerID: serverID,
				Filters: []*models.ContentFilter{
					{
						ID:        uuid.New(),
						ServerID:  serverID,
						Name:      "Test Filter",
						Type:      models.FilterTypeNSFW,
						Enabled:   true,
						Threshold: models.NSFWThresholdMedium,
						Action:    models.FilterActionBlock,
					},
				},
				AgeVerification: &models.AgeVerificationSetting{
					ID:               uuid.New(),
					ServerID:         serverID,
					Enabled:          true,
					RequiredAge:      18,
					VerificationType: "automatic",
				},
				ServerDefaultThreshold: models.NSFWThresholdMedium,
			}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/content-safety", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var settings models.ContentSafetySettings
	err = json.NewDecoder(resp.Body).Decode(&settings)
	assert.NoError(t, err)
	assert.Equal(t, serverID, settings.ServerID)
	assert.Len(t, settings.Filters, 1)
	assert.NotNil(t, settings.AgeVerification)
}

func TestContentSafetyHandler_TestContentScan(t *testing.T) {
	serverID := uuid.New()
	userID := uuid.New()
	svcMock := &mockContentSafetyService{
		scanContentFunc: func(ctx context.Context, serverID, channelID, userID uuid.UUID, userRoles []uuid.UUID, content string) (*models.ContentScanResult, error) {
			if content == "bad content" {
				return &models.ContentScanResult{
					Passed:     false,
					ActionTaken: models.FilterActionBlock,
					FilterName: "Bad Content Filter",
					Flags: []models.ContentFlag{
						{Type: models.FilterTypeCustomKeyword, Severity: 5},
					},
				}, nil
			}
			return &models.ContentScanResult{Passed: true}, nil
		},
	}
	app := setupContentSafetyTestApp(svcMock, nil)

	reqBody := map[string]string{"content": "bad content"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/servers/"+serverID.String()+"/content-safety/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", userID.String())

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.ContentScanResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, "Bad Content Filter", result.FilterName)
	assert.Len(t, result.Flags, 1)
}

func TestContentSafetyHandler_ListContentFilters_Unauthorized(t *testing.T) {
	svcMock := &mockContentSafetyService{}
	app := setupContentSafetyTestApp(svcMock, nil)

	serverID := uuid.New()
	req := httptest.NewRequest("GET", "/servers/"+serverID.String()+"/content-filters", nil)
	// No X-Test-User-ID header

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}
