package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrInviteNotFound    = errors.New("invite not found")
	ErrInviteExpired     = errors.New("invite has expired")
	ErrInviteMaxUses     = errors.New("invite has reached maximum uses")
	ErrAlreadyMember     = errors.New("already a member of this server")
	ErrBannedFromServer  = errors.New("you are banned from this server")
)

// InviteRepository defines invite data operations
type InviteRepository interface {
	Create(ctx context.Context, invite *models.Invite) error
	GetByCode(ctx context.Context, code string) (*models.Invite, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error)
	IncrementUses(ctx context.Context, code string) error
	Delete(ctx context.Context, code string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// BanRepository defines ban data operations
type BanRepository interface {
	Create(ctx context.Context, ban *models.Ban) error
	GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error)
	Delete(ctx context.Context, serverID, userID uuid.UUID) error
}

// InviteService handles invite-related business logic
type InviteService struct {
	inviteRepo InviteRepository
	banRepo    BanRepository
	serverRepo ServerRepository
	cache      CacheService
	eventBus   EventBus
}

// NewInviteService creates a new invite service
func NewInviteService(
	inviteRepo InviteRepository,
	banRepo BanRepository,
	serverRepo ServerRepository,
	cache CacheService,
	eventBus EventBus,
) *InviteService {
	return &InviteService{
		inviteRepo: inviteRepo,
		banRepo:    banRepo,
		serverRepo: serverRepo,
		cache:      cache,
		eventBus:   eventBus,
	}
}

// CreateInviteRequest represents an invite creation request
type CreateInviteRequest struct {
	ServerID   uuid.UUID
	ChannelID  uuid.UUID
	CreatorID  uuid.UUID
	MaxUses    int           // 0 = unlimited
	MaxAge     time.Duration // 0 = never expires
	Temporary  bool
}

// CreateInvite creates a new server invite
func (s *InviteService) CreateInvite(ctx context.Context, req *CreateInviteRequest) (*models.Invite, error) {
	// Verify creator is a member
	member, err := s.serverRepo.GetMember(ctx, req.ServerID, req.CreatorID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// TODO: Check CREATE_INVITE permission

	// Generate unique code
	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if req.MaxAge > 0 {
		t := time.Now().Add(req.MaxAge)
		expiresAt = &t
	}

	invite := &models.Invite{
		Code:      code,
		ServerID:  req.ServerID,
		ChannelID: req.ChannelID,
		CreatorID: req.CreatorID,
		MaxUses:   req.MaxUses,
		Uses:      0,
		ExpiresAt: expiresAt,
		Temporary: req.Temporary,
		CreatedAt: time.Now(),
	}

	if err := s.inviteRepo.Create(ctx, invite); err != nil {
		return nil, err
	}

	return invite, nil
}

// GetInvite retrieves an invite by code
func (s *InviteService) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	invite, err := s.inviteRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, ErrInviteNotFound
	}

	// Load server info
	server, _ := s.serverRepo.GetByID(ctx, invite.ServerID)
	invite.Server = server

	return invite, nil
}

// UseInvite joins a user to a server via invite
func (s *InviteService) UseInvite(ctx context.Context, code string, userID uuid.UUID) (*models.Server, error) {
	invite, err := s.inviteRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, ErrInviteNotFound
	}

	// Check if invite is valid
	if invite.IsExpired() {
		return nil, ErrInviteExpired
	}
	if invite.IsMaxUsesReached() {
		return nil, ErrInviteMaxUses
	}

	// Check if user is banned
	ban, _ := s.banRepo.GetByServerAndUser(ctx, invite.ServerID, userID)
	if ban != nil {
		return nil, ErrBannedFromServer
	}

	// Check if already a member
	existing, _ := s.serverRepo.GetMember(ctx, invite.ServerID, userID)
	if existing != nil {
		return nil, ErrAlreadyMember
	}

	// Add member
	member := &models.Member{
		ServerID:  invite.ServerID,
		UserID:    userID,
		JoinedAt:  time.Now(),
		Temporary: invite.Temporary,
	}

	if err := s.serverRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	// Increment invite uses
	_ = s.inviteRepo.IncrementUses(ctx, code)

	// Get server to return
	server, err := s.serverRepo.GetByID(ctx, invite.ServerID)
	if err != nil {
		return nil, err
	}

	s.eventBus.Publish("server.member_joined", &MemberJoinedEvent{
		ServerID: invite.ServerID,
		UserID:   userID,
		InviteCode: code,
	})

	return server, nil
}

