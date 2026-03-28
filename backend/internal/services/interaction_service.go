package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// InteractionRepository handles interaction token storage
type InteractionRepository interface {
	SaveToken(ctx context.Context, token *InteractionToken) error
	GetToken(ctx context.Context, token string) (*InteractionToken, error)
	MarkTokenUsed(ctx context.Context, token string) error
}

// InteractionToken represents a stored interaction token
type InteractionToken struct {
	Token         string
	InteractionID uuid.UUID
	AppID         uuid.UUID
	UserID        uuid.UUID
	ServerID      *uuid.UUID
	ChannelID     uuid.UUID
	ExpiresAt     time.Time
	Used          bool
	CreatedAt     time.Time
}

// InteractionService handles interaction responses
type InteractionService struct {
	repo            InteractionRepository
	slashCmdService *SlashCommandService
	webhookService  WebhookCommander
	cache           CacheService
}

// NewInteractionService creates a new interaction service
func NewInteractionService(
	repo InteractionRepository,
	slashCmdService *SlashCommandService,
	webhookService WebhookCommander,
	cache CacheService,
) *InteractionService {
	return &InteractionService{
		repo:            repo,
		slashCmdService: slashCmdService,
		webhookService:  webhookService,
		cache:           cache,
	}
}

// HandleInteraction processes an incoming interaction
func (s *InteractionService) HandleInteraction(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
	if interaction.Token == "" {
		interaction.Token = s.generateToken()
	}

	// Store the interaction token
	token := &InteractionToken{
		Token:         interaction.Token,
		InteractionID: interaction.ID,
		AppID:         interaction.AppID,
		UserID:        interaction.UserID,
		ServerID:      interaction.ServerID,
		ChannelID:     interaction.ChannelID,
		ExpiresAt:     time.Now().Add(15 * time.Minute),
		Used:          false,
		CreatedAt:     time.Now().UTC(),
	}
	s.repo.SaveToken(ctx, token)

	// Handle based on interaction type
	switch interaction.Type {
	case models.InteractionTypePing:
		return &models.InteractionResponse{
			Type: models.CallbackTypePong,
		}, nil

	case models.InteractionTypeApplicationCommand:
		return s.slashCmdService.ExecuteCommand(ctx, interaction)

	case models.InteractionTypeAutocomplete:
		return s.slashCmdService.GetAutocomplete(ctx, interaction)

	case models.InteractionTypeMessageComponent,
		models.InteractionTypeModalSubmit:
		// These are handled by the component service
		return s.handleComponentInteraction(ctx, interaction)

	default:
		return nil, fmt.Errorf("unknown interaction type: %d", interaction.Type)
	}
}

// CreateResponse creates a follow-up response to an interaction
func (s *InteractionService) CreateResponse(ctx context.Context, token string, response *models.InteractionResponse) error {
	storedToken, err := s.repo.GetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	if storedToken.Used {
		return fmt.Errorf("token already used")
	}
	if time.Now().After(storedToken.ExpiresAt) {
		return fmt.Errorf("token expired")
	}

	// Send response via webhook to the application
	if s.webhookService != nil {
		_, err := s.webhookService.SendCommandWebhook(ctx, storedToken.AppID, map[string]interface{}{
			"type":           "followup",
			"token":          token,
			"response":       response,
			"interaction_id": storedToken.InteractionID.String(),
		})
		if err != nil {
			return err
		}
	}

	return s.repo.MarkTokenUsed(ctx, token)
}

// EditResponse edits a previous response to an interaction
func (s *InteractionService) EditResponse(ctx context.Context, token string, response *models.InteractionResponse) error {
	storedToken, err := s.repo.GetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	if time.Now().After(storedToken.ExpiresAt) {
		return fmt.Errorf("token expired")
	}

	// Send edit via webhook
	if s.webhookService != nil {
		_, err := s.webhookService.SendCommandWebhook(ctx, storedToken.AppID, map[string]interface{}{
			"type":           "edit",
			"token":          token,
			"response":       response,
			"interaction_id": storedToken.InteractionID.String(),
		})
		return err
	}
	return nil
}

// DeleteResponse deletes a follow-up message
func (s *InteractionService) DeleteResponse(ctx context.Context, token, messageID string) error {
	storedToken, err := s.repo.GetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}
	if time.Now().After(storedToken.ExpiresAt) {
		return fmt.Errorf("token expired")
	}

	if s.webhookService != nil {
		_, err := s.webhookService.SendCommandWebhook(ctx, storedToken.AppID, map[string]interface{}{
			"type":           "delete_followup",
			"token":          token,
			"message_id":     messageID,
			"interaction_id": storedToken.InteractionID.String(),
		})
		return err
	}
	return nil
}

func (s *InteractionService) handleComponentInteraction(ctx context.Context, interaction *models.Interaction) (*models.InteractionResponse, error) {
	// Component interactions are forwarded to the webhook
	if s.webhookService != nil {
		return s.webhookService.SendCommandWebhook(ctx, interaction.AppID, map[string]interface{}{
			"type":        "component",
			"interaction": interaction,
		})
	}
	return &models.InteractionResponse{
		Type: models.CallbackTypeDeferredUpdateMessage,
	}, nil
}

func (s *InteractionService) generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
