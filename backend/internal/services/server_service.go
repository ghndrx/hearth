package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/ratelimit"
)

var (
	ErrMaxServersReached = errors.New("maximum servers reached")
)

// ServerRepository defines the interface for server data access
type ServerRepository interface {
	Create(ctx context.Context, server *models.Server) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error)
	Update(ctx context.Context, server *models.Server) error
	Delete(ctx context.Context, id uuid.UUID) error
	TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error

	// Members
	GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error)
	GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error)
	GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
	GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error)
	AddMember(ctx context.Context, member *models.Member) error
	UpdateMember(ctx context.Context, member *models.Member) error
	RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error
	GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error)

	// User's servers
	GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error)
	GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error)

	// Bans
	GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error)
	AddBan(ctx context.Context, ban *models.Ban) error
	RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error
	GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error)

	// Invites
	CreateInvite(ctx context.Context, invite *models.Invite) error
	GetInvite(ctx context.Context, code string) (*models.Invite, error)
	GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error)
	GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error)
	DeleteInvite(ctx context.Context, code string) error
	IncrementInviteUses(ctx context.Context, code string) error

	// Invite analytics
	LogInviteUse(ctx context.Context, log *models.InviteUseLog) error
	GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error)
	GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error)
}

// ServerService handles server-related business logic
type ServerService struct {
	repo             ServerRepository
	channelRepo      ChannelRepository
	roleRepo         RoleRepository
	messageRepo      MessageRepository
	quotaService     *QuotaService
	permService      *PermissionService
	cache            CacheService
	eventBus         EventBus
	inviteRateLimiter InviteRateLimiter
}

// NewServerService creates a new server service
func NewServerService(
	repo ServerRepository,
	channelRepo ChannelRepository,
	roleRepo RoleRepository,
	messageRepo MessageRepository,
	quotaService *QuotaService,
	permService *PermissionService,
	cache CacheService,
	eventBus EventBus,
) *ServerService {
	return &ServerService{
		repo:         repo,
		channelRepo:  channelRepo,
		roleRepo:     roleRepo,
		messageRepo:  messageRepo,
		quotaService: quotaService,
		permService:  permService,
		cache:        cache,
		eventBus:     eventBus,
	}
}

// SetInviteRateLimiter sets the rate limiter for invite creation
func (s *ServerService) SetInviteRateLimiter(rateLimiter InviteRateLimiter) {
	s.inviteRateLimiter = rateLimiter
}

// CreateServer creates a new server
func (s *ServerService) CreateServer(ctx context.Context, ownerID uuid.UUID, name, icon string) (*models.Server, error) {
	// Check quota
	ownedCount, err := s.repo.GetOwnedServersCount(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	limits, err := s.quotaService.GetEffectiveLimits(ctx, ownerID, nil)
	if err != nil {
		return nil, err
	}

	if limits.MaxServersOwned > 0 && ownedCount >= limits.MaxServersOwned {
		return nil, ErrMaxServersReached
	}

	// Create server
	var iconURL *string
	if icon != "" {
		iconURL = &icon
	}
	server := &models.Server{
		ID:        uuid.New(),
		Name:      name,
		IconURL:   iconURL,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, server); err != nil {
		return nil, err
	}

	// Create @everyone role (color 0x99AAB5 = 10066613 in decimal)
	everyoneRole := &models.Role{
		ID:          uuid.New(),
		ServerID:    server.ID,
		Name:        "@everyone",
		Color:       0x99AAB5,
		Position:    0,
		Permissions: models.DefaultPermissions,
		IsDefault:   true,
		CreatedAt:   time.Now(),
	}
	if err := s.roleRepo.Create(ctx, everyoneRole); err != nil {
		// Rollback server creation
		_ = s.repo.Delete(ctx, server.ID)
		return nil, err
	}

	// Create default channels
	defaultChannels := []struct {
		name     string
		chanType models.ChannelType
	}{
		{"general", models.ChannelTypeText},
		{"General", models.ChannelTypeVoice},
	}

	for i, ch := range defaultChannels {
		channel := &models.Channel{
			ID:        uuid.New(),
			ServerID:  &server.ID,
			Name:      ch.name,
			Type:      ch.chanType,
			Position:  i,
			CreatedAt: time.Now(),
		}
		if err := s.channelRepo.Create(ctx, channel); err != nil {
			// Continue anyway, not critical
			continue
		}
	}

	// Add owner as member with all roles
	member := &models.Member{
		UserID:   ownerID,
		ServerID: server.ID,
		Nickname: nil,
		JoinedAt: time.Now(),
		Roles:    []uuid.UUID{everyoneRole.ID},
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	// Emit event
	s.eventBus.Publish("server.created", &ServerCreatedEvent{
		Server:  server,
		OwnerID: ownerID,
	})

	return server, nil
}

// GetServer retrieves a server by ID
func (s *ServerService) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	server, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}
	return server, nil
}

// UpdateServer updates server settings
func (s *ServerService) UpdateServer(ctx context.Context, id uuid.UUID, requesterID uuid.UUID, updates *models.ServerUpdate) (*models.Server, error) {
	server, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}

	// Check permissions (owner or admin)
	if server.OwnerID != requesterID {
		member, err := s.repo.GetMember(ctx, id, requesterID)
		if err != nil || member == nil {
			return nil, ErrNotServerMember
		}
		// Require MANAGE_SERVER permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, id, requesterID, models.PermManageServer); err != nil {
				return nil, err
			}
		}
	}

	// Apply updates
	if updates.Name != nil {
		server.Name = *updates.Name
	}
	if updates.IconURL != nil {
		server.IconURL = updates.IconURL
	}
	if updates.BannerURL != nil {
		server.BannerURL = updates.BannerURL
	}
	if updates.Description != nil {
		server.Description = updates.Description
	}

	server.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, server); err != nil {
		return nil, err
	}

	s.eventBus.Publish("server.updated", &ServerUpdatedEvent{
		Server: server,
	})

	return server, nil
}

