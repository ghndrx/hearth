package services

import (
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// ComponentRepository defines component data access
type ComponentRepository interface {
	CreateComponent(ctx context.Context, c *models.MessageComponent) error
	GetComponentByID(ctx context.Context, id uuid.UUID) (*models.MessageComponent, error)
	GetComponentsByMessageID(ctx context.Context, messageID uuid.UUID) ([]*models.MessageComponent, error)
	UpdateComponent(ctx context.Context, c *models.MessageComponent) error
	DeleteComponent(ctx context.Context, id uuid.UUID) error
	DeleteComponentsByMessageID(ctx context.Context, messageID uuid.UUID) error
	CreateInteraction(ctx context.Context, i *models.ComponentInteraction) error
	GetInteractionsByComponentID(ctx context.Context, componentID uuid.UUID) ([]*models.ComponentInteraction, error)
	GetComponentsByCustomID(ctx context.Context, customID string) ([]*models.MessageComponent, error)
}

// ComponentService handles component-related business logic
type ComponentService struct {
	repo        ComponentRepository
	messageRepo MessageRepository
	channelRepo ChannelRepository
	serverRepo  ServerRepository
	eventBus    EventBus
}

// NewComponentService creates a new component service
func NewComponentService(
	repo ComponentRepository,
	messageRepo MessageRepository,
	channelRepo ChannelRepository,
	serverRepo ServerRepository,
	eventBus EventBus,
) *ComponentService {
	return &ComponentService{
		repo:        repo,
		messageRepo: messageRepo,
		channelRepo: channelRepo,
		serverRepo:  serverRepo,
		eventBus:    eventBus,
	}
}

// CreateComponent creates a new component for a message
func (s *ComponentService) CreateComponent(ctx context.Context, msgID uuid.UUID, comp *models.MessageComponent) (*models.MessageComponent, error) {
	// Validate the component
	if err := s.validateComponent(comp); err != nil {
		return nil, err
	}

	// Verify message exists
	message, err := s.messageRepo.GetByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}

	comp.ID = uuid.New()
	comp.MessageID = msgID
	comp.CreatedAt = time.Now()

	if err := s.repo.CreateComponent(ctx, comp); err != nil {
		return nil, err
	}

	return comp, nil
}

// GetMessageComponents retrieves all components for a message
func (s *ComponentService) GetMessageComponents(ctx context.Context, msgID uuid.UUID) ([]*models.MessageComponent, error) {
	return s.repo.GetComponentsByMessageID(ctx, msgID)
}

// UpdateComponent updates a component
func (s *ComponentService) UpdateComponent(ctx context.Context, id uuid.UUID, updates *models.MessageComponent) (*models.MessageComponent, error) {
	existing, err := s.repo.GetComponentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrMessageNotFound
	}

	// Apply updates
	if updates.Style != "" {
		existing.Style = updates.Style
	}
	if updates.Label != "" {
		existing.Label = updates.Label
	}
	if updates.CustomID != "" {
		existing.CustomID = updates.CustomID
	}
	if updates.URL != "" {
		existing.URL = updates.URL
	}
	existing.Disabled = updates.Disabled
	if len(updates.Options) > 0 {
		existing.Options = updates.Options
	}
	if updates.MinValues != nil {
		existing.MinValues = updates.MinValues
	}
	if updates.MaxValues != nil {
		existing.MaxValues = updates.MaxValues
	}
	if updates.Placeholder != "" {
		existing.Placeholder = updates.Placeholder
	}
	existing.Required = updates.Required
	if updates.Value != "" {
		existing.Value = updates.Value
	}
	if updates.MinLength != nil {
		existing.MinLength = updates.MinLength
	}
	if updates.MaxLength != nil {
		existing.MaxLength = updates.MaxLength
	}

	// Validate the updated component
	if err := s.validateComponent(existing); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateComponent(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// DeleteComponent deletes a component
func (s *ComponentService) DeleteComponent(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetComponentByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrMessageNotFound
	}

	return s.repo.DeleteComponent(ctx, id)
}

// HandleInteraction handles a user's interaction with a component
func (s *ComponentService) HandleInteraction(
	ctx context.Context,
	userID, channelID, msgID, componentID uuid.UUID,
	customID string,
	values []string,
) (*models.ComponentInteraction, error) {
	// Verify message exists
	message, err := s.messageRepo.GetByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}

	// Verify channel exists
	channel, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	// Get component by ID
	component, err := s.repo.GetComponentByID(ctx, componentID)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, ErrMessageNotFound
	}

	// Verify custom_id matches
	if component.CustomID != customID {
		return nil, ErrMessageNotFound
	}

	// Check if component is disabled
	if component.Disabled {
		return nil, ErrNoPermission
	}

	// Validate interaction based on component type
	if err := s.validateInteraction(component, values); err != nil {
		return nil, err
	}

	// Create interaction record
	interaction := &models.ComponentInteraction{
		ID:          uuid.New(),
		Type:        models.InteractionTypeComponent,
		UserID:      userID,
		ChannelID:   channelID,
		MessageID:   msgID,
		ComponentID: componentID,
		CustomID:    customID,
		Values:      values,
		CreatedAt:   time.Now(),
	}

	if component.Type == models.ComponentTypeTextInput {
		interaction.Type = models.InteractionTypeTextInput
	}

	if err := s.repo.CreateInteraction(ctx, interaction); err != nil {
		return nil, err
	}

	// Emit WebSocket event for component interaction
	s.eventBus.Publish("component.interaction", &ComponentInteractionEvent{
		Interaction: interaction,
		Component:   component,
		Message:     message,
		ChannelID:   channelID,
	})

	return interaction, nil
}

