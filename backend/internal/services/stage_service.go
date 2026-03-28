package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrStageNotFound            = errors.New("stage not found")
	ErrStageAlreadyExists       = errors.New("stage already exists for this channel")
	ErrStageNotActive           = errors.New("stage is not active")
	ErrStageNotLive             = errors.New("stage is not live")
	ErrStageNotPaused           = errors.New("stage is not paused")
	ErrNotStageHost             = errors.New("not the stage host")
	ErrNotStageModerator        = errors.New("not a stage moderator")
	ErrNotStageParticipant      = errors.New("not a stage participant")
	ErrCannotModifyHost         = errors.New("cannot modify host role")
	ErrMaxSpeakersReached       = errors.New("maximum speakers reached")
	ErrSpeakerRequestPending    = errors.New("speaker request already pending")
	ErrSpeakerRequestNotPending = errors.New("no pending speaker request")
	ErrNotAudienceMember        = errors.New("user is not an audience member")
	ErrNotSpeaker               = errors.New("user is not a speaker")
	ErrModeratorOnly            = errors.New("stage is moderator-only, cannot request to speak")
	ErrChannelNotStage          = errors.New("channel is not a stage channel")
	ErrNotStageHostOrMod        = errors.New("not the stage host or a moderator")
)

// StageRepository defines stage data access
type StageRepository interface {
	Create(ctx context.Context, stage *models.Stage) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Stage, error)
	GetByChannelID(ctx context.Context, channelID uuid.UUID) (*models.Stage, error)
	Update(ctx context.Context, stage *models.Stage) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddParticipant(ctx context.Context, p *models.StageParticipant) error
	GetParticipant(ctx context.Context, stageID, userID uuid.UUID) (*models.StageParticipant, error)
	UpdateParticipant(ctx context.Context, p *models.StageParticipant) error
	RemoveParticipant(ctx context.Context, stageID, userID uuid.UUID) error
	ListParticipants(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error)
	ListParticipantsByRole(ctx context.Context, stageID uuid.UUID, role models.StageRole) ([]models.StageParticipant, error)
	ListPendingRequests(ctx context.Context, stageID uuid.UUID) ([]models.StageParticipant, error)
	CountParticipantsByRole(ctx context.Context, stageID uuid.UUID) (speakers int, audience int, pending int, err error)
	UpdateParticipantMute(ctx context.Context, stageID, userID uuid.UUID, isMuted bool) error
	UpdateParticipantDeaf(ctx context.Context, stageID, userID uuid.UUID, isDeafened bool) error
	ApproveSpeakerRequest(ctx context.Context, stageID, userID uuid.UUID) error
	ClearAllParticipants(ctx context.Context, stageID uuid.UUID) error
}

// StageService handles stage channel business logic
type StageService struct {
	stageRepo    StageRepository
	channelRepo  ChannelRepository
	serverRepo   ServerRepository
	permService  *PermissionService
	voiceService *VoiceService
	eventBus     EventBus
}

// NewStageService creates a new stage service
func NewStageService(
	stageRepo StageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	voiceService *VoiceService,
	eventBus EventBus,
) *StageService {
	return &StageService{
		stageRepo:    stageRepo,
		channelRepo:  channelRepo,
		serverRepo:   serverRepo,
		permService:  permService,
		voiceService: voiceService,
		eventBus:     eventBus,
	}
}