// DeleteServer deletes a server (owner only)
func (s *ServerService) DeleteServer(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) error {
	server, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	if server.OwnerID != requesterID {
		return ErrNotServerOwner
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.eventBus.Publish("server.deleted", &ServerDeletedEvent{
		ServerID: id,
		OwnerID:  requesterID,
	})

	return nil
}

// TransferOwnership transfers server ownership to a new owner (owner only)
func (s *ServerService) TransferOwnership(ctx context.Context, serverID, requesterID, newOwnerID uuid.UUID) (*models.Server, error) {
	server, err := s.repo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}

	// Only owner can transfer
	if server.OwnerID != requesterID {
		return nil, ErrNotServerOwner
	}

	// Cannot transfer to self
	if requesterID == newOwnerID {
		return nil, ErrSelfAction
	}

	// New owner must be a member
	member, err := s.repo.GetMember(ctx, serverID, newOwnerID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotServerMember
	}

	// Transfer ownership
	if err := s.repo.TransferOwnership(ctx, serverID, newOwnerID); err != nil {
		return nil, err
	}

	// Update local server object
	server.OwnerID = newOwnerID
	server.UpdatedAt = time.Now()

	s.eventBus.Publish("server.ownership_transferred", &OwnershipTransferredEvent{
		ServerID:   serverID,
		OldOwnerID: requesterID,
		NewOwnerID: newOwnerID,
	})

	return server, nil
}

// JoinServer joins a server via invite
func (s *ServerService) JoinServer(ctx context.Context, userID uuid.UUID, inviteCode string) (*models.Server, error) {
	invite, err := s.repo.GetInvite(ctx, inviteCode)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, ErrInviteNotFound
	}

	// Check expiration
	if invite.ExpiresAt != nil && invite.ExpiresAt.Before(time.Now()) {
		return nil, ErrInviteExpired
	}

	// Check max uses
	if invite.MaxUses > 0 && invite.Uses >= invite.MaxUses {
		return nil, ErrInviteExpired
	}

	server, err := s.repo.GetByID(ctx, invite.ServerID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}

	// Check if banned
	ban, _ := s.repo.GetBan(ctx, invite.ServerID, userID)
	if ban != nil {
		return nil, ErrBannedFromServer
	}

	// Check if already member
	existing, _ := s.repo.GetMember(ctx, invite.ServerID, userID)
	if existing != nil {
		return nil, ErrAlreadyMember
	}

	// Check quota
	limits, err := s.quotaService.GetEffectiveLimits(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	userServers, err := s.repo.GetUserServers(ctx, userID)
	if err != nil {
		return nil, err
	}

	if limits.MaxServersJoined > 0 && len(userServers) >= limits.MaxServersJoined {
		return nil, ErrMaxServersReached
	}

	// Get @everyone role
	roles, err := s.roleRepo.GetByServerID(ctx, invite.ServerID)
	if err != nil {
		return nil, err
	}
	var everyoneRoleID uuid.UUID
	for _, r := range roles {
		if r.IsDefault {
			everyoneRoleID = r.ID
			break
		}
	}

	// Add member
	member := &models.Member{
		UserID:   userID,
		ServerID: invite.ServerID,
		JoinedAt: time.Now(),
		Roles:    []uuid.UUID{everyoneRoleID},
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	// Increment invite uses
	_ = s.repo.IncrementInviteUses(ctx, inviteCode)

	// Log invite use for analytics
	_ = s.repo.LogInviteUse(ctx, &models.InviteUseLog{
		InviteCode:       inviteCode,
		ServerID:         invite.ServerID,
		UserID:           userID,
		JoinedAt:         time.Now(),
		AccountCreatedAt: time.Now(), // Will be enriched by handler if available
	})

	// Auto-delete one-time invites (max_uses == 1 and now used)
	if invite.MaxUses == 1 {
		_ = s.repo.DeleteInvite(ctx, inviteCode)
	}

	s.eventBus.Publish("server.member_joined", &MemberJoinedEvent{
		ServerID:   invite.ServerID,
		UserID:     userID,
		InviteCode: inviteCode,
	})

	return server, nil
}

// LeaveServer leaves a server
func (s *ServerService) LeaveServer(ctx context.Context, serverID, userID uuid.UUID) error {
	server, err := s.repo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	// Owner can't leave, must transfer or delete
	if server.OwnerID == userID {
		return errors.New("owner cannot leave server, transfer ownership or delete")
	}

	if err := s.repo.RemoveMember(ctx, serverID, userID); err != nil {
		return err
	}

	s.eventBus.Publish("server.member_left", &MemberLeftEvent{
		ServerID: serverID,
		UserID:   userID,
	})

	return nil
}

// KickMember kicks a member from server
func (s *ServerService) KickMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID, reason string) error {
	server, err := s.repo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	// Can't kick owner
	if server.OwnerID == targetID {
		return errors.New("cannot kick server owner")
	}

	// Require KICK_MEMBERS permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermKickMembers); err != nil {
			return err
		}
		// Check role hierarchy - can't kick members with higher/equal roles
		canManage, err := s.permService.CanManageMember(ctx, serverID, requesterID, targetID)
		if err != nil {
			return err
		}
		if !canManage {
			return ErrCannotManageMember
		}
	}

	if err := s.repo.RemoveMember(ctx, serverID, targetID); err != nil {
		return err
	}

	s.eventBus.Publish("server.member_kicked", &MemberKickedEvent{
		ServerID: serverID,
		UserID:   targetID,
		KickedBy: requesterID,
		Reason:   reason,
	})

	return nil
}

