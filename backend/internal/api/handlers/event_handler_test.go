package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
	"hearth/internal/services"
)

// Mock services for event handler tests

type mockEventService struct {
	listEventsFunc    func(ctx context.Context, serverID uuid.UUID, status *int) ([]*models.Event, error)
	createEventFunc   func(ctx context.Context, serverID, userID uuid.UUID, req *models.CreateEventRequest) (*models.Event, error)
	getEventFunc      func(ctx context.Context, eventID uuid.UUID) (*models.Event, error)
	updateEventFunc   func(ctx context.Context, eventID, userID uuid.UUID, req *models.UpdateEventRequest) (*models.Event, error)
	deleteEventFunc   func(ctx context.Context, eventID, userID uuid.UUID) error
	rsvpFunc          func(ctx context.Context, eventID, userID uuid.UUID, status string) error
	removeRSVPFunc    func(ctx context.Context, eventID, userID uuid.UUID) error
	getEventUsersFunc func(ctx context.Context, eventID uuid.UUID) ([]*models.EventRSVP, error)
	startEventFunc    func(ctx context.Context, eventID, userID uuid.UUID) error
}

func (m *mockEventService) ListEvents(ctx context.Context, serverID uuid.UUID, status *int) ([]*models.Event, error) {
	if m.listEventsFunc != nil {
		return m.listEventsFunc(ctx, serverID, status)
	}
	return nil, nil
}

func (m *mockEventService) CreateEvent(ctx context.Context, serverID, userID uuid.UUID, req *models.CreateEventRequest) (*models.Event, error) {
	if m.createEventFunc != nil {
		return m.createEventFunc(ctx, serverID, userID, req)
	}
	return nil, nil
}

func (m *mockEventService) GetEvent(ctx context.Context, eventID uuid.UUID) (*models.Event, error) {
	if m.getEventFunc != nil {
		return m.getEventFunc(ctx, eventID)
	}
	return nil, nil
}

func (m *mockEventService) UpdateEvent(ctx context.Context, eventID, userID uuid.UUID, req *models.UpdateEventRequest) (*models.Event, error) {
	if m.updateEventFunc != nil {
		return m.updateEventFunc(ctx, eventID, userID, req)
	}
	return nil, nil
}

func (m *mockEventService) DeleteEvent(ctx context.Context, eventID, userID uuid.UUID) error {
	if m.deleteEventFunc != nil {
		return m.deleteEventFunc(ctx, eventID, userID)
	}
	return nil
}

func (m *mockEventService) RSVP(ctx context.Context, eventID, userID uuid.UUID, status string) error {
	if m.rsvpFunc != nil {
		return m.rsvpFunc(ctx, eventID, userID, status)
	}
	return nil
}

func (m *mockEventService) RemoveRSVP(ctx context.Context, eventID, userID uuid.UUID) error {
	if m.removeRSVPFunc != nil {
		return m.removeRSVPFunc(ctx, eventID, userID)
	}
	return nil
}

func (m *mockEventService) GetEventUsers(ctx context.Context, eventID uuid.UUID) ([]*models.EventRSVP, error) {
	if m.getEventUsersFunc != nil {
		return m.getEventUsersFunc(ctx, eventID)
	}
	return nil, nil
}

func (m *mockEventService) StartEvent(ctx context.Context, eventID, userID uuid.UUID) error {
	if m.startEventFunc != nil {
		return m.startEventFunc(ctx, eventID, userID)
	}
	return nil
}

type mockEventServerService struct {
	getMemberFunc func(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
}

func (m *mockEventServerService) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	if m.getMemberFunc != nil {
		return m.getMemberFunc(ctx, serverID, userID)
	}
	return nil, nil
}