// DeleteInvite deletes an invite
func (s *InviteService) DeleteInvite(ctx context.Context, code string, requesterID uuid.UUID) error {
	invite, err := s.inviteRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if invite == nil {
		return ErrInviteNotFound
	}

	// Check permissions (creator or MANAGE_SERVER)
	if invite.CreatorID != requesterID {
		member, err := s.serverRepo.GetMember(ctx, invite.ServerID, requesterID)
		if err != nil || member == nil {
			return ErrNotServerMember
		}
		// TODO: Check MANAGE_SERVER permission
	}

	return s.inviteRepo.Delete(ctx, code)
}

// GetServerInvites gets all invites for a server
func (s *InviteService) GetServerInvites(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Invite, error) {
	// Verify requester is a member
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// TODO: Check MANAGE_SERVER permission

	return s.inviteRepo.GetByServerID(ctx, serverID)
}

// CleanupExpiredInvites removes expired invites
func (s *InviteService) CleanupExpiredInvites(ctx context.Context) (int64, error) {
	return s.inviteRepo.DeleteExpired(ctx)
}

// Ban operations

// BanMember bans a user from a server
func (s *InviteService) BanMember(ctx context.Context, serverID, userID, moderatorID uuid.UUID, reason string) error {
	// Check moderator permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, moderatorID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	// TODO: Check BAN_MEMBERS permission

	// Can't ban yourself
	if userID == moderatorID {
		return errors.New("cannot ban yourself")
	}

	// Remove from server first
	_ = s.serverRepo.RemoveMember(ctx, serverID, userID)

	// Create ban
	ban := &models.Ban{
		ServerID:  serverID,
		UserID:    userID,
		Reason:    reason,
		BannedBy:  moderatorID,
		CreatedAt: time.Now(),
	}

	if err := s.banRepo.Create(ctx, ban); err != nil {
		return err
	}

	s.eventBus.Publish("server.member_banned", &MemberBannedEvent{
		ServerID:    serverID,
		UserID:      userID,
		ModeratorID: moderatorID,
		Reason:      reason,
	})

	return nil
}

// UnbanMember removes a ban
func (s *InviteService) UnbanMember(ctx context.Context, serverID, userID, moderatorID uuid.UUID) error {
	// Check moderator permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, moderatorID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	// TODO: Check BAN_MEMBERS permission

	if err := s.banRepo.Delete(ctx, serverID, userID); err != nil {
		return err
	}

	s.eventBus.Publish("server.member_unbanned", &MemberUnbannedEvent{
		ServerID:    serverID,
		UserID:      userID,
		ModeratorID: moderatorID,
	})

	return nil
}

// GetServerBans gets all bans for a server
func (s *InviteService) GetServerBans(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Ban, error) {
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}
	// TODO: Check BAN_MEMBERS permission

	return s.banRepo.GetByServerID(ctx, serverID)
}

// Helper functions

func generateInviteCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// Events

type MemberJoinedEvent struct {
	ServerID   uuid.UUID
	UserID     uuid.UUID
	InviteCode string
}

type MemberBannedEvent struct {
	ServerID    uuid.UUID
	UserID      uuid.UUID
	ModeratorID uuid.UUID
	Reason      string
}

type MemberUnbannedEvent struct {
	ServerID    uuid.UUID
	UserID      uuid.UUID
	ModeratorID uuid.UUID
}
