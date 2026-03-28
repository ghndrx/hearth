package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrStageNotFound        = errors.New("stage instance not found")
	ErrStageAlreadyActive   = errors.New("a stage is already active in this channel")
	ErrStageNotStageChannel = errors.New("channel is not a stage channel")
	ErrStageNotStartedBy    = errors.New("only the stage creator or moderators can end this stage")
	ErrStageAlreadyJoined   = errors.New("user is already in this stage")
	ErrStageNotParticipant  = errors.New("user is not a participant in this stage")
	ErrStageEnded           = errors.New("stage has already ended")
)

// StageRepository defines data access for stage instances
type StageRepository interface {
	Create(ctx context.Context, stage *models.StageInstance) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.StageInstance, error)
	GetActiveByChannel(ctx context.Context, channelID uuid.UUID) (*models.StageInstance, error)
	End(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, stage *models.StageInstance) error
	AddParticipant(ctx context.Context, stageID, userID uuid.UUID, role models.StageParticipantRole) error
	RemoveParticipant(ctx context.Context, stageID, userID uuid.UUID) error
	GetParticipant(ctx context.Context, stageID, userID uuid.UUID) (*models.StageParticipant, error)
	GetParticipants(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error)
	UpdateParticipantRole(ctx context.Context, stageID, userID uuid.UUID, role models.StageParticipantRole) error
	RemoveAllParticipants(ctx context.Context, stageID uuid.UUID) error
	CountParticipants(ctx context.Context, stageID uuid.UUID) (speakers int, audience int, err error)
}

// StageService handles stage channel business logic
type StageService struct {
	stageRepo   StageRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	permService *PermissionService
	eventBus    EventBus
}

// NewStageService creates a new stage service
func NewStageService(
	stageRepo StageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	eventBus EventBus,
) *StageService {
	return &StageService{
		stageRepo:   stageRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		permService: permService,
		eventBus:    eventBus,
	}
}

// StartStage creates a new stage instance in a stage channel
func (s *StageService) StartStage(ctx context.Context, channelID, userID uuid.UUID, req *models.CreateStageRequest) (*models.StageInstance, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	if channel.Type != models.ChannelTypeStage {
		return nil, ErrStageNotStageChannel
	}
	if channel.ServerID == nil {
		return nil, ErrStageNotStageChannel
	}

	// Check membership
	member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Check permission to manage channels (required to start a stage)
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermManageChannels); err != nil {
			return nil, err
		}
	}

	// Check no active stage exists
	existing, err := s.stageRepo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrStageAlreadyActive
	}

	privacyLevel := models.StagePrivacyGuildOnly
	if req.PrivacyLevel != 0 {
		privacyLevel = req.PrivacyLevel
	}

	stage := &models.StageInstance{
		ID:           uuid.New(),
		ChannelID:    channelID,
		ServerID:     *channel.ServerID,
		Topic:        req.Topic,
		PrivacyLevel: privacyLevel,
		StartedBy:    userID,
		SpeakerCount: 1, // creator is a speaker
		CreatedAt:    time.Now(),
	}

	if err := s.stageRepo.Create(ctx, stage); err != nil {
		return nil, err
	}

	// Add creator as speaker
	if err := s.stageRepo.AddParticipant(ctx, stage.ID, userID, models.StageRoleSpeaker); err != nil {
		return nil, err
	}

	s.eventBus.Publish("stage.created", &models.StageInstanceCreateEvent{
		StageInstance: stage,
		ChannelID:     channelID,
		ServerID:      *channel.ServerID,
	})

	return stage, nil
}

// GetStage returns the active stage for a channel
func (s *StageService) GetStage(ctx context.Context, channelID uuid.UUID) (*models.StageInstance, error) {
	stage, err := s.stageRepo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Populate participants
	participants, err := s.stageRepo.GetParticipants(ctx, stage.ID)
	if err != nil {
		return nil, err
	}

	for _, p := range participants {
		if p.Role == models.StageRoleSpeaker {
			stage.Speakers = append(stage.Speakers, p)
		} else {
			stage.Audience = append(stage.Audience, p)
		}
	}

	return stage, nil
}

