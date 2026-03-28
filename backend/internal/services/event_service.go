package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrEventNotFound      = errors.New("event not found")
	ErrEventInvalid       = errors.New("invalid event")
	ErrEventNameRequired  = errors.New("event name is required")
	ErrEventNameTooLong   = errors.New("event name too long (max 100 chars)")
	ErrEventInvalidType   = errors.New("invalid event type")
	ErrEventInvalidStatus = errors.New("invalid event status")
	ErrNotEventCreator    = errors.New("not the event creator")
	ErrEventCannotStart   = errors.New("event cannot be started")
	ErrEventNotActive     = errors.New("event is not active")
	ErrChannelRequired    = errors.New("channel required for stage/voice events")
	ErrInvalidChannelType = errors.New("channel must be voice or stage type")
)

// EventRepository defines event data access
type EventRepository interface {
	CreateEvent(ctx context.Context, event *models.Event) error
	GetEventByID(ctx context.Context, id uuid.UUID) (*models.Event, error)
	UpdateEvent(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*models.Event, error)
	DeleteEvent(ctx context.Context, id uuid.UUID) error
	ListServerEvents(ctx context.Context, serverID uuid.UUID, statusFilter *int) ([]*models.Event, error)
	ListUserRSVPEvents(ctx context.Context, userID uuid.UUID) ([]*models.Event, error)
	RSVP(ctx context.Context, eventID, userID uuid.UUID, status models.RSVPStatus) error
	RemoveRSVP(ctx context.Context, eventID, userID uuid.UUID) error
	GetEventUsers(ctx context.Context, eventID uuid.UUID) ([]*models.EventRSVP, error)
	GetUserRSVP(ctx context.Context, eventID, userID uuid.UUID) (*models.EventRSVP, error)
	IncrementUserCount(ctx context.Context, eventID uuid.UUID, delta int) error
}

// EventService handles event-related business logic
type EventService struct {
	eventRepo   EventRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	permService *PermissionService
	eventBus    EventBus
}

// NewEventService creates a new event service
func NewEventService(
	eventRepo EventRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	eventBus EventBus,
) *EventService {
	return &EventService{
		eventRepo:   eventRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		permService: permService,
		eventBus:    eventBus,
	}
}

// CreateEvent creates a new scheduled event
func (s *EventService) CreateEvent(ctx context.Context, serverID, creatorID uuid.UUID, req *models.CreateEventRequest) (*models.Event, error) {
	// Validate name
	if req.Name == "" {
		return nil, ErrEventNameRequired
	}
	if len(req.Name) > 100 {
		return nil, ErrEventNameTooLong
	}

	// Validate event type
	if req.EntityType < models.EventTypeStage || req.EntityType > models.EventTypeExternal {
		return nil, ErrEventInvalidType
	}

	// Validate channel for stage/voice events
	if req.EntityType != models.EventTypeExternal {
		if req.ChannelID == nil {
			return nil, ErrChannelRequired
		}
		// Validate channel exists and is voice or stage type
		if s.channelRepo != nil {
			channel, err := s.channelRepo.GetByID(ctx, *req.ChannelID)
			if err != nil || channel == nil {
				return nil, ErrChannelRequired
			}
			if channel.Type != models.ChannelTypeVoice && channel.Type != models.ChannelTypeStage {
				return nil, ErrInvalidChannelType
			}
		}
	}

	// Check MANAGE_EVENTS permission if permService is available
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, creatorID, models.PermManageEvents); err != nil {
			return nil, err
		}
	}

	event := &models.Event{
		ID:             uuid.New(),
		ServerID:       serverID,
		ChannelID:      req.ChannelID,
		CreatorID:      creatorID,
		Name:           req.Name,
		Description:    req.Description,
		ImageURL:       req.ImageURL,
		ScheduledStart: req.ScheduledStart,
		ScheduledEnd:   req.ScheduledEnd,
		EntityType:     req.EntityType,
		Location:       req.Location,
		Status:         models.EventStatusScheduled,
		UserCount:      0,
		RecurrenceRule: req.RecurrenceRule,
		CreatedAt:      time.Now(),
	}

	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		return nil, err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("event.created", &EventCreatedEvent{
			Event:    event,
			ServerID: serverID,
		})
	}

	return event, nil
}

// GetEvent retrieves an event by ID
func (s *EventService) GetEvent(ctx context.Context, eventID uuid.UUID) (*models.Event, error) {
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}
	return event, nil
}

