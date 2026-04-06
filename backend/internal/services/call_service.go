package services

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// CallRepositoryInterface defines the interface for call data access
type CallRepositoryInterface interface {
	Create(ctx context.Context, call *models.Call) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Call, error)
	GetActiveByChannel(ctx context.Context, channelID uuid.UUID) ([]*models.Call, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.CallStatus, endReason string) error
	AddParticipant(ctx context.Context, participant *models.CallParticipant) error
	RemoveParticipant(ctx context.Context, callID, userID uuid.UUID) error
	GetParticipants(ctx context.Context, callID uuid.UUID) ([]models.CallParticipant, error)
	GetActiveParticipantCount(ctx context.Context, callID uuid.UUID) (int, error)
	CreateSession(ctx context.Context, session *models.CallSession) error
	EndSession(ctx context.Context, sessionID uuid.UUID) error
}

// CallService handles call business logic
type CallService struct {
	callRepo CallRepositoryInterface
}

// NewCallService creates a new call service
func NewCallService(callRepo CallRepositoryInterface) *CallService {
	return &CallService{
		callRepo: callRepo,
	}
}

var (
	ErrCallNotFound      = errors.New("call not found")
	ErrCallAlreadyEnded  = errors.New("call has already ended")
	ErrAlreadyInCall     = errors.New("user is already in this call")
	ErrNotInCall         = errors.New("user is not in this call")
	ErrInvalidCallType   = errors.New("invalid call type")
)

// CreateCall creates a new call
func (s *CallService) CreateCall(ctx context.Context, initiatorID, channelID uuid.UUID, serverID *uuid.UUID, callType models.CallType) (*models.Call, error) {
	if callType != models.CallTypeDirect && callType != models.CallTypeGroup && callType != models.CallTypeChannel {
		return nil, ErrInvalidCallType
	}

	call := &models.Call{
		ChannelID:   channelID,
		ServerID:    serverID,
		InitiatorID: initiatorID,
		Type:        callType,
		Status:      models.CallStatusRinging,
		StartedAt:   time.Now(),
	}

	if err := s.callRepo.Create(ctx, call); err != nil {
		return nil, err
	}

	// Add initiator as first participant
	participant := &models.CallParticipant{
		CallID:    call.ID,
		UserID:    initiatorID,
		IsMuted:   true,
		IsVideoOn: false,
	}
	if err := s.callRepo.AddParticipant(ctx, participant); err != nil {
		return nil, err
	}

	return s.callRepo.GetByID(ctx, call.ID)
}

// GetCall retrieves a call by ID
func (s *CallService) GetCall(ctx context.Context, callID uuid.UUID) (*models.Call, error) {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, ErrCallNotFound
	}
	return call, nil
}

// JoinCall adds a user to an existing call
func (s *CallService) JoinCall(ctx context.Context, callID, userID uuid.UUID) (*models.JoinCallResponse, error) {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return nil, err
	}
	if call == nil {
		return nil, ErrCallNotFound
	}
	if call.Status == models.CallStatusEnded {
		return nil, ErrCallAlreadyEnded
	}

	// Update call status to active if it was ringing
	if call.Status == models.CallStatusRinging {
		if err := s.callRepo.UpdateStatus(ctx, callID, models.CallStatusActive, ""); err != nil {
			return nil, err
		}
	}

	participant := &models.CallParticipant{
		CallID:    callID,
		UserID:    userID,
		IsMuted:   true,
		IsVideoOn: false,
	}
	if err := s.callRepo.AddParticipant(ctx, participant); err != nil {
		return nil, err
	}

	return &models.JoinCallResponse{
		CallID:     callID,
		UserID:     userID,
		JoinedAt:   participant.JoinedAt,
		ICEServers: s.getICEServers(),
	}, nil
}

// LeaveCall removes a user from a call
func (s *CallService) LeaveCall(ctx context.Context, callID, userID uuid.UUID) error {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return err
	}
	if call == nil {
		return ErrCallNotFound
	}

	if err := s.callRepo.RemoveParticipant(ctx, callID, userID); err != nil {
		return err
	}

	// Check if any participants remain
	count, err := s.callRepo.GetActiveParticipantCount(ctx, callID)
	if err != nil {
		return err
	}

	// End call if no participants remain
	if count == 0 {
		return s.callRepo.UpdateStatus(ctx, callID, models.CallStatusEnded, string(models.CallEndReasonCompleted))
	}

	return nil
}

// EndCall ends a call with a given reason
func (s *CallService) EndCall(ctx context.Context, callID uuid.UUID, reason models.CallEndReason) error {
	call, err := s.callRepo.GetByID(ctx, callID)
	if err != nil {
		return err
	}
	if call == nil {
		return ErrCallNotFound
	}
	if call.Status == models.CallStatusEnded {
		return ErrCallAlreadyEnded
	}

	return s.callRepo.UpdateStatus(ctx, callID, models.CallStatusEnded, string(reason))
}

// GetActiveCallsForChannel returns active calls in a channel
func (s *CallService) GetActiveCallsForChannel(ctx context.Context, channelID uuid.UUID) ([]*models.Call, error) {
	return s.callRepo.GetActiveByChannel(ctx, channelID)
}

// getICEServers returns the configured ICE servers
func (s *CallService) getICEServers() []models.ICEServer {
	servers := []models.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	}

	// TODO: Add configurable TURN server support
	turnURL := os.Getenv("TURN_SERVER_URL")
	if turnURL != "" {
		servers = append(servers, models.ICEServer{
			URLs:       []string{turnURL},
			Username:   os.Getenv("TURN_USERNAME"),
			Credential: os.Getenv("TURN_CREDENTIAL"),
		})
	}

	return servers
}
