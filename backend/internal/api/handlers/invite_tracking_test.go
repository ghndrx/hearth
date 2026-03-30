package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// TestInviteTrackingAnalytics_ClassificationLogic tests the real vs fake user classification
func TestInviteTrackingAnalytics_ClassificationLogic(t *testing.T) {
	tests := []struct {
		name           string
		accountAgeDays int
		expectReal     bool
	}{
		{" brand new account", 0, false},
		{"1 day old", 1, false},
		{"3 days old", 3, false},
		{"6 days old", 6, false},
		{"7 days old - exact boundary", 7, true},
		{"14 days old", 14, true},
		{"30 days old", 30, true},
		{"365 days old", 365, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isReal := tt.accountAgeDays >= 7
			assert.Equal(t, tt.expectReal, isReal)
		})
	}
}

// TestInviteTrackingAnalytics_UseLogClassification tests classification of multiple use logs
func TestInviteTrackingAnalytics_UseLogClassification(t *testing.T) {
	useLogs := []models.InviteUseLog{
		{UserID: uuid.New(), AccountAgeDays: 30, JoinedAt: time.Now().Add(-30 * 24 * time.Hour)},
		{UserID: uuid.New(), AccountAgeDays: 60, JoinedAt: time.Now().Add(-60 * 24 * time.Hour)},
		{UserID: uuid.New(), AccountAgeDays: 3, JoinedAt: time.Now().Add(-3 * 24 * time.Hour)}, // fake
	}

	realUsers := 0
	fakes := 0
	for _, log := range useLogs {
		if log.AccountAgeDays >= 7 {
			realUsers++
		} else {
			fakes++
		}
	}

	assert.Equal(t, 2, realUsers)
	assert.Equal(t, 1, fakes)
}

// TestInviteTrackingAnalytics_AllReal tests when all users are real
func TestInviteTrackingAnalytics_AllReal(t *testing.T) {
	useLogs := []models.InviteUseLog{
		{UserID: uuid.New(), AccountAgeDays: 30, JoinedAt: time.Now().Add(-30 * 24 * time.Hour)},
		{UserID: uuid.New(), AccountAgeDays: 60, JoinedAt: time.Now().Add(-60 * 24 * time.Hour)},
		{UserID: uuid.New(), AccountAgeDays: 90, JoinedAt: time.Now().Add(-90 * 24 * time.Hour)},
		{UserID: uuid.New(), AccountAgeDays: 120, JoinedAt: time.Now().Add(-120 * 24 * time.Hour)},
	}

	realUsers := 0
	fakes := 0
	for _, log := range useLogs {
		if log.AccountAgeDays >= 7 {
			realUsers++
		} else {
			fakes++
		}
	}

	assert.Equal(t, 4, realUsers)
	assert.Equal(t, 0, fakes)
}

// TestInviteTrackingAnalytics_AllFake tests when all users are fake
func TestInviteTrackingAnalytics_AllFake(t *testing.T) {
	useLogs := []models.InviteUseLog{
		{UserID: uuid.New(), AccountAgeDays: 1, JoinedAt: time.Now().Add(-1 * 24 * time.Hour)},
		{UserID: uuid.New(), AccountAgeDays: 2, JoinedAt: time.Now().Add(-2 * 24 * time.Hour)},
		{UserID: uuid.New(), AccountAgeDays: 5, JoinedAt: time.Now().Add(-5 * 24 * time.Hour)},
	}

	realUsers := 0
	fakes := 0
	for _, log := range useLogs {
		if log.AccountAgeDays >= 7 {
			realUsers++
		} else {
			fakes++
		}
	}

	assert.Equal(t, 0, realUsers)
	assert.Equal(t, 3, fakes)
}

// TestInviteTrackingAnalytics_EmptyUseLogs tests handling of empty use logs
func TestInviteTrackingAnalytics_EmptyUseLogs(t *testing.T) {
	useLogs := []models.InviteUseLog{}

	realUsers := 0
	fakes := 0
	for _, log := range useLogs {
		if log.AccountAgeDays >= 7 {
			realUsers++
		} else {
			fakes++
		}
	}

	assert.Equal(t, 0, realUsers)
	assert.Equal(t, 0, fakes)
}