// EndStage ends an active stage
func (s *StageService) EndStage(ctx context.Context, channelID, userID uuid.UUID) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	stage, err := s.stageRepo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only creator or moderators can end
	if stage.StartedBy != userID {
		if s.permService != nil && channel.ServerID != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermManageChannels); err != nil {
				return ErrStageNotStartedBy
			}
		} else {
			return ErrStageNotStartedBy
		}
	}

	if err := s.stageRepo.RemoveAllParticipants(ctx, stage.ID); err != nil {
		return err
	}

	if err := s.stageRepo.End(ctx, stage.ID); err != nil {
		return err
	}

	s.eventBus.Publish("stage.deleted", &models.StageInstanceDeleteEvent{
		StageID:   stage.ID,
		ChannelID: channelID,
		ServerID:  stage.ServerID,
	})

	return nil
}

// JoinStage adds a user to a stage as audience
func (s *StageService) JoinStage(ctx context.Context, channelID, userID uuid.UUID) (*models.StageParticipant, error) {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	stage, err := s.stageRepo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}
	if stage.EndedAt != nil {
		return nil, ErrStageEnded
	}

	// Check membership
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	// Check not already in stage
	existing, err := s.stageRepo.GetParticipant(ctx, stage.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrStageAlreadyJoined
	}

	if err := s.stageRepo.AddParticipant(ctx, stage.ID, userID, models.StageRoleAudience); err != nil {
		return nil, err
	}

	// Update counts
	speakers, audience, err := s.stageRepo.CountParticipants(ctx, stage.ID)
	if err == nil {
		stage.SpeakerCount = speakers
		stage.AudienceCount = audience
		_ = s.stageRepo.Update(ctx, stage)
	}

	participant := &models.StageParticipant{
		StageID:  stage.ID,
		UserID:   userID,
		Role:     models.StageRoleAudience,
		JoinedAt: time.Now(),
	}

	s.eventBus.Publish("stage.participant_added", &models.StageParticipantEvent{
		StageID:   stage.ID,
		UserID:    userID,
		Role:      models.StageRoleAudience,
		ChannelID: channelID,
		ServerID:  stage.ServerID,
	})

	return participant, nil
}

// LeaveStage removes a user from a stage
func (s *StageService) LeaveStage(ctx context.Context, channelID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	existing, err := s.stageRepo.GetParticipant(ctx, stage.ID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrStageNotParticipant
	}

	if err := s.stageRepo.RemoveParticipant(ctx, stage.ID, userID); err != nil {
		return err
	}

	// Update counts
	speakers, audience, err := s.stageRepo.CountParticipants(ctx, stage.ID)
	if err == nil {
		stage.SpeakerCount = speakers
		stage.AudienceCount = audience
		_ = s.stageRepo.Update(ctx, stage)
	}

	s.eventBus.Publish("stage.participant_removed", &models.StageParticipantEvent{
		StageID:   stage.ID,
		UserID:    userID,
		Role:      existing.Role,
		ChannelID: channelID,
		ServerID:  stage.ServerID,
	})

	return nil
}

// UpdateParticipantRole changes a participant between speaker and audience
func (s *StageService) UpdateParticipantRole(ctx context.Context, channelID, userID, targetUserID uuid.UUID, role models.StageParticipantRole) error {
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channel == nil {
		return ErrChannelNotFound
	}

	stage, err := s.stageRepo.GetActiveByChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only stage creator or moderators can promote/demote
	if stage.StartedBy != userID {
		if s.permService != nil && channel.ServerID != nil {
			if err := s.permService.RequirePermission(ctx, *channel.ServerID, userID, models.PermManageChannels); err != nil {
				return ErrStageNotStartedBy
			}
		} else {
			return ErrStageNotStartedBy
		}
	}

	existing, err := s.stageRepo.GetParticipant(ctx, stage.ID, targetUserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrStageNotParticipant
	}

	if err := s.stageRepo.UpdateParticipantRole(ctx, stage.ID, targetUserID, role); err != nil {
		return err
	}

	// Update counts
	speakers, audience, err := s.stageRepo.CountParticipants(ctx, stage.ID)
	if err == nil {
		stage.SpeakerCount = speakers
		stage.AudienceCount = audience
		_ = s.stageRepo.Update(ctx, stage)
	}

	s.eventBus.Publish("stage.participant_updated", &models.StageParticipantEvent{
		StageID:   stage.ID,
		UserID:    targetUserID,
		Role:      role,
		ChannelID: channelID,
		ServerID:  stage.ServerID,
	})

	return nil
}
