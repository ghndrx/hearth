package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	ErrTemplateNotFound     = errors.New("template not found")
	ErrTemplateNameRequired = errors.New("template name is required")
	ErrTemplateNameTooLong  = errors.New("template name too long (max 100 chars)")
	ErrInvalidTemplateData  = errors.New("invalid template data")
)

// TemplateRepository defines the interface for template data access
type TemplateRepository interface {
	Create(ctx context.Context, tmpl *models.ServerTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ServerTemplate, error)
	GetByCode(ctx context.Context, code string) (*models.ServerTemplate, error)
	GetByCreator(ctx context.Context, creatorID uuid.UUID, limit int) ([]*models.ServerTemplate, error)
	ListPublic(ctx context.Context, cursor *uuid.UUID, limit int) ([]*models.ServerTemplate, *uuid.UUID, error)
	Update(ctx context.Context, tmpl *models.ServerTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementUsage(ctx context.Context, code string) error
	GenerateUniqueCode(ctx context.Context) (string, error)
}

// ChannelRepository defines the interface for channel data access (subset needed for templates)
type TemplateChannelRepository interface {
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error)
	Create(ctx context.Context, channel *models.Channel) error
}

// RoleRepository defines the interface for role data access (subset needed for templates)
type TemplateRoleRepository interface {
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error)
	Create(ctx context.Context, role *models.Role) error
}

// TemplateService handles server template business logic
type TemplateService struct {
	repo        TemplateRepository
	channelRepo TemplateChannelRepository
	roleRepo    TemplateRoleRepository
	serverRepo  ServerRepository
}

// NewTemplateService creates a new template service
func NewTemplateService(
	repo TemplateRepository,
	channelRepo TemplateChannelRepository,
	roleRepo TemplateRoleRepository,
	serverRepo ServerRepository,
) *TemplateService {
	return &TemplateService{
		repo:        repo,
		channelRepo: channelRepo,
		roleRepo:    roleRepo,
		serverRepo:  serverRepo,
	}
}

// CreateTemplate creates a new template from an existing server
func (s *TemplateService) CreateTemplate(
	ctx context.Context,
	serverID uuid.UUID,
	creatorID uuid.UUID,
	name string,
	description string,
	isPublic bool,
) (*models.ServerTemplate, error) {
	if name == "" {
		return nil, ErrTemplateNameRequired
	}
	if len(name) > 100 {
		return nil, ErrTemplateNameTooLong
	}

	// Verify server exists
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrServerNotFound
	}

	// Serialize server structure
	serializedData, err := s.serializeServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	serializedJSON, err := json.Marshal(serializedData)
	if err != nil {
		return nil, err
	}

	// Generate unique code
	code, err := s.repo.GenerateUniqueCode(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	tmpl := &models.ServerTemplate{
		ID:             uuid.New(),
		Code:           code,
		Name:           name,
		Description:    description,
		SourceServerID: &serverID,
		CreatorID:      creatorID,
		SerializedData: serializedJSON,
		UsageCount:     0,
		IsPublic:       isPublic,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, tmpl); err != nil {
		return nil, err
	}

	return tmpl, nil
}

// serializeServer captures the current structure of a server for templating
func (s *TemplateService) serializeServer(ctx context.Context, serverID uuid.UUID) (*models.TemplateSerializedData, error) {
	// Get server settings first
	server, err := s.serverRepo.GetByID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Get channels
	channels, err := s.channelRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	// Build category map for parent lookups
	categoryMap := make(map[uuid.UUID]string) // parentID -> name
	for _, ch := range channels {
		if ch.Type == models.ChannelTypeCategory && ch.ID != uuid.Nil {
			categoryMap[ch.ID] = ch.Name
		}
	}

	// Convert channels to template format
	templateChannels := make([]models.TemplateChannel, 0, len(channels))
	for _, ch := range channels {
		// Skip DMs and group DMs
		if ch.Type == models.ChannelTypeDM || ch.Type == models.ChannelTypeGroupDM {
			continue
		}

		tc := models.TemplateChannel{
			Name:      ch.Name,
			Type:      ch.Type,
			Topic:     ch.Topic,
			Position:  ch.Position,
			NSFW:      ch.NSFW,
			Slowmode:  ch.Slowmode,
			Bitrate:   ch.Bitrate,
			UserLimit: ch.UserLimit,
		}

		// If channel has a parent, store the parent's name
		if ch.ParentID != nil {
			if parentName, ok := categoryMap[*ch.ParentID]; ok {
				tc.ParentName = parentName
			}
		}

		templateChannels = append(templateChannels, tc)
	}

	// Get roles (exclude @everyone - it's created automatically)
	roles, err := s.roleRepo.GetByServerID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	templateRoles := make([]models.TemplateRole, 0, len(roles))
	for _, role := range roles {
		// Skip default @everyone role
		if role.IsDefault {
			continue
		}

		templateRoles = append(templateRoles, models.TemplateRole{
			Name:        role.Name,
			Color:       role.Color,
			Permissions: role.Permissions,
			Position:    role.Position,
			Hoist:       role.Hoist,
			Mentionable: role.Mentionable,
		})
	}

	return &models.TemplateSerializedData{
		Channels: templateChannels,
		Roles:    templateRoles,
		Settings: models.TemplateSettings{
			VerificationLevel:     server.VerificationLevel,
			ExplicitContentFilter: server.ExplicitContentFilter,
			DefaultNotifications:  server.DefaultNotifications,
			AFKTimeout:            server.AFKTimeout,
		},
	}, nil
}