// UpdateEvent updates an event
func (s *EventService) UpdateEvent(ctx context.Context, eventID, userID uuid.UUID, req *models.UpdateEventRequest) (*models.Event, error) {
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	// Check permission: only creator or MANAGE_EVENTS
	isCreator := event.CreatorID == userID
	hasPermission := false
	if s.permService != nil {
		if has, err := s.permService.HasPermission(ctx, event.ServerID, userID, models.PermManageEvents); err == nil {
			hasPermission = has
		}
	}

	if !isCreator && !hasPermission {
		return nil, ErrMissingPermission
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ErrEventNameRequired
		}
		if len(*req.Name) > 100 {
			return nil, ErrEventNameTooLong
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	if req.ScheduledStart != nil {
		updates["scheduled_start"] = *req.ScheduledStart
	}
	if req.ScheduledEnd != nil {
		updates["scheduled_end"] = *req.ScheduledEnd
	}
	if req.EntityType != nil {
		if *req.EntityType < models.EventTypeStage || *req.EntityType > models.EventTypeExternal {
			return nil, ErrEventInvalidType
		}
		updates["entity_type"] = *req.EntityType
	}
	if req.ChannelID != nil {
		updates["channel_id"] = *req.ChannelID
	}
	if req.Location != nil {
		updates["location"] = *req.Location
	}
	if req.Status != nil {
		if *req.Status < models.EventStatusScheduled || *req.Status > models.EventStatusCancelled {
			return nil, ErrEventInvalidStatus
		}
		updates["status"] = *req.Status
	}
	if req.RecurrenceRule != nil {
		updates["recurrence_rule"] = req.RecurrenceRule
	}

	updated, err := s.eventRepo.UpdateEvent(ctx, eventID, updates)
	if err != nil {
		return nil, err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("event.updated", &EventUpdatedEvent{
			Event: updated,
		})
	}

	return updated, nil
}

// DeleteEvent deletes an event
func (s *EventService) DeleteEvent(ctx context.Context, eventID, userID uuid.UUID) error {
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Check permission: only creator or MANAGE_EVENTS
	isCreator := event.CreatorID == userID
	hasPermission := false
	if s.permService != nil {
		if has, err := s.permService.HasPermission(ctx, event.ServerID, userID, models.PermManageEvents); err == nil {
			hasPermission = has
		}
	}

	if !isCreator && !hasPermission {
		return ErrMissingPermission
	}

	if err := s.eventRepo.DeleteEvent(ctx, eventID); err != nil {
		return err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("event.deleted", &EventDeletedEvent{
			EventID:  eventID,
			ServerID: event.ServerID,
		})
	}

	return nil
}

// ListEvents lists events for a server
func (s *EventService) ListEvents(ctx context.Context, serverID uuid.UUID, status *int) ([]*models.Event, error) {
	return s.eventRepo.ListServerEvents(ctx, serverID, status)
}

// RSVP adds or updates a user's RSVP to an event
func (s *EventService) RSVP(ctx context.Context, eventID, userID uuid.UUID, status models.RSVPStatus) error {
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Check if user already has an RSVP
	existingRSVP, err := s.eventRepo.GetUserRSVP(ctx, eventID, userID)
	if err != nil {
		return err
	}

	if err := s.eventRepo.RSVP(ctx, eventID, userID, status); err != nil {
		return err
	}

	// Update user count
	if existingRSVP == nil {
		// New RSVP
		if err := s.eventRepo.IncrementUserCount(ctx, eventID, 1); err != nil {
			return err
		}
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("event.rsvp", &EventRSVPEvent{
			EventID: eventID,
			UserID:  userID,
			Status:  status,
			Added:   existingRSVP == nil,
		})
	}

	return nil
}

// RemoveRSVP removes a user's RSVP from an event
func (s *EventService) RemoveRSVP(ctx context.Context, eventID, userID uuid.UUID) error {
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Check if user has an RSVP
	existingRSVP, err := s.eventRepo.GetUserRSVP(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if existingRSVP == nil {
		// No RSVP to remove
		return nil
	}

	if err := s.eventRepo.RemoveRSVP(ctx, eventID, userID); err != nil {
		return err
	}

	// Decrement user count
	if err := s.eventRepo.IncrementUserCount(ctx, eventID, -1); err != nil {
		return err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("event.rsvp.removed", &EventRSVPRemovedEvent{
			EventID: eventID,
			UserID:  userID,
		})
	}

	return nil
}

// GetEventUsers gets all users who RSVPed to an event
func (s *EventService) GetEventUsers(ctx context.Context, eventID uuid.UUID) ([]*models.EventRSVP, error) {
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	return s.eventRepo.GetEventUsers(ctx, eventID)
}

// StartEvent transitions an event to active status
func (s *EventService) StartEvent(ctx context.Context, eventID, userID uuid.UUID) error {
	event, err := s.eventRepo.GetEventByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event == nil {
		return ErrEventNotFound
	}

	// Check permission: only creator or MANAGE_EVENTS
	isCreator := event.CreatorID == userID
	hasPermission := false
	if s.permService != nil {
		if has, err := s.permService.HasPermission(ctx, event.ServerID, userID, models.PermManageEvents); err == nil {
			hasPermission = has
		}
	}

	if !isCreator && !hasPermission {
		return ErrMissingPermission
	}

	// Can only start scheduled events
	if event.Status != models.EventStatusScheduled {
		return ErrEventNotActive
	}

	updates := map[string]interface{}{
		"status": models.EventStatusActive,
	}

	updated, err := s.eventRepo.UpdateEvent(ctx, eventID, updates)
	if err != nil {
		return err
	}

	// Publish event
	if s.eventBus != nil {
		s.eventBus.Publish("event.started", &EventStartedEvent{
			Event: updated,
		})
	}

	return nil
}

// GetUserRSVPEvents gets events a user has RSVPed to
func (s *EventService) GetUserRSVPEvents(ctx context.Context, userID uuid.UUID) ([]*models.Event, error) {
	return s.eventRepo.ListUserRSVPEvents(ctx, userID)
}

// GetUserRSVP gets a user's RSVP for a specific event
func (s *EventService) GetUserRSVP(ctx context.Context, eventID, userID uuid.UUID) (*models.EventRSVP, error) {
	return s.eventRepo.GetUserRSVP(ctx, eventID, userID)
}

// Events

type EventCreatedEvent struct {
	Event    *models.Event
	ServerID uuid.UUID
}

type EventUpdatedEvent struct {
	Event *models.Event
}

type EventDeletedEvent struct {
	EventID  uuid.UUID
	ServerID uuid.UUID
}

type EventRSVPEvent struct {
	EventID uuid.UUID
	UserID  uuid.UUID
	Status  models.RSVPStatus
	Added   bool
}

type EventRSVPRemovedEvent struct {
	EventID uuid.UUID
	UserID  uuid.UUID
}

type EventStartedEvent struct {
	Event *models.Event
}