// BanMember bans a member from server
func (s *ServerService) BanMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID, reason string, deleteDays int) error {
	server, err := s.repo.GetByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return ErrServerNotFound
	}

	// Can't ban owner
	if server.OwnerID == targetID {
		return errors.New("cannot ban server owner")
	}

	// Require BAN_MEMBERS permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermBanMembers); err != nil {
			return err
		}
		// Check role hierarchy - can't ban members with higher/equal roles
		canManage, err := s.permService.CanManageMember(ctx, serverID, requesterID, targetID)
		if err != nil {
			return err
		}
		if !canManage {
			return ErrCannotManageMember
		}
	}

	// Remove member first
	_ = s.repo.RemoveMember(ctx, serverID, targetID)

	// Add ban
	var banReason *string
	if reason != "" {
		banReason = &reason
	}
	ban := &models.Ban{
		ServerID:  serverID,
		UserID:    targetID,
		Reason:    banReason,
		BannedBy:  &requesterID,
		CreatedAt: time.Now(),
	}

	if err := s.repo.AddBan(ctx, ban); err != nil {
		return err
	}

	// Delete messages from last N days if deleteDays > 0
	if deleteDays > 0 && s.messageRepo != nil {
		since := time.Now().AddDate(0, 0, -deleteDays)
		channels, err := s.channelRepo.GetByServerID(ctx, serverID)
		if err != nil {
			log.Printf("failed to get channels for message deletion during ban: server=%s err=%v", serverID, err)
		}
		for _, ch := range channels {
			// Best effort - don't fail ban if message deletion fails
			if _, err := s.messageRepo.DeleteByAuthor(ctx, ch.ID, targetID, since); err != nil {
				log.Printf("failed to delete messages by author during ban: channel=%s author=%s err=%v", ch.ID, targetID, err)
			}
		}
	}

	s.eventBus.Publish("server.member_banned", &MemberBannedEvent{
		ServerID:    serverID,
		UserID:      targetID,
		ModeratorID: requesterID,
		Reason:      reason,
	})

	return nil
}

// CreateInvite creates a server invite
func (s *ServerService) CreateInvite(ctx context.Context, serverID, channelID, creatorID uuid.UUID, maxUses int, expiresIn *time.Duration) (*models.Invite, error) {
	// Verify member
	member, err := s.repo.GetMember(ctx, serverID, creatorID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Require CREATE_INVITE permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, creatorID, models.PermCreateInvite); err != nil {
			return nil, err
		}
	}

	// Check invite creation rate limit
	if s.inviteRateLimiter != nil {
		if err := s.inviteRateLimiter.CheckInviteCreation(ctx, creatorID); err != nil {
			return nil, err
		}
	}

	// Generate invite code
	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().Add(*expiresIn)
		expiresAt = &t
	}

	invite := &models.Invite{
		Code:      code,
		ServerID:  serverID,
		ChannelID: channelID,
		CreatorID: creatorID,
		MaxUses:   maxUses,
		Uses:      0,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateInvite(ctx, invite); err != nil {
		return nil, err
	}

	return invite, nil
}

// CreateVanityInvite creates or updates a vanity invite for a server
func (s *ServerService) CreateVanityInvite(ctx context.Context, serverID, channelID, creatorID uuid.UUID, vanityCode string) (*models.Invite, error) {
	// Verify member
	member, err := s.repo.GetMember(ctx, serverID, creatorID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Require MANAGE_SERVER permission for vanity URLs
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, creatorID, models.PermManageServer); err != nil {
			return nil, err
		}
	}

	// Validate vanity code
	if err := validateVanityCode(vanityCode); err != nil {
		return nil, err
	}

	// Check if vanity code is already taken by another server
	existing, _ := s.repo.GetInviteByVanityCode(ctx, vanityCode)
	if existing != nil && existing.ServerID != serverID {
		return nil, ErrVanityCodeTaken
	}

	// Delete existing vanity invite for this server if any
	existingInvites, _ := s.repo.GetInvites(ctx, serverID)
	for _, inv := range existingInvites {
		if inv.IsVanity {
			_ = s.repo.DeleteInvite(ctx, inv.Code)
			break
		}
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	invite := &models.Invite{
		Code:       code,
		ServerID:   serverID,
		ChannelID:  channelID,
		CreatorID:  creatorID,
		MaxUses:    0, // Vanity invites don't expire
		Uses:       0,
		IsVanity:   true,
		VanityCode: &vanityCode,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateInvite(ctx, invite); err != nil {
		return nil, err
	}

	return invite, nil
}

// GetInviteByVanityCode retrieves an invite by its vanity code
func (s *ServerService) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	invite, err := s.repo.GetInviteByVanityCode(ctx, vanityCode)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, ErrInviteNotFound
	}
	return invite, nil
}

// GetInviteAnalytics returns analytics for a specific invite
func (s *ServerService) GetInviteAnalytics(ctx context.Context, serverID, requesterID uuid.UUID, inviteCode string) (*models.InviteAnalytics, error) {
	member, err := s.repo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageServer); err != nil {
			return nil, err
		}
	}

	logs, err := s.repo.GetInviteUseLogs(ctx, inviteCode)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)
	recentUses := 0
	newAccountJoins := 0
	for _, l := range logs {
		if l.JoinedAt.After(dayAgo) {
			recentUses++
		}
		if l.AccountAgeDays < 7 {
			newAccountJoins++
		}
	}

	return &models.InviteAnalytics{
		Code:           inviteCode,
		TotalUses:      len(logs),
		RecentUses:     recentUses,
		NewAccountJoin: newAccountJoins,
		UseLogs:        logs,
	}, nil
}