// UpdateMessageComponents replaces all components on a message
func (s *ComponentService) UpdateMessageComponents(ctx context.Context, msgID uuid.UUID, components []*models.MessageComponent) ([]*models.MessageComponent, error) {
	// Verify message exists
	message, err := s.messageRepo.GetByID(ctx, msgID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}

	// Delete existing components
	if err := s.repo.DeleteComponentsByMessageID(ctx, msgID); err != nil {
		return nil, err
	}

	// Create new components
	var created []*models.MessageComponent
	for _, comp := range components {
		comp.ID = uuid.New()
		comp.MessageID = msgID
		comp.CreatedAt = time.Now()

		if err := s.validateComponent(comp); err != nil {
			return nil, err
		}

		if err := s.repo.CreateComponent(ctx, comp); err != nil {
			return nil, err
		}

		created = append(created, comp)
	}

	// Emit message update event
	s.eventBus.Publish("message.updated", &MessageUpdatedEvent{
		Message:   message,
		ChannelID: message.ChannelID,
	})

	return created, nil
}

// RemoveAllComponents removes all components from a message
func (s *ComponentService) RemoveAllComponents(ctx context.Context, msgID uuid.UUID) error {
	// Verify message exists
	message, err := s.messageRepo.GetByID(ctx, msgID)
	if err != nil {
		return err
	}
	if message == nil {
		return ErrMessageNotFound
	}

	return s.repo.DeleteComponentsByMessageID(ctx, msgID)
}

// validateComponent validates a component based on its type
func (s *ComponentService) validateComponent(comp *models.MessageComponent) error {
	switch comp.Type {
	case models.ComponentTypeActionRow:
		// Action rows can contain other components but don't have style/label themselves
		return nil

	case models.ComponentTypeButton:
		if comp.CustomID == "" {
			return ErrInvalidCredentials // Reusing error for "custom_id required"
		}
		if len(comp.CustomID) > 100 {
			return ErrInvalidCredentials // custom_id too long
		}
		if comp.Style == "" {
			comp.Style = models.ButtonStylePrimary
		}
		if comp.Style == models.ButtonStyleLink {
			if comp.URL == "" {
				return ErrInvalidCredentials // URL required for link buttons
			}
			if err := validateURL(comp.URL); err != nil {
				return err
			}
		}
		if comp.Label == "" && comp.EmojiName == "" {
			return ErrInvalidCredentials // button needs label or emoji
		}

	case models.ComponentTypeSelectMenu:
		if comp.CustomID == "" {
			return ErrInvalidCredentials
		}
		if len(comp.Options) == 0 {
			return ErrInvalidCredentials // select menu needs options
		}
		if len(comp.Options) > 25 {
			return ErrInvalidCredentials // max 25 options
		}
		for _, opt := range comp.Options {
			if opt.Value == "" {
				return ErrInvalidCredentials // option needs value
			}
		}

	case models.ComponentTypeTextInput:
		if comp.CustomID == "" {
			return ErrInvalidCredentials
		}
		if comp.Style == "" {
			comp.Style = models.TextInputStyleShort
		}
		if comp.MaxLength != nil && comp.MinLength != nil {
			if *comp.MinLength > *comp.MaxLength {
				return ErrInvalidCredentials
			}
		}
	}

	return nil
}

// validateInteraction validates a component interaction
func (s *ComponentService) validateInteraction(comp *models.MessageComponent, values []string) error {
	switch comp.Type {
	case models.ComponentTypeSelectMenu:
		minVals := 0
		maxVals := 1
		if comp.MinValues != nil {
			minVals = *comp.MinValues
		}
		if comp.MaxValues != nil {
			maxVals = *comp.MaxValues
		}
		if len(values) < minVals {
			return ErrInvalidCredentials
		}
		if len(values) > maxVals {
			return ErrInvalidCredentials
		}

	case models.ComponentTypeTextInput:
		if comp.Required && len(values) > 0 && values[0] == "" {
			return ErrInvalidCredentials
		}
		if comp.MinLength != nil && len(values) > 0 && len(values[0]) < *comp.MinLength {
			return ErrInvalidCredentials
		}
		if comp.MaxLength != nil && len(values) > 0 && len(values[0]) > *comp.MaxLength {
			return ErrInvalidCredentials
		}
	}

	return nil
}

// validateURL validates a URL for link buttons
func validateURL(u string) error {
	parsed, err := url.Parse(u)
	if err != nil {
		return ErrInvalidCredentials
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidCredentials
	}
	return nil
}

// ComponentInteractionEvent is published when a component is interacted with
type ComponentInteractionEvent struct {
	Interaction *models.ComponentInteraction
	Component   *models.MessageComponent
	Message     *models.Message
	ChannelID   uuid.UUID
}
