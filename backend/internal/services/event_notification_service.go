package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// EventNotificationService handles notifications for scheduled events
type EventNotificationService struct {
	eventRepo    EventRepository
	userRepo     UserRepository
	serverRepo   ServerRepository
	notifService *NotificationService
	eventBus     EventBus
}

// NewEventNotificationService creates a new event notification service
func NewEventNotificationService(
	eventRepo EventRepository,
	userRepo UserRepository,
	serverRepo ServerRepository,
	notifService *NotificationService,
	eventBus EventBus,
) *EventNotificationService {
	svc := &EventNotificationService{
		eventRepo:    eventRepo,
		userRepo:     userRepo,
		serverRepo:   serverRepo,
		notifService: notifService,
		eventBus:     eventBus,
	}

	// Subscribe to event events
	if eventBus != nil {
		eventBus.Subscribe("event.created", svc.handleEventCreated)
		eventBus.Subscribe("event.rsvp", svc.handleEventRSVP)
		eventBus.Subscribe("event.started", svc.handleEventStarted)
	}

	return svc
}

// handleEventCreated sends notification to server members about new event
func (s *EventNotificationService) handleEventCreated(data interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	event, ok := data.(*EventCreatedEvent)
	if !ok || event == nil || event.Event == nil {
		log.Printf("event_notification: invalid event.created data")
		return
	}

	// Get server to find server name
	server, err := s.serverRepo.GetByID(ctx, event.ServerID)
	if err != nil || server == nil {
		log.Printf("event_notification: failed to get server %s: %v", event.ServerID, err)
		return
	}

	// Create notification data
	notifData, _ := json.Marshal(map[string]interface{}{
		"event_id":  event.Event.ID.String(),
		"server_id": event.ServerID.String(),
	})

	// Notify event creator (RSVP confirmation)
	title := "Event Created"
	body := fmt.Sprintf("Your event '%s' has been scheduled for %s", event.Event.Name, event.Event.ScheduledStart.Format("Jan 2 at 3:04 PM"))

	createReq := &models.CreateNotificationRequest{
		UserID:   event.Event.CreatorID,
		Type:     models.NotificationTypeEventInvite,
		Title:    title,
		Body:     body,
		Data:     stringPtr(string(notifData)),
		ServerID: &event.ServerID,
	}

	if _, err := s.notifService.CreateNotification(ctx, createReq); err != nil {
		log.Printf("event_notification: failed to notify creator: %v", err)
	}

	log.Printf("event_notification: notified creator %s about event %s", event.Event.CreatorID, event.Event.ID)
}

// handleEventRSVP sends confirmation to user when they RSVP
func (s *EventNotificationService) handleEventRSVP(data interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rsvpEvent, ok := data.(*EventRSVPEvent)
	if !ok {
		log.Printf("event_notification: invalid event.rsvp data")
		return
	}

	// Get event details
	event, err := s.eventRepo.GetEventByID(ctx, rsvpEvent.EventID)
	if err != nil || event == nil {
		log.Printf("event_notification: failed to get event %s: %v", rsvpEvent.EventID, err)
		return
	}

	// Create notification data
	notifData, _ := json.Marshal(map[string]interface{}{
		"event_id": rsvpEvent.EventID.String(),
		"status":   int(rsvpEvent.Status),
	})

	statusText := "Interested"
	if rsvpEvent.Status == models.RSVPStatusGoing {
		statusText = "Going"
	}

	title := "RSVP Confirmed"
	body := fmt.Sprintf("You're %s to '%s' on %s", statusText, event.Name, event.ScheduledStart.Format("Jan 2 at 3:04 PM"))

	createReq := &models.CreateNotificationRequest{
		UserID:   rsvpEvent.UserID,
		Type:     models.NotificationTypeEventRSVP,
		Title:    title,
		Body:     body,
		Data:     stringPtr(string(notifData)),
		ServerID: &event.ServerID,
	}

	if _, err := s.notifService.CreateNotification(ctx, createReq); err != nil {
		log.Printf("event_notification: failed to notify user %s: %v", rsvpEvent.UserID, err)
	}

	log.Printf("event_notification: notified user %s about RSVP to event %s", rsvpEvent.UserID, rsvpEvent.EventID)
}

// handleEventStarted sends notification to RSVPed users when event starts
func (s *EventNotificationService) handleEventStarted(data interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	eventData, ok := data.(*EventStartedEvent)
	if !ok || eventData == nil || eventData.Event == nil {
		log.Printf("event_notification: invalid event.started data")
		return
	}

	event := eventData.Event

	// Get all users who RSVPed
	rsvps, err := s.eventRepo.GetEventUsers(ctx, event.ID)
	if err != nil {
		log.Printf("event_notification: failed to get event users for %s: %v", event.ID, err)
		return
	}

	// Create notification for each user
	notifData, _ := json.Marshal(map[string]interface{}{
		"event_id":    event.ID.String(),
		"channel_id":  channelIDToString(event.ChannelID),
		"server_id":   event.ServerID.String(),
	})

	title := "Event Starting Now!"
	body := fmt.Sprintf("'%s' is starting now. Don't miss it!", event.Name)

	for _, rsvp := range rsvps {
		createReq := &models.CreateNotificationRequest{
			UserID:   rsvp.UserID,
			Type:     models.NotificationTypeEventStart,
			Title:    title,
			Body:     body,
			Data:     stringPtr(string(notifData)),
			ServerID: &event.ServerID,
			ChannelID: event.ChannelID,
		}

		if _, err := s.notifService.CreateNotification(ctx, createReq); err != nil {
			log.Printf("event_notification: failed to notify user %s: %v", rsvp.UserID, err)
		}
	}

	log.Printf("event_notification: notified %d users that event %s started", len(rsvps), event.ID)
}

// channelIDToString converts a UUID pointer to string
func channelIDToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
