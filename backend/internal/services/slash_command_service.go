package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// SlashCommandRepository interface for data access
type SlashCommandRepository interface {
	Create(ctx context.Context, cmd *models.SlashCommand) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SlashCommand, error)
	GetByAppID(ctx context.Context, appID uuid.UUID) ([]*models.SlashCommand, error)
	GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.SlashCommand, error)
	GetByName(ctx context.Context, appID uuid.UUID, name string, serverID *uuid.UUID) (*models.SlashCommand, error)
	Update(ctx context.Context, cmd *models.SlashCommand) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateExecutionLog(ctx context.Context, log *models.CommandExecution) error
}

// SlashCommandService handles slash command business logic
type SlashCommandService struct {
	repo           SlashCommandRepository
	webhookService WebhookCommander
	permService    PermissionServiceInterface
	cache          CacheService
}

// NewSlashCommandService creates a new slash command service
func NewSlashCommandService(
	repo SlashCommandRepository,
	webhookService WebhookCommander,
	permService PermissionServiceInterface,
	cache CacheService,
) *SlashCommandService {
	return &SlashCommandService{
		repo:           repo,
		webhookService: webhookService,
		permService:    permService,
		cache:          cache,
	}
}

// RegisterCommand registers a new slash command for an application
func (s *SlashCommandService) RegisterCommand(ctx context.Context, appID uuid.UUID, cmd *models.SlashCommand) error {
	cmd.ID = uuid.New()
	cmd.AppID = appID
	cmd.Version = uuid.New().String()
	cmd.CreatedAt = time.Now().UTC()
	cmd.UpdatedAt = time.Now().UTC()
	cmd.Default = true

	if err := cmd.Validate(); err != nil {
		return err
	}

	// Check for duplicate name
	existing, err := s.repo.GetByName(ctx, appID, cmd.Name, cmd.ServerID)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("command with name '%s' already exists", cmd.Name)
	}

	return s.repo.Create(ctx, cmd)
}

// UpdateCommand updates an existing slash command
func (s *SlashCommandService) UpdateCommand(ctx context.Context, appID, commandID uuid.UUID, cmd *models.SlashCommand) error {
	existing, err := s.repo.GetByID(ctx, commandID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("command not found")
	}
	if existing.AppID != appID {
		return fmt.Errorf("command does not belong to this application")
	}

	cmd.ID = commandID
	cmd.AppID = appID
	cmd.Version = uuid.New().String()
	cmd.UpdatedAt = time.Now().UTC()

	if err := cmd.Validate(); err != nil {
		return err
	}

	return s.repo.Update(ctx, cmd)
}

// GetCommand retrieves a command by ID
func (s *SlashCommandService) GetCommand(ctx context.Context, appID, commandID uuid.UUID) (*models.SlashCommand, error) {
	cmd, err := s.repo.GetByID(ctx, commandID)
	if err != nil {
		return nil, err
	}
	if cmd == nil || cmd.AppID != appID {
		return nil, fmt.Errorf("command not found")
	}
	return cmd, nil
}

// GetCommands retrieves all commands for an application
func (s *SlashCommandService) GetCommands(ctx context.Context, appID uuid.UUID) ([]*models.SlashCommand, error) {
	return s.repo.GetByAppID(ctx, appID)
}

// GetServerCommands retrieves all commands available in a server
func (s *SlashCommandService) GetServerCommands(ctx context.Context, serverID uuid.UUID) ([]*models.SlashCommand, error) {
	return s.repo.GetByServerID(ctx, serverID)
}

// DeleteCommand deletes a slash command
func (s *SlashCommandService) DeleteCommand(ctx context.Context, appID, commandID uuid.UUID) error {
	cmd, err := s.repo.GetByID(ctx, commandID)
	if err != nil {
		return err
	}
	if cmd == nil {
		return fmt.Errorf("command not found")
	}
	if cmd.AppID != appID {
		return fmt.Errorf("command does not belong to this application")
	}
	return s.repo.Delete(ctx, commandID)
}

// ExecuteCommand executes a slash command and returns the interaction response
func (s *SlashCommandService) ExecuteCommand(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
	start := time.Now()
	var cmdData *models.CommandInteractionData
	var ok bool

	switch it := interaction.Data.(type) {
	case *models.CommandInteractionData:
		cmdData = it
		ok = true
	case map[string]interface{}:
		// Parse from map
		data, _ := json.Marshal(it)
		json.Unmarshal(data, &cmdData)
		ok = true
	}
	if !ok || cmdData == nil {
		return nil, fmt.Errorf("invalid interaction data type")
	}

	// Find the command
	var serverID *uuid.UUID
	if interaction.ServerID != nil {
		serverID = interaction.ServerID
	}

	cmd, err := s.repo.GetByName(ctx, interaction.AppID, cmdData.Name, serverID)
	if err != nil {
		return nil, err
	}
	if cmd == nil && serverID != nil {
		// Fallback to global command
		cmd, err = s.repo.GetByName(ctx, interaction.AppID, cmdData.Name, nil)
		if err != nil {
			return nil, err
		}
	}
	if cmd == nil {
		return s.logExecution(cmdData.ID, interaction, start, "error", "command not found"), nil
	}

	// Check permissions
	if !cmd.Default && s.permService != nil && interaction.ServerID != nil {
		allowed, err := s.checkCommandPermissions(ctx, cmd, interaction)
		if err != nil {
			return s.logExecution(cmd.ID, interaction, start, "error", err.Error()), nil
		}
		if !allowed {
			return &models.InteractionResponse{
				Type: models.CallbackTypeChannelMessage,
				Data: &models.InteractionCallbackData{
					Content: strPtr("You do not have permission to use this command."),
					Flags:   1 << 6, // Ephemeral
				},
			}, nil
		}
	}

	// Route to webhook for the application
	if s.webhookService != nil {
		// Build webhook message from interaction
		msg := s.buildWebhookMessage(interaction, cmd)
		resp, err := s.webhookService.SendCommandWebhook(ctx, interaction.AppID, msg)
		if err != nil {
			return s.logExecution(cmd.ID, interaction, start, "error", err.Error()), nil
		}
		s.logExecution(cmd.ID, interaction, start, "success", "")
		return resp, nil
	}

	// No webhook configured - return deferred response
	return &models.InteractionResponse{
		Type: models.CallbackTypeDeferredChannelMessage,
		Data: &models.InteractionCallbackData{},
	}, nil
}