// GetServerInviteAnalytics returns analytics for all invites in a server
func (s *ServerService) GetServerInviteAnalytics(ctx context.Context, serverID, requesterID uuid.UUID) ([]models.InviteAnalytics, error) {
	member, err := s.repo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageServer); err != nil {
			return nil, err
		}
	}

	invites, err := s.repo.GetInvites(ctx, serverID)
	if err != nil {
		return nil, err
	}

	allLogs, err := s.repo.GetServerInviteUseLogs(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Group logs by invite code
	logsByCode := make(map[string][]models.InviteUseLog)
	for _, l := range allLogs {
		logsByCode[l.InviteCode] = append(logsByCode[l.InviteCode], l)
	}

	now := time.Now()
	dayAgo := now.Add(-24 * time.Hour)
	var analytics []models.InviteAnalytics
	for _, inv := range invites {
		logs := logsByCode[inv.Code]
		recentUses := 0
		newAccountJoins := 0
		for _, l := range logs {
			if l.JoinedAt.After(dayAgo) {
				recentUses++
			}
			if l.AccountAgeDays < 7 {
				newAccountJoins++
			}
		}
		analytics = append(analytics, models.InviteAnalytics{
			Code:           inv.Code,
			TotalUses:      len(logs),
			RecentUses:     recentUses,
			NewAccountJoin: newAccountJoins,
		})
	}

	return analytics, nil
}

func validateVanityCode(code string) error {
	if len(code) < 3 || len(code) > 32 {
		return ErrVanityCodeInvalid
	}
	for _, c := range code {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return ErrVanityCodeInvalid
		}
	}
	return nil
}

// Events

type ServerCreatedEvent struct {
	Server  *models.Server
	OwnerID uuid.UUID
}

type ServerUpdatedEvent struct {
	Server *models.Server
}

type ServerDeletedEvent struct {
	ServerID uuid.UUID
	OwnerID  uuid.UUID
}

type MemberLeftEvent struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
}

type MemberKickedEvent struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
	KickedBy uuid.UUID
	Reason   string
}

type OwnershipTransferredEvent struct {
	ServerID   uuid.UUID
	OldOwnerID uuid.UUID
	NewOwnerID uuid.UUID
}

// MemberBannedEvent and MemberJoinedEvent are defined in invite_service.go

// GetUserServers retrieves all servers a user is a member of
func (s *ServerService) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	return s.repo.GetUserServers(ctx, userID)
}

// GetMembers retrieves members of a server with pagination
func (s *ServerService) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	return s.repo.GetMembers(ctx, serverID, limit, offset)
}

// GetMember retrieves a specific member
func (s *ServerService) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	member, err := s.repo.GetMember(ctx, serverID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrNotServerMember
	}
	return member, nil
}

// UpdateMember updates a member's nickname/roles
func (s *ServerService) UpdateMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID, nickname *string, roles []uuid.UUID) (*models.Member, error) {
	member, err := s.repo.GetMember(ctx, serverID, targetID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Check permissions
	if s.permService != nil {
		if requesterID == targetID {
			// User editing self - only need CHANGE_NICKNAME for nickname changes
			if nickname != nil {
				if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermChangeNickname); err != nil {
					return nil, err
				}
			}
			// Users can't change their own roles
			if roles != nil {
				return nil, ErrNoPermission
			}
		} else {
			// Admin editing others - need MANAGE_NICKNAMES for nickname, MANAGE_ROLES for roles
			if nickname != nil {
				if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageNicknames); err != nil {
					return nil, err
				}
			}
			if roles != nil {
				if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageRoles); err != nil {
					return nil, err
				}
			}
		}
	}

	if nickname != nil {
		member.Nickname = nickname
	}
	if roles != nil {
		member.Roles = roles
	}

	if err := s.repo.UpdateMember(ctx, member); err != nil {
		return nil, err
	}

	return member, nil
}

// GetBans retrieves all bans for a server
func (s *ServerService) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	return s.repo.GetBans(ctx, serverID)
}

// UnbanMember removes a ban
func (s *ServerService) UnbanMember(ctx context.Context, serverID, requesterID, targetID uuid.UUID) error {
	// Require BAN_MEMBERS permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermBanMembers); err != nil {
			return err
		}
	}
	return s.repo.RemoveBan(ctx, serverID, targetID)
}

// GetInvites retrieves all invites for a server
func (s *ServerService) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	return s.repo.GetInvites(ctx, serverID)
}

// GetInvite retrieves an invite by code
func (s *ServerService) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	invite, err := s.repo.GetInvite(ctx, code)
	if err != nil {
		return nil, err
	}
	if invite == nil {
		return nil, ErrInviteNotFound
	}
	return invite, nil
}

// DeleteInvite deletes an invite
func (s *ServerService) DeleteInvite(ctx context.Context, code string, requesterID uuid.UUID) error {
	invite, err := s.repo.GetInvite(ctx, code)
	if err != nil || invite == nil {
		return ErrInviteNotFound
	}

	// Creator can always delete their own invite
	if invite.CreatorID != requesterID {
		// Non-creator needs MANAGE_SERVER permission
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, invite.ServerID, requesterID, models.PermManageServer); err != nil {
				return err
			}
		}
	}

	return s.repo.DeleteInvite(ctx, code)
}

