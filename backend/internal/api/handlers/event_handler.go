package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

type EventHandler struct {
	eventService  *services.EventService
	serverService *services.ServerService
	permService   *services.PermissionService
}

// NewEventHandler creates a new event handler
func NewEventHandler(
	eventService *services.EventService,
	serverService *services.ServerService,
	permService *services.PermissionService,
) *EventHandler {
	return &EventHandler{
		eventService:  eventService,
		serverService: serverService,
		permService:   permService,
	}
}

// ListServerEvents lists events for a server
// @Summary List server events
// @Description Retrieves all scheduled events for a server
// @Tags Events
// @Produce json
// @Param id path string true "Server ID"
// @Param status query int false "Filter by status (1=scheduled, 2=active, 3=completed, 4=cancelled)"
// @Success 200 {array} models.Event "List of events"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/events [get]
func (h *EventHandler) ListServerEvents(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Verify user is a member
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "not a server member",
		})
	}

	// Optional status filter
	var status *int
	if s := c.Query("status"); s != "" {
		st := c.QueryInt("status", 0)
		if st > 0 {
			status = &st
		}
	}

	events, err := h.eventService.ListEvents(c.Context(), serverID, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list events",
		})
	}

	return c.JSON(events)
}

// CreateEvent creates a new scheduled event
// @Summary Create event
// @Description Creates a new scheduled event in a server
// @Tags Events
// @Accept json
// @Produce json
// @Param id path string true "Server ID"
// @Param body body models.CreateEventRequest true "Event data"
// @Success 201 {object} models.Event "Event created"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 403 {object} fiber.Map "Missing MANAGE_EVENTS permission"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/events [post]
func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Verify user is a member
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
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

	event, err := h.eventService.CreateEvent(c.Context(), serverID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrMissingPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "missing MANAGE_EVENTS permission",
			})
		case services.ErrEventNameRequired:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "event name is required",
			})
		case services.ErrEventNameTooLong:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "event name too long (max 100 chars)",
			})
		case services.ErrEventInvalidType:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid event type",
			})
		case services.ErrChannelRequired:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel required for stage/voice events",
			})
		case services.ErrInvalidChannelType:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "channel must be voice or stage type",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to create event",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(event)
}

// GetEvent retrieves an event by ID
// @Summary Get event
// @Description Retrieves a scheduled event by ID
// @Tags Events
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.Event "Event found"
// @Failure 400 {object} fiber.Map "Invalid event ID"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id} [get]
func (h *EventHandler) GetEvent(c *fiber.Ctx) error {
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	event, err := h.eventService.GetEvent(c.Context(), eventID)
	if err != nil {
		if err == services.ErrEventNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get event",
		})
	}

	return c.JSON(event)
}

// UpdateEvent updates an event
// @Summary Update event
// @Description Updates a scheduled event (creator or MANAGE_EVENTS only)
// @Tags Events
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param body body models.UpdateEventRequest true "Event update data"
// @Success 200 {object} models.Event "Event updated"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id} [patch]
func (h *EventHandler) UpdateEvent(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	var req models.UpdateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	event, err := h.eventService.UpdateEvent(c.Context(), eventID, userID, &req)
	if err != nil {
		switch err {
		case services.ErrEventNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		case services.ErrMissingPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to update this event",
			})
		case services.ErrEventNameRequired:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "event name is required",
			})
		case services.ErrEventNameTooLong:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "event name too long (max 100 chars)",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update event",
			})
		}
	}

	return c.JSON(event)
}

