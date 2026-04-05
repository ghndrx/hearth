package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// Activity signaling message types (client → server)
const (
	ActivitySignalStart    = "ACTIVITY_START"
	ActivitySignalJoin     = "ACTIVITY_JOIN"
	ActivitySignalLeave    = "ACTIVITY_LEAVE"
	ActivitySignalEnd      = "ACTIVITY_END"
	ActivitySignalGameMove = "ACTIVITY_GAME_MOVE"
	ActivitySignalSync     = "ACTIVITY_SYNC"
)

// Activity event types (server → client broadcast)
const (
	EventActivityStart     = "ACTIVITY_START"
	EventActivityJoin      = "ACTIVITY_PARTICIPANT_JOIN"
	EventActivityLeave     = "ACTIVITY_PARTICIPANT_LEAVE"
	EventActivityEnd       = "ACTIVITY_END"
	EventActivityStateSync = "ACTIVITY_STATE_SYNC"
	EventActivityGameMove  = "ACTIVITY_GAME_MOVE"
)

// ActivityStartData represents data for starting an activity
type ActivityStartData struct {
	ChannelID       uuid.UUID       `json:"channel_id"`
	ServerID        uuid.UUID       `json:"server_id"`
	ActivityType    string          `json:"activity_type"`
	MaxParticipants int             `json:"max_participants,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
}

// ActivityJoinData represents data for joining an activity
type ActivityJoinData struct {
	ActivityID uuid.UUID `json:"activity_id"`
}

// ActivityLeaveData represents data for leaving an activity
type ActivityLeaveData struct {
	ActivityID uuid.UUID `json:"activity_id"`
}

// ActivityEndData represents data for ending an activity
type ActivityEndData struct {
	ActivityID uuid.UUID `json:"activity_id"`
}

// ActivityGameMoveData represents a game move via WebSocket
type ActivityGameMoveData struct {
	ActivityID uuid.UUID       `json:"activity_id"`
	Action     string          `json:"action"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// ActivitySyncData represents a state sync request
type ActivitySyncData struct {
	ActivityID uuid.UUID `json:"activity_id"`
}

// ActivitySignalingService handles voice activity signaling via WebSocket
type ActivitySignalingService struct {
	hub             HubInterface
	activityService *services.VoiceActivityService

	// Track which activities are active in which channels
	// channelID -> activityID
	activeActivities   map[uuid.UUID]uuid.UUID
	activeActivitiesMu sync.RWMutex
}

// NewActivitySignalingService creates a new activity signaling service
func NewActivitySignalingService(hub HubInterface, activityService *services.VoiceActivityService) *ActivitySignalingService {
	return &ActivitySignalingService{
		hub:              hub,
		activityService:  activityService,
		activeActivities: make(map[uuid.UUID]uuid.UUID),
	}
}

// HandleActivityMessage handles incoming activity-related messages
func (s *ActivitySignalingService) HandleActivityMessage(ctx context.Context, client *Client, sessionID string, msgType string, data json.RawMessage) error {
	switch msgType {
	case ActivitySignalStart:
		return s.handleStart(ctx, client, data)
	case ActivitySignalJoin:
		return s.handleJoin(ctx, client, data)
	case ActivitySignalLeave:
		return s.handleLeave(ctx, client, data)
	case ActivitySignalEnd:
		return s.handleEnd(ctx, client, data)
	case ActivitySignalGameMove:
		return s.handleGameMove(ctx, client, data)
	case ActivitySignalSync:
		return s.handleSync(ctx, client, data)
	default:
		log.Printf("[Activity] Unknown message type: %s", msgType)
		return nil
	}
}

func (s *ActivitySignalingService) handleStart(ctx context.Context, client *Client, data json.RawMessage) error {
	var startData ActivityStartData
	if err := json.Unmarshal(data, &startData); err != nil {
		return err
	}

	log.Printf("[Activity] User %s starting %s in channel %s", client.UserID, startData.ActivityType, startData.ChannelID)

	result, err := s.activityService.StartActivity(ctx, startData.ChannelID, startData.ServerID, client.UserID, &models.StartActivityRequest{
		ActivityType:    models.VoiceActivityType(startData.ActivityType),
		MaxParticipants: startData.MaxParticipants,
		Metadata:        startData.Metadata,
	})
	if err != nil {
		log.Printf("[Activity] Failed to start: %v", err)
		return nil
	}

	s.activeActivitiesMu.Lock()
	s.activeActivities[startData.ChannelID] = result.ID
	s.activeActivitiesMu.Unlock()

	s.hub.SendToServer(startData.ServerID, &Event{
		Op:   OpDispatch,
		Type: EventActivityStart,
		Data: result,
	})

	return nil
}

func (s *ActivitySignalingService) handleJoin(ctx context.Context, client *Client, data json.RawMessage) error {
	var joinData ActivityJoinData
	if err := json.Unmarshal(data, &joinData); err != nil {
		return err
	}

	log.Printf("[Activity] User %s joining activity %s", client.UserID, joinData.ActivityID)

	result, err := s.activityService.JoinActivity(ctx, joinData.ActivityID, client.UserID)
	if err != nil {
		log.Printf("[Activity] Failed to join: %v", err)
		return nil
	}

	s.hub.SendToServer(result.ServerID, &Event{
		Op:   OpDispatch,
		Type: EventActivityJoin,
		Data: map[string]interface{}{
			"activity_id": joinData.ActivityID.String(),
			"user_id":     client.UserID.String(),
			"channel_id":  result.ChannelID.String(),
			"server_id":   result.ServerID.String(),
		},
	})

	return nil
}

func (s *ActivitySignalingService) handleLeave(ctx context.Context, client *Client, data json.RawMessage) error {
	var leaveData ActivityLeaveData
	if err := json.Unmarshal(data, &leaveData); err != nil {
		return err
	}

	log.Printf("[Activity] User %s leaving activity %s", client.UserID, leaveData.ActivityID)

	// Get activity info before leaving for broadcast
	activity, _ := s.activityService.GetActivity(ctx, leaveData.ActivityID)

	if err := s.activityService.LeaveActivity(ctx, leaveData.ActivityID, client.UserID); err != nil {
		log.Printf("[Activity] Failed to leave: %v", err)
		return nil
	}

	if activity != nil {
		s.hub.SendToServer(activity.ServerID, &Event{
			Op:   OpDispatch,
			Type: EventActivityLeave,
			Data: map[string]interface{}{
				"activity_id": leaveData.ActivityID.String(),
				"user_id":     client.UserID.String(),
				"channel_id":  activity.ChannelID.String(),
				"server_id":   activity.ServerID.String(),
			},
		})
	}

	return nil
}

func (s *ActivitySignalingService) handleEnd(ctx context.Context, client *Client, data json.RawMessage) error {
	var endData ActivityEndData
	if err := json.Unmarshal(data, &endData); err != nil {
		return err
	}

	log.Printf("[Activity] User %s ending activity %s", client.UserID, endData.ActivityID)

	activity, _ := s.activityService.GetActivity(ctx, endData.ActivityID)

	if err := s.activityService.EndActivity(ctx, endData.ActivityID, client.UserID); err != nil {
		log.Printf("[Activity] Failed to end: %v", err)
		return nil
	}

	if activity != nil {
		s.activeActivitiesMu.Lock()
		delete(s.activeActivities, activity.ChannelID)
		s.activeActivitiesMu.Unlock()

		s.hub.SendToServer(activity.ServerID, &Event{
			Op:   OpDispatch,
			Type: EventActivityEnd,
			Data: map[string]interface{}{
				"activity_id": endData.ActivityID.String(),
				"channel_id":  activity.ChannelID.String(),
				"server_id":   activity.ServerID.String(),
				"ended_by":    client.UserID.String(),
			},
		})
	}

	return nil
}

func (s *ActivitySignalingService) handleGameMove(ctx context.Context, client *Client, data json.RawMessage) error {
	var moveData ActivityGameMoveData
	if err := json.Unmarshal(data, &moveData); err != nil {
		return err
	}

	log.Printf("[Activity] User %s game move in activity %s: %s", client.UserID, moveData.ActivityID, moveData.Action)

	state, err := s.activityService.ProcessGameMove(ctx, moveData.ActivityID, client.UserID, &models.GameMoveRequest{
		Action: moveData.Action,
		Data:   moveData.Data,
	})
	if err != nil {
		log.Printf("[Activity] Game move failed: %v", err)
		return nil
	}

	// Get the activity to know which server to broadcast to
	activity, _ := s.activityService.GetActivity(ctx, moveData.ActivityID)
	if activity != nil {
		s.hub.SendToServer(activity.ServerID, &Event{
			Op:   OpDispatch,
			Type: EventActivityGameMove,
			Data: map[string]interface{}{
				"activity_id": moveData.ActivityID.String(),
				"channel_id":  activity.ChannelID.String(),
				"user_id":     client.UserID.String(),
				"action":      moveData.Action,
				"state":       state,
			},
		})
	}

	return nil
}

func (s *ActivitySignalingService) handleSync(ctx context.Context, client *Client, data json.RawMessage) error {
	var syncData ActivitySyncData
	if err := json.Unmarshal(data, &syncData); err != nil {
		return err
	}

	state, err := s.activityService.GetGameState(ctx, syncData.ActivityID)
	if err != nil || state == nil {
		return nil
	}

	// Send state only to the requesting user
	s.hub.SendToUser(client.UserID, &Event{
		Op:   OpDispatch,
		Type: EventActivityStateSync,
		Data: map[string]interface{}{
			"activity_id": syncData.ActivityID.String(),
			"state":       state,
		},
	})

	return nil
}