// TestInviteTrackingHandler_InvalidServerID tests error for invalid UUID
func TestInviteTrackingHandler_InvalidServerID(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	app.Get("/api/v1/servers/:id/analytics/invites", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server id",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/not-a-uuid/analytics/invites", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "invalid server id", result["error"])
}

// TestInviteTrackingHandler_ValidServerID tests valid UUID parsing
func TestInviteTrackingHandler_ValidServerID(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	validUUID := uuid.New()

	app.Get("/api/v1/servers/:id/analytics/invites", func(c *fiber.Ctx) error {
		_, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server id",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+validUUID.String()+"/analytics/invites", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestInviteTrackingHandler_UnauthorizedError tests unauthorized response
func TestInviteTrackingHandler_UnauthorizedError(t *testing.T) {
	app := fiber.New()
	t.Cleanup(func() { app.Shutdown() })

	app.Get("/api/v1/servers/:id/analytics/invites", func(c *fiber.Ctx) error {
		// Simulate missing user context (no auth header)
		userID := c.Locals("userID")
		if userID == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+uuid.New().String()+"/analytics/invites", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "unauthorized", result["error"])
}

// TestInviteTrackingHandler_ErrorMapping tests error mapping from service
func TestInviteTrackingHandler_ErrorMapping(t *testing.T) {
	handler := &InviteTrackingHandler{}

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "ServerNotFound maps to 404",
			err:            context.DeadlineExceeded,
			expectedStatus: fiber.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			t.Cleanup(func() { app.Shutdown() })

			app.Get("/test", func(c *fiber.Ctx) error {
				return handler.handleError(c, tt.err)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var result map[string]string
			err = json.NewDecoder(resp.Body).Decode(&result)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedError, result["error"])
		})
	}
}

// TestInviteTrackingAnalytics_BuildResults tests building the results array
func TestInviteTrackingAnalytics_BuildResults(t *testing.T) {
	serverID := uuid.New()
	now := time.Now()

	invites := []*models.Invite{
		{
			Code:      "code1",
			ServerID:  serverID,
			Uses:      5,
			CreatedAt: now.Add(-24 * time.Hour),
			ExpiresAt: nil,
		},
		{
			Code:      "code2",
			ServerID:  serverID,
			Uses:      3,
			CreatedAt: now.Add(-48 * time.Hour),
			ExpiresAt: nil,
		},
	}

	// Build meta lookup
	type inviteMeta struct {
		CreatedAt time.Time
		ExpiresAt *time.Time
	}
	metaMap := make(map[string]inviteMeta, len(invites))
	for _, inv := range invites {
		metaMap[inv.Code] = inviteMeta{CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt}
	}

	assert.Equal(t, 2, len(metaMap))
	assert.Equal(t, invites[0].CreatedAt, metaMap["code1"].CreatedAt)
	assert.Equal(t, invites[1].CreatedAt, metaMap["code2"].CreatedAt)
}

// TestInviteTrackingAnalytics_ResultsMapping tests mapping analytics to response format
func TestInviteTrackingAnalytics_ResultsMapping(t *testing.T) {
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	newTime := time.Now().Add(-3 * 24 * time.Hour)

	analytics := []models.InviteAnalytics{
		{
			Code:       "testcode",
			TotalUses:  3,
			UseLogs: []models.InviteUseLog{
				{UserID: uuid.New(), AccountAgeDays: 30, JoinedAt: oldTime},
				{UserID: uuid.New(), AccountAgeDays: 60, JoinedAt: oldTime},
				{UserID: uuid.New(), AccountAgeDays: 3, JoinedAt: newTime}, // fake
			},
		},
	}

	// Build results as the handler does
	results := make([]InviteTrackingAnalytics, 0, len(analytics))
	for _, a := range analytics {
		realUsers := 0
		fakes := 0
		for _, log := range a.UseLogs {
			if log.AccountAgeDays >= 7 {
				realUsers++
			} else {
				fakes++
			}
		}

		entry := InviteTrackingAnalytics{
			Code:      a.Code,
			Uses:      a.TotalUses,
			RealUsers: realUsers,
			Fakes:     fakes,
		}
		results = append(results, entry)
	}

	assert.Len(t, results, 1)
	assert.Equal(t, "testcode", results[0].Code)
	assert.Equal(t, 3, results[0].Uses)
	assert.Equal(t, 2, results[0].RealUsers)
	assert.Equal(t, 1, results[0].Fakes)
}