// DeleteEvent deletes an event
// @Summary Delete event
// @Description Deletes a scheduled event (creator or MANAGE_EVENTS only)
// @Tags Events
// @Param id path string true "Event ID"
// @Success 204 "Event deleted"
// @Failure 400 {object} fiber.Map "Invalid event ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id} [delete]
func (h *EventHandler) DeleteEvent(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	if err := h.eventService.DeleteEvent(c.Context(), eventID, userID); err != nil {
		switch err {
		case services.ErrEventNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		case services.ErrMissingPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to delete this event",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete event",
			})
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RSVP adds or updates a user's RSVP to an event
// @Summary RSVP to event
// @Description Adds or updates a user's RSVP to an event
// @Tags Events
// @Accept json
// @Produce json
// @Param id path string true "Event ID"
// @Param body body models.RSVPRequest true "RSVP data"
// @Success 200 {object} fiber.Map "RSVP updated"
// @Failure 400 {object} fiber.Map "Invalid request"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id}/rsvp [post]
func (h *EventHandler) RSVP(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	var req models.RSVPRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.eventService.RSVP(c.Context(), eventID, userID, req.Status); err != nil {
		switch err {
		case services.ErrEventNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to RSVP",
			})
		}
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// RemoveRSVP removes a user's RSVP from an event
// @Summary Remove RSVP
// @Description Removes a user's RSVP from an event
// @Tags Events
// @Param id path string true "Event ID"
// @Success 200 {object} fiber.Map "RSVP removed"
// @Failure 400 {object} fiber.Map "Invalid event ID"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id}/rsvp [delete]
func (h *EventHandler) RemoveRSVP(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	if err := h.eventService.RemoveRSVP(c.Context(), eventID, userID); err != nil {
		switch err {
		case services.ErrEventNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to remove RSVP",
			})
		}
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// ListEventUsers lists users who RSVPed to an event
// @Summary List event users
// @Description Retrieves all users who have RSVPed to an event
// @Tags Events
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {array} models.EventRSVP "List of RSVPs"
// @Failure 400 {object} fiber.Map "Invalid event ID"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id}/users [get]
func (h *EventHandler) ListEventUsers(c *fiber.Ctx) error {
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	users, err := h.eventService.GetEventUsers(c.Context(), eventID)
	if err != nil {
		switch err {
		case services.ErrEventNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list event users",
			})
		}
	}

	return c.JSON(users)
}

// StartEvent transitions an event to active status
// @Summary Start event
// @Description Starts a scheduled event (creator or MANAGE_EVENTS only)
// @Tags Events
// @Param id path string true "Event ID"
// @Success 200 {object} fiber.Map "Event started"
// @Failure 400 {object} fiber.Map "Invalid event ID"
// @Failure 403 {object} fiber.Map "Not authorized"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 400 {object} fiber.Map "Event cannot be started"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id}/start [post]
func (h *EventHandler) StartEvent(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	if err := h.eventService.StartEvent(c.Context(), eventID, userID); err != nil {
		switch err {
		case services.ErrEventNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		case services.ErrMissingPermission:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "not authorized to start this event",
			})
		case services.ErrEventNotActive:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "event cannot be started (not in scheduled status)",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to start event",
			})
		}
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// ExportEventICal exports a single event as iCal format
// @Summary Export event as iCal
// @Description Exports a single event as an iCalendar file
// @Tags Events
// @Produce text/calendar
// @Param id path string true "Event ID"
// @Success 200 {file} binary "iCalendar file"
// @Failure 400 {object} fiber.Map "Invalid event ID"
// @Failure 404 {object} fiber.Map "Event not found"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /events/{id}/ical [get]
func (h *EventHandler) ExportEventICal(c *fiber.Ctx) error {
	eventID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid event id",
		})
	}

	event, err := h.eventService.GetEvent(c.Context(), eventID)
	if err != nil {
		if err == services.ErrEventNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "event not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get event",
		})
	}

	ical := generateICal([]*models.Event{event})
	filename := fmt.Sprintf("event-%s.ics", event.ID.String())

	c.Set("Content-Type", "text/calendar; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.SendString(ical)
}