// GetMutualServersLimited returns mutual servers between two users with a limit
func (s *ServerService) GetMutualServersLimited(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.Server, int, error) {
	// Check if repo has the limited method
	if repo, ok := s.repo.(interface {
		GetMutualServersLimited(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.Server, int, error)
	}); ok {
		return repo.GetMutualServersLimited(ctx, userID1, userID2, limit)
	}

	// Fallback to getting all and limiting in memory
	if repo, ok := s.repo.(interface {
		GetMutualServers(ctx context.Context, userID1, userID2 uuid.UUID) ([]*models.Server, error)
	}); ok {
		servers, err := repo.GetMutualServers(ctx, userID1, userID2)
		if err != nil {
			return nil, 0, err
		}
		total := len(servers)
		if limit > 0 && len(servers) > limit {
			servers = servers[:limit]
		}
		return servers, total, nil
	}

	return []*models.Server{}, 0, nil
}

// GetChannels retrieves all channels for a server
func (s *ServerService) GetChannels(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	return s.channelRepo.GetByServerID(ctx, serverID)
}

// GetRoles retrieves all roles for a server
func (s *ServerService) GetRoles(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	return s.roleRepo.GetByServerID(ctx, serverID)
}

var (
	ErrBanNotFound      = errors.New("ban not found")
	ErrBanAlreadyExist  = errors.New("user is already banned in this guild")
	ErrBanAlreadyActive = errors.New("ban is already active")
)

// BanManagementRepository defines the contract for ban persistence operations.
// This is separate from the BanRepository in invite_service.go which has different needs.
type BanManagementRepository interface {
	FindByUserAndGuild(ctx context.Context, guildID, userID uuid.UUID) (*models.Ban, error)
	Create(ctx context.Context, ban *models.Ban) error
	GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error)
	Delete(ctx context.Context, serverID, userID uuid.UUID) error
}

// BanService handles business logic related to banning users.
type BanService struct {
	repo BanManagementRepository
}

// NewBanService creates a new BanService instance.
func NewBanService(repo BanManagementRepository) *BanService {
	return &BanService{
		repo: repo,
	}
}

// CreateBan creates a new ban record.
// It checks if the user is already banned before inserting.
func (s *BanService) CreateBan(ctx context.Context, guildID, userID uuid.UUID, reason string, bannedBy uuid.UUID) (*models.Ban, error) {
	// Check if user is currently banned
	existingBan, err := s.repo.FindByUserAndGuild(ctx, guildID, userID)

	// Handle database specific errors (should be wrapped externally, but checked here)
	if err != nil && !errors.Is(err, ErrBanNotFound) {
		return nil, fmt.Errorf("failed to check existing bans: %w", err)
	}

	if existingBan != nil {
		// User is already banned
		return nil, ErrBanAlreadyExist
	}

	newBan := &models.Ban{
		ServerID: guildID,
		UserID:   userID,
		Reason:   &reason,
		BannedBy: &bannedBy,
	}

	if err := s.repo.Create(ctx, newBan); err != nil {
		return nil, fmt.Errorf("failed to create ban: %w", err)
	}

	return newBan, nil
}

// Unban removes a ban by server and user ID.
func (s *BanService) Unban(ctx context.Context, serverID, userID uuid.UUID) error {
	// Verify it exists
	ban, err := s.repo.GetByServerAndUser(ctx, serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve ban: %w", err)
	}

	// Avoid unused variable error
	_ = ban

	if err := s.repo.Delete(ctx, serverID, userID); err != nil {
		return fmt.Errorf("failed to delete ban: %w", err)
	}

	return nil
}

// GetBan retrieves a ban by server and user ID.
func (s *BanService) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	return s.repo.GetByServerAndUser(ctx, serverID, userID)
}

// CheckIfBanned checks if a user is currently banned in a guild.
// It returns true if banned. It is the caller's responsibility to check context cancellation.
func (s *BanService) CheckIfBanned(ctx context.Context, guildID, userID uuid.UUID) (bool, error) {
	ban, err := s.repo.FindByUserAndGuild(ctx, guildID, userID)
	if err != nil {
		// If it's an "Not Found" error, the user is not banned.
		if errors.Is(err, ErrBanNotFound) {
			return false, nil
		}
		return false, err
	}

	// Avoid unused variable error
	_ = ban
	return true, nil
}

// RoleService handles role-related business logic
type RoleService struct {
	roleRepo    RoleRepository
	serverRepo  ServerRepository
	cache       CacheService
	eventBus    EventBus
	permService *PermissionService
}

// NewRoleService creates a new role service
func NewRoleService(
	roleRepo RoleRepository,
	serverRepo ServerRepository,
	cache CacheService,
	eventBus EventBus,
	permService *PermissionService,
) *RoleService {
	return &RoleService{
		roleRepo:    roleRepo,
		serverRepo:  serverRepo,
		cache:       cache,
		eventBus:    eventBus,
		permService: permService,
	}
}

// checkManageRolesPermission verifies the requester has MANAGE_ROLES permission
func (s *RoleService) checkManageRolesPermission(ctx context.Context, serverID, requesterID uuid.UUID) error {
	// Use permService if available for consistent permission checking
	if s.permService != nil {
		return s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageRoles)
	}
	// Fallback to local permission check
	perms, err := s.ComputeMemberPermissions(ctx, serverID, requesterID)
	if err != nil {
		return err
	}
	if perms&models.PermManageRoles == 0 && perms&models.PermAdministrator == 0 {
		return ErrMissingManageRoles
	}
	return nil
}

// getHighestRolePosition returns the highest (lowest number = highest priority) role position for a member
// Returns -1 for server owner (highest possible)
func (s *RoleService) getHighestRolePosition(ctx context.Context, serverID, userID uuid.UUID) (int, error) {
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return 0, err
	}
	if server != nil && server.OwnerID == userID {
		return -1, nil // Owner is above all roles
	}

	roles, err := s.roleRepo.GetMemberRoles(ctx, serverID, userID)
	if err != nil {
		return 0, err
	}
	if len(roles) == 0 {
		return 999999, nil // No roles = lowest position
	}

	highest := roles[0].Position
	for _, role := range roles[1:] {
		if role.Position < highest {
			highest = role.Position
		}
	}
	return highest, nil
}