// CreateStage creates and starts a new stage in a channel
func (s *StageService) CreateStage(ctx context.Context, channelID, userID uuid.UUID, req *models.CreateStageRequest) (*models.StageInfo, error) {
	// Verify channel exists and is a stage channel
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	if channel.Type != models.ChannelTypeStage {
		return nil, ErrChannelNotStage
	}

	// Check if a stage already exists for this channel
	existing, err := s.stageRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status != models.StageStatusEnded {
		return nil, ErrStageAlreadyExists
	}

	// Verify user is a member of the server (if this is a server channel)
	if channel.ServerID != nil {
		member, err := s.serverRepo.GetMember(ctx, *channel.ServerID, userID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
	}

	// Build stage config
	discoveryDisabled := false
	requestToSpeak := true
	moderatorOnly := false
	maxSpeakers := 10

	if req.DiscoveryDisabled != nil {
		discoveryDisabled = *req.DiscoveryDisabled
	}
	if req.RequestToSpeak != nil {
		requestToSpeak = *req.RequestToSpeak
	}
	if req.ModeratorOnly != nil {
		moderatorOnly = *req.ModeratorOnly
	}
	if req.MaxSpeakers != nil {
		maxSpeakers = *req.MaxSpeakers
	}

	now := time.Now()
	stage := &models.Stage{
		ID:                uuid.New(),
		ChannelID:         channelID,
		Topic:             req.Topic,
		Description:       req.Description,
		Status:            models.StageStatusLive,
		HostUserID:        userID,
		DiscoveryDisabled: discoveryDisabled,
		RequestToSpeak:    requestToSpeak,
		ModeratorOnly:     moderatorOnly,
		MaxSpeakers:       maxSpeakers,
		CreatedAt:         now,
		StartedAt:         &now,
		UpdatedAt:         now,
	}

	if err := s.stageRepo.Create(ctx, stage); err != nil {
		return nil, err
	}

	// Add host as participant
	hostParticipant := &models.StageParticipant{
		StageID:  stage.ID,
		UserID:   userID,
		Role:     models.StageRoleHost,
		JoinedAt: now,
		IsMuted:  false, // Host can speak
	}
	if err := s.stageRepo.AddParticipant(ctx, hostParticipant); err != nil {
		return nil, err
	}

	// Emit stage created event
	s.emitStageEvent(ctx, models.WSEventStageCreated, stage)

	return s.buildStageInfo(ctx, stage)
}

// GetStage retrieves stage info for a channel
func (s *StageService) GetStage(ctx context.Context, channelID uuid.UUID) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}
	return s.buildStageInfo(ctx, stage)
}

// UpdateStage updates stage metadata (topic, description)
func (s *StageService) UpdateStage(ctx context.Context, stageID, userID uuid.UUID, req *models.UpdateStageRequest) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host or moderator can update
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	// Apply updates
	if req.Topic != nil {
		stage.Topic = *req.Topic
	}
	if req.Description != nil {
		stage.Description = *req.Description
	}
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage updated event
	s.emitStageEvent(ctx, models.WSEventStageUpdated, stage)

	return s.buildStageInfo(ctx, stage)
}

// UpdateStageConfig updates stage configuration
func (s *StageService) UpdateStageConfig(ctx context.Context, stageID, userID uuid.UUID, req *models.StageConfig) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host can update config
	if stage.HostUserID != userID {
		return nil, ErrNotStageHost
	}

	// Apply updates
	if req.DiscoveryDisabled != nil {
		stage.DiscoveryDisabled = *req.DiscoveryDisabled
	}
	if req.RequestToSpeak != nil {
		stage.RequestToSpeak = *req.RequestToSpeak
	}
	if req.ModeratorOnly != nil {
		stage.ModeratorOnly = *req.ModeratorOnly
	}
	if req.MaxSpeakers != nil {
		stage.MaxSpeakers = *req.MaxSpeakers
	}
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage updated event
	s.emitStageEvent(ctx, models.WSEventStageUpdated, stage)

	return s.buildStageInfo(ctx, stage)
}

// PauseStage pauses a live stage
func (s *StageService) PauseStage(ctx context.Context, stageID, userID uuid.UUID) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host or moderator can pause
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	if stage.Status != models.StageStatusLive {
		return nil, ErrStageNotLive
	}

	stage.Status = models.StageStatusPaused
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage paused event
	s.emitStageEvent(ctx, models.WSEventStagePaused, stage)

	return s.buildStageInfo(ctx, stage)
}

// ResumeStage resumes a paused stage
func (s *StageService) ResumeStage(ctx context.Context, stageID, userID uuid.UUID) (*models.StageInfo, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	// Only host or moderator can resume
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	if stage.Status != models.StageStatusPaused {
		return nil, ErrStageNotPaused
	}

	stage.Status = models.StageStatusLive
	stage.UpdatedAt = time.Now()

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return nil, err
	}

	// Emit stage resumed event
	s.emitStageEvent(ctx, models.WSEventStageResumed, stage)

	return s.buildStageInfo(ctx, stage)
}

// EndStage ends a stage
func (s *StageService) EndStage(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only host or moderator can end
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return ErrNotStageHostOrMod
	}

	if stage.Status == models.StageStatusEnded {
		return nil // Already ended
	}

	now := time.Now()
	stage.Status = models.StageStatusEnded
	stage.EndedAt = &now
	stage.UpdatedAt = now

	if err := s.stageRepo.Update(ctx, stage); err != nil {
		return err
	}

	// Remove all participants
	if err := s.stageRepo.ClearAllParticipants(ctx, stageID); err != nil {
		return err
	}

	// Emit stage ended event
	s.emitStageEvent(ctx, models.WSEventStageEnded, stage)

	return nil
}