func setupEventTestApp(eventSvc *mockEventService, serverSvc *mockEventServerService) *fiber.App {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		userIDStr := c.Get("X-Test-User-ID")
		if userIDStr != "" {
			userID, err := uuid.Parse(userIDStr)
			if err == nil {
				c.Locals("userID", userID)
			} else {
				c.Locals("userID", uuid.Nil)
			}
		} else {
			c.Locals("userID", uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		}
		return c.Next()
	})

	// GET /servers/:id/events - List Server Events
	app.Get("/servers/:id/events", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}

		// Check membership
		_, err = serverSvc.GetMember(c.Context(), serverID, userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		}

		var status *int
		if s := c.Query("status"); s != "" {
			var v int
			if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
				status = &v
			}
		}

		events, err := eventSvc.ListEvents(c.Context(), serverID, status)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if events == nil {
			events = []*models.Event{}
		}

		return c.JSON(events)
	})

	// POST /servers/:id/events - Create Event
	app.Post("/servers/:id/events", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		serverID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID",
			})
		}

		// Check membership
		_, err = serverSvc.GetMember(c.Context(), serverID, userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not a server member",
			})
		}

		var req models.CreateEventRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		event, err := eventSvc.CreateEvent(c.Context(), serverID, userID, &req)
		if err != nil {
			switch err {
			case services.ErrEventNameRequired:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			case services.ErrEventNameTooLong:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			case services.ErrEventInvalidType:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			case services.ErrChannelRequired:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			case services.ErrInvalidChannelType:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			case services.ErrMissingPermission:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}

		return c.Status(fiber.StatusCreated).JSON(event)
	})

	// GET /events/:id - Get Event
	app.Get("/events/:id", func(c *fiber.Ctx) error {
		eventID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event ID",
			})
		}

		event, err := eventSvc.GetEvent(c.Context(), eventID)
		if err != nil {
			if err == services.ErrEventNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "event not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(event)
	})

	// PATCH /events/:id - Update Event
	app.Patch("/events/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		eventID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event ID",
			})
		}

		var req models.UpdateEventRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		event, err := eventSvc.UpdateEvent(c.Context(), eventID, userID, &req)
		if err != nil {
			switch err {
			case services.ErrEventNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			case services.ErrMissingPermission:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
			case services.ErrEventNameRequired:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			case services.ErrEventNameTooLong:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}

		return c.JSON(event)
	})

	// DELETE /events/:id - Delete Event
	app.Delete("/events/:id", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		eventID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event ID",
			})
		}

		err = eventSvc.DeleteEvent(c.Context(), eventID, userID)
		if err != nil {
			switch err {
			case services.ErrEventNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			case services.ErrMissingPermission:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	// POST /events/:id/rsvp - RSVP to Event
	app.Post("/events/:id/rsvp", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		eventID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event ID",
			})
		}

		var req struct {
			Status string `json:"status"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid request body",
			})
		}

		err = eventSvc.RSVP(c.Context(), eventID, userID, req.Status)
		if err != nil {
			if err == services.ErrEventNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "ok"})
	})

	// DELETE /events/:id/rsvp - Remove RSVP
	app.Delete("/events/:id/rsvp", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		eventID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event ID",
			})
		}

		err = eventSvc.RemoveRSVP(c.Context(), eventID, userID)
		if err != nil {
			if err == services.ErrEventNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "ok"})
	})

	// GET /events/:id/users - List Event Users
	app.Get("/events/:id/users", func(c *fiber.Ctx) error {
		eventID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event ID",
			})
		}

		users, err := eventSvc.GetEventUsers(c.Context(), eventID)
		if err != nil {
			if err == services.ErrEventNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if users == nil {
			users = []*models.EventRSVP{}
		}

		return c.JSON(users)
	})

	// POST /events/:id/start - Start Event
	app.Post("/events/:id/start", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(uuid.UUID)

		eventID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event ID",
			})
		}

		err = eventSvc.StartEvent(c.Context(), eventID, userID)
		if err != nil {
			switch err {
			case services.ErrEventNotFound:
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			case services.ErrMissingPermission:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
			case services.ErrEventNotActive:
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		}

		return c.JSON(fiber.Map{"status": "ok"})
	})

	return app
}

// Test ListServerEvents - Success
func TestListServerEvents_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	serverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	serverSvc := &mockEventServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{UserID: uID, ServerID: sID, JoinedAt: time.Now()}, nil
		},
	}

	eventSvc := &mockEventService{
		listEventsFunc: func(ctx context.Context, sID uuid.UUID, status *int) ([]*models.Event, error) {
			assert.Equal(t, serverID, sID)
			return []*models.Event{
				{ID: eventID, ServerID: serverID, Name: "Test Event", CreatorID: testUserID, Status: models.EventStatusScheduled},
			}, nil
		},
	}

	app := setupEventTestApp(eventSvc, serverSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", fmt.Sprintf("/servers/%s/events", serverID.String()), nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var events []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&events)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
}

// Test ListServerEvents - Invalid server ID
func TestListServerEvents_InvalidServerID(t *testing.T) {
	app := setupEventTestApp(&mockEventService{}, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/servers/not-a-uuid/events", nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// Test ListServerEvents - Not member
func TestListServerEvents_NotMember(t *testing.T) {
	serverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	serverSvc := &mockEventServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupEventTestApp(&mockEventService{}, serverSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", fmt.Sprintf("/servers/%s/events", serverID.String()), nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// Test CreateEvent - Success
func TestCreateEvent_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	serverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	serverSvc := &mockEventServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{UserID: uID, ServerID: sID, JoinedAt: time.Now()}, nil
		},
	}

	eventSvc := &mockEventService{
		createEventFunc: func(ctx context.Context, sID, uID uuid.UUID, req *models.CreateEventRequest) (*models.Event, error) {
			assert.Equal(t, serverID, sID)
			assert.Equal(t, testUserID, uID)
			assert.Equal(t, "Game Night", req.Name)
			return &models.Event{
				ID:        eventID,
				ServerID:  serverID,
				Name:      req.Name,
				CreatorID: testUserID,
				Status:    models.EventStatusScheduled,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupEventTestApp(eventSvc, serverSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"Game Night","description":"Weekly game night","entity_type":3,"scheduled_start":"2026-04-01T20:00:00Z"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/servers/%s/events", serverID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, eventID.String(), result["id"])
	assert.Equal(t, "Game Night", result["name"])
}

// Test CreateEvent - Invalid body
func TestCreateEvent_InvalidBody(t *testing.T) {
	serverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	serverSvc := &mockEventServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{UserID: uID, ServerID: sID, JoinedAt: time.Now()}, nil
		},
	}

	app := setupEventTestApp(&mockEventService{}, serverSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{invalid json`
	req := httptest.NewRequest("POST", fmt.Sprintf("/servers/%s/events", serverID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// Test CreateEvent - Name required error
func TestCreateEvent_NameRequired(t *testing.T) {
	serverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	serverSvc := &mockEventServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return &models.Member{UserID: uID, ServerID: sID, JoinedAt: time.Now()}, nil
		},
	}

	eventSvc := &mockEventService{
		createEventFunc: func(ctx context.Context, sID, uID uuid.UUID, req *models.CreateEventRequest) (*models.Event, error) {
			return nil, services.ErrEventNameRequired
		},
	}

	app := setupEventTestApp(eventSvc, serverSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"","entity_type":3}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/servers/%s/events", serverID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "event name is required")
}

// Test CreateEvent - Not member
func TestCreateEvent_NotMember(t *testing.T) {
	serverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	serverSvc := &mockEventServerService{
		getMemberFunc: func(ctx context.Context, sID, uID uuid.UUID) (*models.Member, error) {
			return nil, services.ErrNotServerMember
		},
	}

	app := setupEventTestApp(&mockEventService{}, serverSvc)
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"Test Event","entity_type":3}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/servers/%s/events", serverID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// Test GetEvent - Success
func TestGetEvent_Success(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	serverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	creatorID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	eventSvc := &mockEventService{
		getEventFunc: func(ctx context.Context, eID uuid.UUID) (*models.Event, error) {
			assert.Equal(t, eventID, eID)
			return &models.Event{
				ID:        eventID,
				ServerID:  serverID,
				Name:      "Test Event",
				CreatorID: creatorID,
				Status:    models.EventStatusScheduled,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", fmt.Sprintf("/events/%s", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", creatorID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "Test Event", result["name"])
}

// Test GetEvent - Not found
func TestGetEvent_NotFound(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		getEventFunc: func(ctx context.Context, eID uuid.UUID) (*models.Event, error) {
			return nil, services.ErrEventNotFound
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", fmt.Sprintf("/events/%s", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Test GetEvent - Invalid ID
func TestGetEvent_InvalidID(t *testing.T) {
	app := setupEventTestApp(&mockEventService{}, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", "/events/not-a-uuid", nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// Test UpdateEvent - Success
func TestUpdateEvent_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	newName := "Updated Event"

	eventSvc := &mockEventService{
		updateEventFunc: func(ctx context.Context, eID, uID uuid.UUID, req *models.UpdateEventRequest) (*models.Event, error) {
			assert.Equal(t, eventID, eID)
			assert.Equal(t, testUserID, uID)
			return &models.Event{
				ID:        eventID,
				Name:      *req.Name,
				CreatorID: testUserID,
				Status:    models.EventStatusScheduled,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := fmt.Sprintf(`{"name":"%s"}`, newName)
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/events/%s", eventID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, newName, result["name"])
}

// Test UpdateEvent - Not found
func TestUpdateEvent_NotFound(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		updateEventFunc: func(ctx context.Context, eID, uID uuid.UUID, req *models.UpdateEventRequest) (*models.Event, error) {
			return nil, services.ErrEventNotFound
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/events/%s", eventID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Test DeleteEvent - Success
func TestDeleteEvent_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		deleteEventFunc: func(ctx context.Context, eID, uID uuid.UUID) error {
			assert.Equal(t, eventID, eID)
			assert.Equal(t, testUserID, uID)
			return nil
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/events/%s", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

// Test DeleteEvent - Not found
func TestDeleteEvent_NotFound(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		deleteEventFunc: func(ctx context.Context, eID, uID uuid.UUID) error {
			return services.ErrEventNotFound
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/events/%s", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Test RSVP - Success
func TestRSVP_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		rsvpFunc: func(ctx context.Context, eID, uID uuid.UUID, status string) error {
			assert.Equal(t, eventID, eID)
			assert.Equal(t, testUserID, uID)
			assert.Equal(t, "interested", status)
			return nil
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"status":"interested"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/events/%s/rsvp", eventID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// Test RSVP - Event not found
func TestRSVP_EventNotFound(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		rsvpFunc: func(ctx context.Context, eID, uID uuid.UUID, status string) error {
			return services.ErrEventNotFound
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	body := `{"status":"interested"}`
	req := httptest.NewRequest("POST", fmt.Sprintf("/events/%s/rsvp", eventID.String()), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Test RemoveRSVP - Success
func TestRemoveRSVP_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		removeRSVPFunc: func(ctx context.Context, eID, uID uuid.UUID) error {
			assert.Equal(t, eventID, eID)
			assert.Equal(t, testUserID, uID)
			return nil
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/events/%s/rsvp", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// Test ListEventUsers - Success
func TestListEventUsers_Success(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	userID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	eventSvc := &mockEventService{
		getEventUsersFunc: func(ctx context.Context, eID uuid.UUID) ([]*models.EventRSVP, error) {
			assert.Equal(t, eventID, eID)
			return []*models.EventRSVP{
				{EventID: eventID, UserID: userID1, Status: models.RSVPStatusGoing, CreatedAt: time.Now()},
				{EventID: eventID, UserID: userID2, Status: models.RSVPStatusInterested, CreatedAt: time.Now()},
			}, nil
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("GET", fmt.Sprintf("/events/%s/users", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", userID1.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var users []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&users)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

// Test StartEvent - Success
func TestStartEvent_Success(t *testing.T) {
	testUserID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		startEventFunc: func(ctx context.Context, eID, uID uuid.UUID) error {
			assert.Equal(t, eventID, eID)
			assert.Equal(t, testUserID, uID)
			return nil
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", fmt.Sprintf("/events/%s/start", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", testUserID.String())

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// Test StartEvent - Not found
func TestStartEvent_NotFound(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		startEventFunc: func(ctx context.Context, eID, uID uuid.UUID) error {
			return services.ErrEventNotFound
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", fmt.Sprintf("/events/%s/start", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

// Test StartEvent - Not authorized (missing permission)
func TestStartEvent_NotAuthorized(t *testing.T) {
	eventID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	eventSvc := &mockEventService{
		startEventFunc: func(ctx context.Context, eID, uID uuid.UUID) error {
			return services.ErrMissingPermission
		},
	}

	app := setupEventTestApp(eventSvc, &mockEventServerService{})
	t.Cleanup(func() { _ = app.Shutdown() })

	req := httptest.NewRequest("POST", fmt.Sprintf("/events/%s/start", eventID.String()), nil)
	req.Header.Set("X-Test-User-ID", "11111111-1111-1111-1111-111111111111")

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