// checkRoleHierarchy ensures requester's highest role is above the target role
func (s *RoleService) checkRoleHierarchy(ctx context.Context, serverID, requesterID uuid.UUID, targetRole *models.Role) error {
	// Use permService if available
	if s.permService != nil {
		canManage, err := s.permService.CanManageRole(ctx, serverID, requesterID, targetRole)
		if err != nil {
			return err
		}
		if !canManage {
			return ErrRoleHierarchy
		}
		return nil
	}
	// Fallback to local hierarchy check
	requesterPos, err := s.getHighestRolePosition(ctx, serverID, requesterID)
	if err != nil {
		return err
	}
	// Lower position number = higher in hierarchy
	if requesterPos >= targetRole.Position {
		return ErrRoleHierarchy
	}
	return nil
}

// CreateRole creates a new role in a server
func (s *RoleService) CreateRole(
	ctx context.Context,
	serverID uuid.UUID,
	creatorID uuid.UUID,
	name string,
	color int,
	permissions int64,
) (*models.Role, error) {
	// Verify creator is a member with permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, creatorID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, creatorID); err != nil {
		return nil, err
	}

	// Get existing roles to determine position
	roles, err := s.roleRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	role := &models.Role{
		ID:          uuid.New(),
		ServerID:    serverID,
		Name:        name,
		Color:       color,
		Position:    len(roles), // Add at end
		Permissions: permissions,
		CreatedAt:   time.Now(),
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	s.eventBus.Publish("role.created", &RoleCreatedEvent{
		Role:     role,
		ServerID: serverID,
	})

	return role, nil
}

// UpdateRole updates a role
func (s *RoleService) UpdateRole(
	ctx context.Context,
	roleID uuid.UUID,
	requesterID uuid.UUID,
	updates *models.RoleUpdate,
) (*models.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, role.ServerID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, role.ServerID, requesterID); err != nil {
		return nil, err
	}
	if err := s.checkRoleHierarchy(ctx, role.ServerID, requesterID, role); err != nil {
		return nil, err
	}

	// Apply updates
	if updates.Name != nil {
		role.Name = *updates.Name
	}
	if updates.Color != nil {
		role.Color = *updates.Color
	}
	if updates.Permissions != nil {
		role.Permissions = *updates.Permissions
	}
	if updates.Hoist != nil {
		role.Hoist = *updates.Hoist
	}
	if updates.Mentionable != nil {
		role.Mentionable = *updates.Mentionable
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}

	s.eventBus.Publish("role.updated", &RoleUpdatedEvent{
		Role: role,
	})

	return role, nil
}

// DeleteRole deletes a role
func (s *RoleService) DeleteRole(ctx context.Context, roleID, requesterID uuid.UUID) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	// Can't delete @everyone role
	if role.IsDefault {
		return ErrCannotDeleteDefault
	}

	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, role.ServerID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, role.ServerID, requesterID); err != nil {
		return err
	}
	if err := s.checkRoleHierarchy(ctx, role.ServerID, requesterID, role); err != nil {
		return err
	}

	if err := s.roleRepo.Delete(ctx, roleID); err != nil {
		return err
	}

	s.eventBus.Publish("role.deleted", &RoleDeletedEvent{
		RoleID:   roleID,
		ServerID: role.ServerID,
	})

	return nil
}

// GetServerRoles gets all roles in a server
func (s *RoleService) GetServerRoles(ctx context.Context, serverID, requesterID uuid.UUID) ([]*models.Role, error) {
	// Verify requester is a member
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	return s.roleRepo.GetByServerID(ctx, serverID)
}

// GetRole retrieves a single role by ID
func (s *RoleService) GetRole(ctx context.Context, roleID, requesterID uuid.UUID) (*models.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, ErrRoleNotFound
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	// Verify requester is a member of the server this role belongs to
	member, err := s.serverRepo.GetMember(ctx, role.ServerID, requesterID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	return role, nil
}

// UpdateRolePositions updates the positions of multiple roles
func (s *RoleService) UpdateRolePositions(
	ctx context.Context,
	serverID uuid.UUID,
	requesterID uuid.UUID,
	positions map[uuid.UUID]int,
) error {
	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, requesterID); err != nil {
		return err
	}

	return s.roleRepo.UpdatePositions(ctx, serverID, positions)
}

// AddRoleToMember assigns a role to a member
func (s *RoleService) AddRoleToMember(
	ctx context.Context,
	serverID, userID, roleID uuid.UUID,
	requesterID uuid.UUID,
) error {
	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, requesterID); err != nil {
		return err
	}

	// Verify target is a member
	target, err := s.serverRepo.GetMember(ctx, serverID, userID)
	if err != nil || target == nil {
		return ErrNotServerMember
	}

	// Verify role exists
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil || role == nil {
		return ErrRoleNotFound
	}
	if role.ServerID != serverID {
		return ErrRoleNotFound
	}

	// Verify requester can manage this role (hierarchy check)
	if err := s.checkRoleHierarchy(ctx, serverID, requesterID, role); err != nil {
		return err
	}

	if err := s.roleRepo.AddRoleToMember(ctx, serverID, userID, roleID); err != nil {
		return err
	}

	s.eventBus.Publish("member.role_added", &MemberRoleAddedEvent{
		ServerID: serverID,
		UserID:   userID,
		RoleID:   roleID,
	})

	return nil
}