// JoinStage adds a user as an audience member to a stage
func (s *StageService) JoinStage(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}
	if stage.Status == models.StageStatusEnded || stage.Status == models.StageStatusScheduled {
		return ErrStageNotActive
	}

	// Verify user isn't already a participant
	existing, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // Already participating
	}

	// Add as audience member (muted by default)
	now := time.Now()
	participant := &models.StageParticipant{
		StageID:  stage.ID,
		UserID:   userID,
		Role:     models.StageRoleAudience,
		JoinedAt: now,
		IsMuted:  true, // Audience always muted
	}

	if err := s.stageRepo.AddParticipant(ctx, participant); err != nil {
		return err
	}

	// Emit speaker added event
	s.emitParticipantEvent(ctx, models.WSEventStageAudienceEnter, stage, userID)

	return nil
}

// LeaveStage removes a user from a stage
func (s *StageService) LeaveStage(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}

	// Host cannot leave without ending the stage
	if participant.Role == models.StageRoleHost {
		return ErrCannotModifyHost
	}

	if err := s.stageRepo.RemoveParticipant(ctx, stageID, userID); err != nil {
		return err
	}

	// Emit participant removed event
	s.emitParticipantEvent(ctx, models.WSEventStageParticipantRemove, stage, userID)

	return nil
}

// RequestToSpeak requests speaker privileges
func (s *StageService) RequestToSpeak(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}
	if stage.Status != models.StageStatusLive && stage.Status != models.StageStatusPaused {
		return ErrStageNotActive
	}

	// Check if moderator only
	if stage.ModeratorOnly {
		return ErrModeratorOnly
	}

	// Check if request to speak is enabled
	if !stage.RequestToSpeak {
		return ErrModeratorOnly
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleAudience {
		return ErrNotAudienceMember
	}

	// Check if already has pending request
	if participant.RequestedAt != nil && participant.ApprovedAt == nil {
		return ErrSpeakerRequestPending
	}

	// Check max speakers
	speakers, _, _, err := s.stageRepo.CountParticipantsByRole(ctx, stageID)
	if err != nil {
		return err
	}
	if speakers >= stage.MaxSpeakers {
		return ErrMaxSpeakersReached
	}

	now := time.Now()
	participant.RequestedAt = &now

	if err := s.stageRepo.UpdateParticipant(ctx, participant); err != nil {
		return err
	}

	// Emit request to speak event
	s.emitParticipantEvent(ctx, models.WSEventStageRequestToSpeak, stage, userID)

	return nil
}

// CancelRequestToSpeak cancels a pending speaker request
func (s *StageService) CancelRequestToSpeak(ctx context.Context, stageID, userID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return err
	}
	if participant == nil {
		return ErrNotStageParticipant
	}
	if participant.RequestedAt == nil || participant.ApprovedAt != nil {
		return ErrSpeakerRequestNotPending
	}

	now := time.Time{}
	participant.RequestedAt = &now // Clear by setting to zero

	if err := s.stageRepo.UpdateParticipant(ctx, participant); err != nil {
		return err
	}

	return nil
}