// ExportServerEventsICal exports all server events as iCal
// @Summary Export server events as iCal
// @Description Exports all scheduled events for a server as an iCalendar file
// @Tags Events
// @Produce text/calendar
// @Param id path string true "Server ID"
// @Success 200 {file} binary "iCalendar file"
// @Failure 400 {object} fiber.Map "Invalid server ID"
// @Failure 403 {object} fiber.Map "Not a server member"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /servers/{id}/events/ical [get]
func (h *EventHandler) ExportServerEventsICal(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid server id",
		})
	}

	// Verify user is a member
	member, err := h.serverService.GetMember(c.Context(), serverID, userID)
	if err != nil || member == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "not a server member",
		})
	}

	events, err := h.eventService.ListEvents(c.Context(), serverID, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list events",
		})
	}

	ical := generateICal(events)
	filename := fmt.Sprintf("server-%s-events.ics", serverID.String())

	c.Set("Content-Type", "text/calendar; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.SendString(ical)
}

// ExportUserEventsICal exports all events a user has RSVPed to as iCal
// @Summary Export user events as iCal
// @Description Exports all events a user has RSVPed to as an iCalendar file
// @Tags Events
// @Produce text/calendar
// @Success 200 {file} binary "iCalendar file"
// @Failure 500 {object} fiber.Map "Internal server error"
// @Router /users/me/events/ical [get]
func (h *EventHandler) ExportUserEventsICal(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		return err
	}

	events, err := h.eventService.GetUserRSVPEvents(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list user events",
		})
	}

	ical := generateICal(events)
	filename := fmt.Sprintf("my-events.ics")

	c.Set("Content-Type", "text/calendar; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.SendString(ical)
}

// generateICal generates an iCalendar string from a list of events
func generateICal(events []*models.Event) string {
	var ical strings.Builder
	ical.WriteString("BEGIN:VCALENDAR\r\n")
	ical.WriteString("VERSION:2.0\r\n")
	ical.WriteString("PRODID:-//Hearth//Hearth Events//EN\r\n")
	ical.WriteString("CALSCALE:GREGORIAN\r\n")
	ical.WriteString("METHOD:PUBLISH\r\n")

	for _, event := range events {
		ical.WriteString("BEGIN:VEVENT\r\n")
		fmt.Fprintf(&ical, "UID:%s@hearth\r\n", event.ID.String())
		fmt.Fprintf(&ical, "DTSTAMP:%s\r\n", formatICalTime(time.Now()))
		fmt.Fprintf(&ical, "DTSTART:%s\r\n", formatICalTime(event.ScheduledStart))
		if event.ScheduledEnd != nil {
			fmt.Fprintf(&ical, "DTEND:%s\r\n", formatICalTime(*event.ScheduledEnd))
		}
		fmt.Fprintf(&ical, "SUMMARY:%s\r\n", escapeICalText(event.Name))
		if event.Description != "" {
			fmt.Fprintf(&ical, "DESCRIPTION:%s\r\n", escapeICalText(event.Description))
		}
		if event.Location != "" {
			fmt.Fprintf(&ical, "LOCATION:%s\r\n", escapeICalText(event.Location))
		}
		ical.WriteString("ORGANIZER;CN=Event Creator:mailto:noreply@hearth.app\r\n")
		ical.WriteString("STATUS:")
		switch event.Status {
		case models.EventStatusScheduled:
			ical.WriteString("CONFIRMED\r\n")
		case models.EventStatusCancelled:
			ical.WriteString("CANCELLED\r\n")
		case models.EventStatusCompleted:
			ical.WriteString("COMPLETED\r\n")
		default:
			ical.WriteString("TENTATIVE\r\n")
		}
		ical.WriteString("END:VEVENT\r\n")
	}

	ical.WriteString("END:VCALENDAR\r\n")
	return ical.String()
}

// formatICalTime formats a time as iCalendar datetime string
func formatICalTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// escapeICalText escapes special characters for iCalendar text fields
func escapeICalText(s string) string {
	s = fmt.Sprintf("%v", s)
	s = replaceAll(s, "\\", "\\\\")
	s = replaceAll(s, ",", "\\,")
	s = replaceAll(s, ";", "\\;")
	s = replaceAll(s, "\n", "\\n")
	return s
}

// replaceAll is a simple helper to replace all occurrences
func replaceAll(s, old, new string) string {
	result := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			result += s
			break
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
	return result
}

// indexOf returns the index of the first occurrence of substr in s
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