// RemoveRoleFromMember removes a role from a member
func (s *RoleService) RemoveRoleFromMember(
	ctx context.Context,
	serverID, userID, roleID uuid.UUID,
	requesterID uuid.UUID,
) error {
	// Check permissions
	member, err := s.serverRepo.GetMember(ctx, serverID, requesterID)
	if err != nil || member == nil {
		return ErrNotServerMember
	}
	if err := s.checkManageRolesPermission(ctx, serverID, requesterID); err != nil {
		return err
	}

	// Fetch role to check hierarchy
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil || role.ServerID != serverID {
		return ErrRoleNotFound
	}
	if err := s.checkRoleHierarchy(ctx, serverID, requesterID, role); err != nil {
		return err
	}

	if err := s.roleRepo.RemoveRoleFromMember(ctx, serverID, userID, roleID); err != nil {
		return err
	}

	s.eventBus.Publish("member.role_removed", &MemberRoleRemovedEvent{
		ServerID: serverID,
		UserID:   userID,
		RoleID:   roleID,
	})

	return nil
}

// GetMemberRoles gets all roles for a member
func (s *RoleService) GetMemberRoles(ctx context.Context, serverID, userID uuid.UUID) ([]*models.Role, error) {
	return s.roleRepo.GetMemberRoles(ctx, serverID, userID)
}

// ComputeMemberPermissions computes effective permissions for a member
func (s *RoleService) ComputeMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error) {
	// Get server to check ownership
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return 0, err
	}
	if server == nil {
		return 0, ErrServerNotFound
	}

	// Owner has all permissions
	if server.OwnerID == userID {
		return models.PermissionAll, nil
	}

	// Get member's roles
	roles, err := s.roleRepo.GetMemberRoles(ctx, serverID, userID)
	if err != nil {
		return 0, err
	}

	// Combine permissions from all roles
	var permissions int64 = 0
	for _, role := range roles {
		permissions |= role.Permissions
	}

	// Administrator grants all permissions
	if permissions&models.PermAdministrator != 0 {
		return models.PermissionAll, nil
	}

	return permissions, nil
}

// Events

type RoleCreatedEvent struct {
	Role     *models.Role
	ServerID uuid.UUID
}

type RoleUpdatedEvent struct {
	Role *models.Role
}

type RoleDeletedEvent struct {
	RoleID   uuid.UUID
	ServerID uuid.UUID
}

type MemberRoleAddedEvent struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
	RoleID   uuid.UUID
}

type MemberRoleRemovedEvent struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
	RoleID   uuid.UUID
}

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
	inviteRepo    InviteRepository
	banRepo       BanRepository
	serverRepo    ServerRepository
	permService   *PermissionService
	cache         CacheService
	eventBus      EventBus
	rateLimiter   InviteRateLimiter
}

// NewInviteService creates a new invite service
func NewInviteService(
	inviteRepo InviteRepository,
	banRepo BanRepository,
	serverRepo ServerRepository,
	permService *PermissionService,
	cache CacheService,
	eventBus EventBus,
) *InviteService {
	return &InviteService{
		inviteRepo:  inviteRepo,
		banRepo:     banRepo,
		serverRepo:  serverRepo,
		permService: permService,
		cache:       cache,
		eventBus:    eventBus,
	}
}

// SetInviteRateLimiter sets the rate limiter for invite creation
func (s *InviteService) SetInviteRateLimiter(rateLimiter InviteRateLimiter) {
	s.rateLimiter = rateLimiter
}

// CreateInviteRequest represents an invite creation request
type CreateInviteRequest struct {
	ServerID  uuid.UUID
	ChannelID uuid.UUID
	CreatorID uuid.UUID
	MaxUses   int           // 0 = unlimited
	MaxAge    time.Duration // 0 = never expires
	Temporary bool
}

// CreateInvite creates a new server invite
func (s *InviteService) CreateInvite(ctx context.Context, req *CreateInviteRequest) (*models.Invite, error) {
	// Verify creator is a member
	member, err := s.serverRepo.GetMember(ctx, req.ServerID, req.CreatorID)
	if err != nil || member == nil {
		return nil, ErrNotServerMember
	}

	// Check CREATE_INVITE permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, req.ServerID, req.CreatorID, models.PermCreateInvite); err != nil {
			return nil, err
		}
	}

	// Check invite creation rate limit
	if s.rateLimiter != nil {
		if err := s.rateLimiter.CheckInviteCreation(ctx, req.CreatorID); err != nil {
			return nil, err
		}
	}

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
		ServerID: invite.ServerID,
		UserID:   userID,
		JoinedAt: time.Now(),
		Pending:  invite.Temporary, // Temporary invites create pending members
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
		ServerID:   invite.ServerID,
		UserID:     userID,
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
		// Require MANAGE_SERVER permission to delete others' invites
		if s.permService != nil {
			if err := s.permService.RequirePermission(ctx, invite.ServerID, requesterID, models.PermManageServer); err != nil {
				return err
			}
		}
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

	// Require MANAGE_SERVER permission to view all invites
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermManageServer); err != nil {
			return nil, err
		}
	}

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

	// Require BAN_MEMBERS permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, moderatorID, models.PermBanMembers); err != nil {
			return err
		}
		// Check role hierarchy - can't ban members with higher/equal roles
		canManage, err := s.permService.CanManageMember(ctx, serverID, moderatorID, userID)
		if err != nil {
			return err
		}
		if !canManage {
			return ErrCannotManageMember
		}
	}

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
		Reason:    &reason,
		BannedBy:  &moderatorID,
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

	// Require BAN_MEMBERS permission
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, moderatorID, models.PermBanMembers); err != nil {
			return err
		}
	}

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

	// Require BAN_MEMBERS permission to view bans
	if s.permService != nil {
		if err := s.permService.RequirePermission(ctx, serverID, requesterID, models.PermBanMembers); err != nil {
			return nil, err
		}
	}

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