// ApproveSpeaker approves a speaker request
func (s *StageService) ApproveSpeaker(ctx context.Context, stageID, approverID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check approver is host or moderator
	approver, err := s.stageRepo.GetParticipant(ctx, stageID, approverID)
	if err != nil {
		return err
	}
	if approver == nil || (approver.Role != models.StageRoleHost && approver.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	// Get target participant
	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.RequestedAt == nil || target.ApprovedAt != nil {
		return ErrSpeakerRequestNotPending
	}

	// Check max speakers
	speakers, _, _, err := s.stageRepo.CountParticipantsByRole(ctx, stageID)
	if err != nil {
		return err
	}
	if speakers >= stage.MaxSpeakers {
		return ErrMaxSpeakersReached
	}

	if err := s.stageRepo.ApproveSpeakerRequest(ctx, stageID, targetUserID); err != nil {
		return err
	}

	// Emit speaker approved event
	s.emitParticipantEvent(ctx, models.WSEventStageSpeakerAdded, stage, targetUserID)

	return nil
}

// DenySpeaker denies a speaker request
func (s *StageService) DenySpeaker(ctx context.Context, stageID, denierID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check denier is host or moderator
	denier, err := s.stageRepo.GetParticipant(ctx, stageID, denierID)
	if err != nil {
		return err
	}
	if denier == nil || (denier.Role != models.StageRoleHost && denier.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	// Get target participant
	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	// Reset request
	now := time.Time{}
	target.RequestedAt = &now
	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	return nil
}

// PromoteToSpeaker promotes an audience member to speaker
func (s *StageService) PromoteToSpeaker(ctx context.Context, stageID, promoterID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check promoter is host or moderator
	promoter, err := s.stageRepo.GetParticipant(ctx, stageID, promoterID)
	if err != nil {
		return err
	}
	if promoter == nil || (promoter.Role != models.StageRoleHost && promoter.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	// Check max speakers
	speakers, _, _, err := s.stageRepo.CountParticipantsByRole(ctx, stageID)
	if err != nil {
		return err
	}
	if speakers >= stage.MaxSpeakers {
		return ErrMaxSpeakersReached
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.Role != models.StageRoleAudience {
		return ErrNotAudienceMember
	}

	// Promote directly (no request needed)
	now := time.Now()
	target.Role = models.StageRoleSpeaker
	target.IsMuted = false // Speakers can speak
	target.ApprovedAt = &now
	target.RequestedAt = nil

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	// Emit speaker added event
	s.emitParticipantEvent(ctx, models.WSEventStageSpeakerAdded, stage, targetUserID)

	return nil
}

// DemoteToAudience demotes a speaker to audience
func (s *StageService) DemoteToAudience(ctx context.Context, stageID, demoterID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check demoter is host or moderator
	demoter, err := s.stageRepo.GetParticipant(ctx, stageID, demoterID)
	if err != nil {
		return err
	}
	if demoter == nil || (demoter.Role != models.StageRoleHost && demoter.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.Role != models.StageRoleSpeaker && target.Role != models.StageRoleModerator {
		return ErrNotSpeaker
	}
	if target.Role == models.StageRoleHost {
		return ErrCannotModifyHost
	}

	// Demote to audience
	target.Role = models.StageRoleAudience
	target.IsMuted = true
	target.ApprovedAt = nil
	now := time.Time{}
	target.RequestedAt = &now

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	// Emit speaker removed event
	s.emitParticipantEvent(ctx, models.WSEventStageSpeakerRemove, stage, targetUserID)

	return nil
}

// AddModerator adds a moderator to the stage
func (s *StageService) AddModerator(ctx context.Context, stageID, hostID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only host can add moderators
	if stage.HostUserID != hostID {
		return ErrNotStageHost
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	target.Role = models.StageRoleModerator
	target.IsMuted = false

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	s.emitParticipantEvent(ctx, models.WSEventStageModeratorAdded, stage, targetUserID)

	return nil
}

// RemoveModerator removes a moderator from the stage
func (s *StageService) RemoveModerator(ctx context.Context, stageID, hostID, targetUserID uuid.UUID) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Only host can remove moderators
	if stage.HostUserID != hostID {
		return ErrNotStageHost
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}
	if target.Role != models.StageRoleModerator {
		return ErrNotStageModerator
	}

	// Demote to speaker
	target.Role = models.StageRoleSpeaker

	if err := s.stageRepo.UpdateParticipant(ctx, target); err != nil {
		return err
	}

	s.emitParticipantEvent(ctx, models.WSEventStageModeratorRemoved, stage, targetUserID)

	return nil
}

// ListParticipants lists all participants in a stage
func (s *StageService) ListParticipants(ctx context.Context, stageID uuid.UUID) ([]models.ParticipantInfo, error) {
	participants, err := s.stageRepo.ListParticipants(ctx, stageID)
	if err != nil {
		return nil, err
	}

	result := make([]models.ParticipantInfo, len(participants))
	for i, p := range participants {
		result[i] = models.ParticipantInfo{
			UserID:            p.UserID,
			Role:              p.Role,
			JoinedAt:          p.JoinedAt,
			IsMuted:           p.IsMuted,
			IsDeafened:        p.IsDeafened,
			HasPendingRequest: p.RequestedAt != nil && p.ApprovedAt == nil,
			RequestedAt:       p.RequestedAt,
		}
	}

	return result, nil
}

// ListPendingRequests lists pending speaker requests
func (s *StageService) ListPendingRequests(ctx context.Context, stageID, userID uuid.UUID) ([]models.ParticipantInfo, error) {
	// Only host or moderator can see requests
	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}
	if participant.Role != models.StageRoleHost && participant.Role != models.StageRoleModerator {
		return nil, ErrNotStageHostOrMod
	}

	participants, err := s.stageRepo.ListPendingRequests(ctx, stageID)
	if err != nil {
		return nil, err
	}

	result := make([]models.ParticipantInfo, len(participants))
	for i, p := range participants {
		result[i] = models.ParticipantInfo{
			UserID:            p.UserID,
			Role:              p.Role,
			JoinedAt:          p.JoinedAt,
			IsMuted:           p.IsMuted,
			IsDeafened:        p.IsDeafened,
			HasPendingRequest: true,
			RequestedAt:       p.RequestedAt,
		}
	}

	return result, nil
}

// MuteParticipant mutes a participant
func (s *StageService) MuteParticipant(ctx context.Context, stageID, moderatorID, targetUserID uuid.UUID, muted bool) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	// Check moderator has permissions
	moderator, err := s.stageRepo.GetParticipant(ctx, stageID, moderatorID)
	if err != nil {
		return err
	}
	if moderator == nil || (moderator.Role != models.StageRoleHost && moderator.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	// Audience can't unmute themselves
	if target.Role == models.StageRoleAudience && !muted {
		return ErrNotSpeaker
	}

	return s.stageRepo.UpdateParticipantMute(ctx, stageID, targetUserID, muted)
}

// DeafenParticipant deafens a participant
func (s *StageService) DeafenParticipant(ctx context.Context, stageID, moderatorID, targetUserID uuid.UUID, deafened bool) error {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return ErrStageNotFound
	}

	moderator, err := s.stageRepo.GetParticipant(ctx, stageID, moderatorID)
	if err != nil {
		return err
	}
	if moderator == nil || (moderator.Role != models.StageRoleHost && moderator.Role != models.StageRoleModerator) {
		return ErrNotStageHostOrMod
	}

	target, err := s.stageRepo.GetParticipant(ctx, stageID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotStageParticipant
	}

	return s.stageRepo.UpdateParticipantDeaf(ctx, stageID, targetUserID, deafened)
}

// GenerateVoiceToken generates a LiveKit token for stage participation
func (s *StageService) GenerateVoiceToken(ctx context.Context, stageID, userID uuid.UUID, userName, displayName, avatarURL string) (*VoiceTokenResponse, error) {
	stage, err := s.stageRepo.GetByID(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil {
		return nil, ErrStageNotFound
	}

	participant, err := s.stageRepo.GetParticipant(ctx, stageID, userID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, ErrNotStageParticipant
	}

	// Generate voice token with the channel ID as the room
	return s.voiceService.GenerateToken(ctx, userID, stage.ChannelID, userName, displayName, avatarURL)
}

// buildStageInfo builds a StageInfo response from a Stage
func (s *StageService) buildStageInfo(ctx context.Context, stage *models.Stage) (*models.StageInfo, error) {
	speakers, audience, pending, err := s.stageRepo.CountParticipantsByRole(ctx, stage.ID)
	if err != nil {
		return nil, err
	}

	return &models.StageInfo{
		ID:                stage.ID,
		ChannelID:         stage.ChannelID,
		Topic:             stage.Topic,
		Description:       stage.Description,
		Status:            stage.Status,
		HostUserID:        stage.HostUserID,
		DiscoveryDisabled: stage.DiscoveryDisabled,
		RequestToSpeak:    stage.RequestToSpeak,
		ModeratorOnly:     stage.ModeratorOnly,
		MaxSpeakers:       stage.MaxSpeakers,
		SpeakerCount:      speakers,
		AudienceCount:     audience,
		PendingCount:      pending,
		CreatedAt:         stage.CreatedAt,
		StartedAt:         stage.StartedAt,
		EndedAt:           stage.EndedAt,
	}, nil
}

// emitStageEvent emits a stage lifecycle event
func (s *StageService) emitStageEvent(ctx context.Context, eventType string, stage *models.Stage) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.Publish(eventType, stage)
}

// emitParticipantEvent emits a participant-specific event
func (s *StageService) emitParticipantEvent(ctx context.Context, eventType string, stage *models.Stage, userID uuid.UUID) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.Publish(eventType, map[string]interface{}{
		"stage_id":   stage.ID,
		"channel_id": stage.ChannelID,
		"user_id":    userID,
		"status":     stage.Status,
	})
}