// GetAutocomplete returns autocomplete suggestions for a command option
func (s *SlashCommandService) GetAutocomplete(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
	var cmdData *models.CommandInteractionData
	if cmdDataMap, ok := interaction.Data.(map[string]interface{}); ok {
		data, _ := json.Marshal(cmdDataMap)
		json.Unmarshal(data, &cmdData)
	}

	if cmdData == nil {
		return nil, fmt.Errorf("invalid interaction data")
	}

	cmd, err := s.repo.GetByName(ctx, interaction.AppID, cmdData.Name, interaction.ServerID)
	if err != nil || cmd == nil {
		// Try global
		cmd, err = s.repo.GetByName(ctx, interaction.AppID, cmdData.Name, nil)
		if err != nil || cmd == nil {
			return nil, fmt.Errorf("command not found")
		}
	}

	// Build suggestion choices from command options
	var choices []*models.CommandChoice
	for _, opt := range cmd.Options {
		if opt.Autocomplete {
			// In a real implementation, this would call the app's autocomplete handler
			// For now, return empty - the webhook would handle this
			_ = opt
		}
	}

	return &models.InteractionResponse{
		Type: models.CallbackTypeAutocompleteResult,
		Data: &models.InteractionCallbackData{
			Choices: choices,
		},
	}, nil
}

func (s *SlashCommandService) checkCommandPermissions(ctx context.Context, cmd *models.SlashCommand, interaction *models.Interaction) (bool, error) {
	if interaction.ServerID == nil {
		return true, nil
	}
	if cmd.Permissions == nil {
		return cmd.Default, nil
	}

	userID := interaction.UserID
	if interaction.Member != nil {
		userID = interaction.Member.UserID
	}

	for _, override := range cmd.Permissions.Overrides {
		if override.ID == userID && override.Type == 2 {
			if override.Denial {
				return false, nil
			}
			if override.Allow {
				return true, nil
			}
		}
	}

	return cmd.Default, nil
}

func (s *SlashCommandService) buildWebhookMessage(interaction *models.Interaction, cmd *models.SlashCommand) interface{} {
	// Build a webhook payload that includes command context
	data := map[string]interface{}{
		"interaction_type": interaction.Type,
		"command": map[string]interface{}{
			"id":      cmd.ID.String(),
			"name":    cmd.Name,
			"type":    cmd.Type,
			"options": interaction.Data,
		},
		"user_id":    interaction.UserID.String(),
		"server_id":  "",
		"channel_id": interaction.ChannelID.String(),
		"token":      interaction.Token,
	}
	if interaction.ServerID != nil {
		data["server_id"] = interaction.ServerID.String()
	}
	return data
}

func (s *SlashCommandService) logExecution(cmdID uuid.UUID, interaction *models.Interaction, start time.Time, status, errMsg string) *models.InteractionResponse {
	optionsJSON, _ := json.Marshal(interaction.Data)
	var errPtr *string
	if errMsg != "" {
		errPtr = &errMsg
	}
	execLog := &models.CommandExecution{
		ID:           uuid.New(),
		CommandID:    cmdID,
		AppID:        interaction.AppID,
		ServerID:     interaction.ServerID,
		ChannelID:    interaction.ChannelID,
		UserID:       interaction.UserID,
		Options:      string(optionsJSON),
		ResponseTime: time.Since(start).Milliseconds(),
		Status:       status,
		ErrorMsg:     errPtr,
		CreatedAt:    time.Now().UTC(),
	}
	s.repo.CreateExecutionLog(context.Background(), execLog)
	return nil
}

// WebhookCommander interface for sending command events to webhooks
type WebhookCommander interface {
	SendCommandWebhook(ctx context.Context, appID uuid.UUID, payload interface{}) (*models.InteractionResponse, error)
}

// PermissionServiceInterface for checking permissions
type PermissionServiceInterface interface {
	HasPermission(ctx context.Context, serverID, userID uuid.UUID, permission int64) (bool, error)
	GetMemberPermissions(ctx context.Context, serverID, userID uuid.UUID) (int64, error)
}
