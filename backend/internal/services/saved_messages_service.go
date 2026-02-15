package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"hearth/internal/models"
)

var (
	// ErrMessageAlreadySaved indicates the message is already saved
	ErrMessageAlreadySaved = errors.New("message already saved")
	// ErrSavedMessageNotFound indicates the saved message entry was not found
	ErrSavedMessageNotFound = errors.New("saved message not found")
	// ErrUnauthorized indicates the user doesn't have permission
	ErrUnauthorized = errors.New("unauthorized")
)

// SavedMessagesRepository defines the interface for saved messages data access
type SavedMessagesRepository interface {
	Save(ctx context.Context, userID, messageID uuid.UUID, note *string) (*models.SavedMessage, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.SavedMessage, error)
	GetByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) (*models.SavedMessage, error)
	GetByUser(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error)
	GetByUserWithMessages(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error)
	UpdateNote(ctx context.Context, id uuid.UUID, note *string) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByUserAndMessage(ctx context.Context, userID, messageID uuid.UUID) error
	Count(ctx context.Context, userID uuid.UUID) (int, error)
	IsSaved(ctx context.Context, userID, messageID uuid.UUID) (bool, error)
}

// MessageExistsChecker defines the interface for checking message existence
type MessageExistsChecker interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Message, error)
}

// SavedMessagesService handles saved messages business logic
type SavedMessagesService struct {
	repo     SavedMessagesRepository
	msgRepo  MessageExistsChecker
	eventBus EventBus
}

// NewSavedMessagesService creates a new saved messages service
func NewSavedMessagesService(repo SavedMessagesRepository, msgRepo MessageExistsChecker, eventBus EventBus) *SavedMessagesService {
	return &SavedMessagesService{
		repo:     repo,
		msgRepo:  msgRepo,
		eventBus: eventBus,
	}
}

// SaveMessage saves/bookmarks a message for a user
func (s *SavedMessagesService) SaveMessage(ctx context.Context, userID, messageID uuid.UUID, note *string) (*models.SavedMessage, error) {
	// Check if the message exists
	message, err := s.msgRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}

	// Save the message (upsert handles duplicates)
	saved, err := s.repo.Save(ctx, userID, messageID, note)
	if err != nil {
		return nil, err
	}

	// Attach the message to the response
	saved.Message = message

	// Emit event
	s.eventBus.Publish("user.message_saved", &MessageSavedEvent{
		UserID:       userID,
		MessageID:    messageID,
		SavedMessage: saved,
		SavedAt:      saved.CreatedAt,
	})

	return saved, nil
}

// GetSavedMessages retrieves all saved messages for a user
func (s *SavedMessagesService) GetSavedMessages(ctx context.Context, userID uuid.UUID, opts *models.SavedMessagesQueryOptions) ([]*models.SavedMessage, error) {
	return s.repo.GetByUserWithMessages(ctx, userID, opts)
}

// GetSavedMessage retrieves a specific saved message by ID
func (s *SavedMessagesService) GetSavedMessage(ctx context.Context, userID, savedID uuid.UUID) (*models.SavedMessage, error) {
	saved, err := s.repo.GetByID(ctx, savedID)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, ErrSavedMessageNotFound
	}
	if saved.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Fetch the message
	message, err := s.msgRepo.GetByID(ctx, saved.MessageID)
	if err != nil {
		return nil, err
	}
	saved.Message = message

	return saved, nil
}

// UpdateSavedMessageNote updates the note on a saved message
func (s *SavedMessagesService) UpdateSavedMessageNote(ctx context.Context, userID, savedID uuid.UUID, note *string) (*models.SavedMessage, error) {
	saved, err := s.repo.GetByID(ctx, savedID)
	if err != nil {
		return nil, err
	}
	if saved == nil {
		return nil, ErrSavedMessageNotFound
	}
	if saved.UserID != userID {
		return nil, ErrUnauthorized
	}

	err = s.repo.UpdateNote(ctx, savedID, note)
	if err != nil {
		return nil, err
	}

	saved.Note = note

	// Fetch the message
	message, err := s.msgRepo.GetByID(ctx, saved.MessageID)
	if err != nil {
		return nil, err
	}
	saved.Message = message

	return saved, nil
}

// RemoveSavedMessage removes a saved message by its ID
func (s *SavedMessagesService) RemoveSavedMessage(ctx context.Context, userID, savedID uuid.UUID) error {
	saved, err := s.repo.GetByID(ctx, savedID)
	if err != nil {
		return err
	}
	if saved == nil {
		return ErrSavedMessageNotFound
	}
	if saved.UserID != userID {
		return ErrUnauthorized
	}

	err = s.repo.Delete(ctx, savedID)
	if err != nil {
		return err
	}

	// Emit event
	s.eventBus.Publish("user.message_unsaved", &MessageUnsavedEvent{
		UserID:    userID,
		MessageID: saved.MessageID,
		SavedID:   savedID,
		RemovedAt: time.Now(),
	})

	return nil
}

// RemoveSavedMessageByMessageID removes a saved message by the original message ID
func (s *SavedMessagesService) RemoveSavedMessageByMessageID(ctx context.Context, userID, messageID uuid.UUID) error {
	saved, err := s.repo.GetByUserAndMessage(ctx, userID, messageID)
	if err != nil {
		return err
	}
	if saved == nil {
		return ErrSavedMessageNotFound
	}

	err = s.repo.DeleteByUserAndMessage(ctx, userID, messageID)
	if err != nil {
		return err
	}

	// Emit event
	s.eventBus.Publish("user.message_unsaved", &MessageUnsavedEvent{
		UserID:    userID,
		MessageID: messageID,
		SavedID:   saved.ID,
		RemovedAt: time.Now(),
	})

	return nil
}

// IsSaved checks if a message is saved by a user
func (s *SavedMessagesService) IsSaved(ctx context.Context, userID, messageID uuid.UUID) (bool, error) {
	return s.repo.IsSaved(ctx, userID, messageID)
}

// GetSavedCount returns the number of saved messages for a user
func (s *SavedMessagesService) GetSavedCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.Count(ctx, userID)
}

// Events

// MessageSavedEvent is emitted when a user saves/bookmarks a message
type MessageSavedEvent struct {
	UserID       uuid.UUID
	MessageID    uuid.UUID
	SavedMessage *models.SavedMessage
	SavedAt      time.Time
}

// MessageUnsavedEvent is emitted when a user removes a saved message
type MessageUnsavedEvent struct {
	UserID    uuid.UUID
	MessageID uuid.UUID
	SavedID   uuid.UUID
	RemovedAt time.Time
}