// DefaultInviteRateLimit is the default maximum invites per user per hour
const DefaultInviteRateLimit = 5

// DefaultInviteRateWindow is the default time window for invite rate limiting
const DefaultInviteRateWindow = 1 * time.Hour

// InviteRateLimiterConfig holds configuration for invite rate limiting
type InviteRateLimiterConfig struct {
	// MaxInvites is the maximum number of invites allowed per window
	MaxInvites int
	// Window is the duration of the rate limit window
	Window time.Duration
}

// DefaultInviteRateLimiterConfig returns the default configuration
func DefaultInviteRateLimiterConfig() InviteRateLimiterConfig {
	return InviteRateLimiterConfig{
		MaxInvites: DefaultInviteRateLimit,
		Window:     DefaultInviteRateWindow,
	}
}

// RedisInviteRateLimiter implements InviteRateLimiter using Redis
type RedisInviteRateLimiter struct {
	redisLimiter *ratelimit.RedisLimiter
	config       InviteRateLimiterConfig
}

// NewRedisInviteRateLimiter creates a new Redis-backed invite rate limiter
func NewRedisInviteRateLimiter(redisLimiter *ratelimit.RedisLimiter, config InviteRateLimiterConfig) *RedisInviteRateLimiter {
	if config.MaxInvites == 0 {
		config.MaxInvites = DefaultInviteRateLimit
	}
	if config.Window == 0 {
		config.Window = DefaultInviteRateWindow
	}
	return &RedisInviteRateLimiter{
		redisLimiter: redisLimiter,
		config:       config,
	}
}

// CheckInviteCreation checks if the user can create an invite
func (r *RedisInviteRateLimiter) CheckInviteCreation(ctx context.Context, userID uuid.UUID) error {
	result, err := r.redisLimiter.CheckUser(ctx, userID, "invite", r.config.MaxInvites, r.config.Window)
	if err != nil {
		// On error, allow the request (fail open)
		return nil
	}

	if !result.Allowed {
		return ErrInviteRateLimited
	}

	return nil
}

// MemoryInviteRateLimiter implements InviteRateLimiter using in-memory storage
type MemoryInviteRateLimiter struct {
	config InviteRateLimiterConfig
	// userWindows stores the start time of each user's current window
	userWindows map[uuid.UUID]windowState
}

type windowState struct {
	start     time.Time
	count     int
	windowEnd time.Time
}

// NewMemoryInviteRateLimiter creates a new in-memory invite rate limiter
func NewMemoryInviteRateLimiter(config InviteRateLimiterConfig) *MemoryInviteRateLimiter {
	if config.MaxInvites == 0 {
		config.MaxInvites = DefaultInviteRateLimit
	}
	if config.Window == 0 {
		config.Window = DefaultInviteRateWindow
	}
	return &MemoryInviteRateLimiter{
		config:     config,
		userWindows: make(map[uuid.UUID]windowState),
	}
}

// CheckInviteCreation checks if the user can create an invite
func (r *MemoryInviteRateLimiter) CheckInviteCreation(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()

	state, exists := r.userWindows[userID]
	if !exists || now.After(state.windowEnd) {
		// Start a new window
		r.userWindows[userID] = windowState{
			start:     now,
			count:     1,
			windowEnd: now.Add(r.config.Window),
		}
		return nil
	}

	// Within window, check count
	if state.count >= r.config.MaxInvites {
		return ErrInviteRateLimited
	}

	// Increment count
	state.count++
	r.userWindows[userID] = state
	return nil
}

// CacheInviteRateLimiter implements InviteRateLimiter using the cache service
// This provides persistence across service restarts
type CacheInviteRateLimiter struct {
	cache   CacheService
	config  InviteRateLimiterConfig
}

// NewCacheInviteRateLimiter creates a new cache-backed invite rate limiter
func NewCacheInviteRateLimiter(cache CacheService, config InviteRateLimiterConfig) *CacheInviteRateLimiter {
	if config.MaxInvites == 0 {
		config.MaxInvites = DefaultInviteRateLimit
	}
	if config.Window == 0 {
		config.Window = DefaultInviteRateWindow
	}
	return &CacheInviteRateLimiter{
		cache:  cache,
		config: config,
	}
}

// cacheKey returns the cache key for a user's invite rate limit
func (r *CacheInviteRateLimiter) cacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("invite_ratelimit:%s", userID.String())
}

// CheckInviteCreation checks if the user can create an invite
func (r *CacheInviteRateLimiter) CheckInviteCreation(ctx context.Context, userID uuid.UUID) error {
	key := r.cacheKey(userID)

	// Try to get existing count
	data, err := r.cache.Get(ctx, key)
	if err != nil || len(data) == 0 {
		// No existing entry, allow and set initial count
		err := r.cache.Set(ctx, key, []byte("1"), r.config.Window)
		if err != nil {
			// Cache error, allow the request
			return nil
		}
		return nil
	}

	// Parse existing count - data is a byte slice
	var count int
	_, parseErr := fmt.Sscanf(string(data), "%d", &count)
	if parseErr != nil {
		// Invalid data, reset
		r.cache.Set(ctx, key, []byte("1"), r.config.Window)
		return nil
	}

	if count >= r.config.MaxInvites {
		return ErrInviteRateLimited
	}

	// Increment count - note: this is not atomic, but for rate limiting it's acceptable
	newCount := count + 1
	r.cache.Set(ctx, key, []byte(fmt.Sprintf("%d", newCount)), r.config.Window)
	return nil
}