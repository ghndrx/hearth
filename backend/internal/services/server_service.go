package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
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
	GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error)
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
	GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error)
	DeleteInvite(ctx context.Context, code string) error
	IncrementInviteUses(ctx context.Context, code string) error
}

// ServerService handles server-related business logic
type ServerService struct {
	repo         ServerRepository
	channelRepo  ChannelRepository
	roleRepo     RoleRepository
	quotaService *QuotaService
	cache        CacheService
	eventBus     EventBus
}

// NewServerService creates a new server service
func NewServerService(
	repo ServerRepository,
	channelRepo ChannelRepository,
	roleRepo RoleRepository,
	quotaService *QuotaService,
	cache CacheService,
	eventBus EventBus,
) *ServerService {
	return &ServerService{
		repo:         repo,
		channelRepo:  channelRepo,
		roleRepo:     roleRepo,
		quotaService: quotaService,
		cache:        cache,
		eventBus:     eventBus,
	}
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
		// TODO: Check admin permission via roles
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
		ServerID:    serverID,
		OldOwnerID:  requesterID,
		NewOwnerID:  newOwnerID,
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

	// TODO: Check requester has KICK_MEMBERS permission

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

	// TODO: Check requester has BAN_MEMBERS permission

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

	// TODO: Delete messages from last N days if deleteDays > 0

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

	// TODO: Check CREATE_INVITE permission

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

	// TODO: Check permissions (user editing self vs. admin editing others)

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
	// TODO: Check permissions
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

	// TODO: Check permissions (creator or admin)

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