// GetTemplate retrieves a template by its code
func (s *TemplateService) GetTemplate(ctx context.Context, code string) (*models.ServerTemplate, error) {
	tmpl, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, ErrTemplateNotFound
	}
	return tmpl, nil
}

// GetTemplateByID retrieves a template by its ID
func (s *TemplateService) GetTemplateByID(ctx context.Context, id uuid.UUID) (*models.ServerTemplate, error) {
	tmpl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, ErrTemplateNotFound
	}
	return tmpl, nil
}

// ListMyTemplates lists templates created by a user
func (s *TemplateService) ListMyTemplates(ctx context.Context, creatorID uuid.UUID) ([]*models.ServerTemplate, error) {
	return s.repo.GetByCreator(ctx, creatorID, 100)
}

// ListPublicTemplates lists public templates with cursor pagination
func (s *TemplateService) ListPublicTemplates(ctx context.Context, cursor *uuid.UUID, limit int) ([]*models.ServerTemplate, *uuid.UUID, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.ListPublic(ctx, cursor, limit)
}

// UpdateTemplate updates a template's metadata
func (s *TemplateService) UpdateTemplate(
	ctx context.Context,
	templateID uuid.UUID,
	name *string,
	description *string,
	isPublic *bool,
) (*models.ServerTemplate, error) {
	tmpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, ErrTemplateNotFound
	}

	if name != nil {
		if len(*name) > 100 {
			return nil, ErrTemplateNameTooLong
		}
		tmpl.Name = *name
	}
	if description != nil {
		tmpl.Description = *description
	}
	if isPublic != nil {
		tmpl.IsPublic = *isPublic
	}
	tmpl.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, tmpl); err != nil {
		return nil, err
	}

	return tmpl, nil
}

// DeleteTemplate deletes a template
func (s *TemplateService) DeleteTemplate(ctx context.Context, templateID uuid.UUID) error {
	tmpl, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return err
	}
	if tmpl == nil {
		return ErrTemplateNotFound
	}

	return s.repo.Delete(ctx, templateID)
}

// UseTemplate creates a new server from a template
func (s *TemplateService) UseTemplate(
	ctx context.Context,
	code string,
	creatorID uuid.UUID,
	serverName string,
) (*models.Server, error) {
	tmpl, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, ErrTemplateNotFound
	}

	// Parse serialized data
	var data models.TemplateSerializedData
	if err := json.Unmarshal(tmpl.SerializedData, &data); err != nil {
		return nil, ErrInvalidTemplateData
	}

	// Create the server
	server := &models.Server{
		ID:        uuid.New(),
		Name:      serverName,
		OwnerID:   creatorID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.serverRepo.Create(ctx, server); err != nil {
		return nil, err
	}

	// Create @everyone role
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
		// Rollback server
		_ = s.serverRepo.Delete(ctx, server.ID)
		return nil, err
	}

	// Create roles from template
	roleMap := make(map[string]*models.Role) // name -> role (for permission sync)
	for _, tr := range data.Roles {
		role := &models.Role{
			ID:          uuid.New(),
			ServerID:    server.ID,
			Name:        tr.Name,
			Color:       tr.Color,
			Permissions: tr.Permissions,
			Position:    tr.Position + 1, // +1 because @everyone is 0
			Hoist:       tr.Hoist,
			Mentionable: tr.Mentionable,
			IsDefault:   false,
			CreatedAt:   time.Now(),
		}
		if err := s.roleRepo.Create(ctx, role); err != nil {
			// Continue on error - not critical
			continue
		}
		roleMap[tr.Name] = role
	}

	// Create channels from template
	// First pass: create all categories
	categoryMap := make(map[string]*models.Channel) // name -> channel (for parent lookups)
	for _, tc := range data.Channels {
		if tc.Type != models.ChannelTypeCategory {
			continue
		}

		channel := &models.Channel{
			ID:        uuid.New(),
			ServerID:  &server.ID,
			Name:      tc.Name,
			Type:      tc.Type,
			Topic:     tc.Topic,
			Position:  tc.Position,
			NSFW:      tc.NSFW,
			Slowmode:  tc.Slowmode,
			CreatedAt: time.Now(),
		}
		if err := s.channelRepo.Create(ctx, channel); err != nil {
			continue
		}
		categoryMap[tc.Name] = channel
	}

	// Second pass: create non-category channels with parent references
	for _, tc := range data.Channels {
		if tc.Type == models.ChannelTypeCategory {
			continue
		}

		channel := &models.Channel{
			ID:        uuid.New(),
			ServerID:  &server.ID,
			Name:      tc.Name,
			Type:      tc.Type,
			Topic:     tc.Topic,
			Position:  tc.Position,
			NSFW:      tc.NSFW,
			Slowmode:  tc.Slowmode,
			Bitrate:   tc.Bitrate,
			UserLimit: tc.UserLimit,
			CreatedAt: time.Now(),
		}

		// Set parent if this channel belongs to a category
		if tc.ParentName != "" {
			if parent, ok := categoryMap[tc.ParentName]; ok {
				channel.ParentID = &parent.ID
			}
		}

		if err := s.channelRepo.Create(ctx, channel); err != nil {
			// Continue on error - not critical
			continue
		}
	}

	// Add creator as member
	member := &models.Member{
		UserID:   creatorID,
		ServerID: server.ID,
		Nickname: nil,
		JoinedAt: time.Now(),
		Roles:    []uuid.UUID{everyoneRole.ID},
	}
	if err := s.serverRepo.AddMember(ctx, member); err != nil {
		// Continue on error
	}

	// Increment usage count
	_ = s.repo.IncrementUsage(ctx, code)

	return server, nil
}
