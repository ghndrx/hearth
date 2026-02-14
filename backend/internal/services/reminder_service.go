package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrReminderNotFound = errors.New("reminder not found")

// Reminder represents a user reminder
type Reminder struct {
	ID        uuid.UUID
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Content   string
	CreatedAt time.Time
}

// ReminderRepository defines the contract for Reminder data persistence
// This interface decouples the service from the specific database implementation.
type ReminderRepository interface {
	// Create persists a reminder.
	Create(ctx context.Context, reminder Reminder) error
	// GetByID retrieves a single reminder by its UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*Reminder, error)
	// Update modifies an existing reminder.
	Update(ctx context.Context, reminder Reminder) error
	// Delete removes a reminder by its ID.
	Delete(ctx context.Context, id uuid.UUID) error
	// GetRemindersByChannel retrieves all reminders for a specific channel.
	GetRemindersByChannel(ctx context.Context, channelID uuid.UUID) ([]Reminder, error)
}

// ReminderService handles business logic for reminders.
type ReminderService struct {
	repo ReminderRepository
}

// NewReminderService creates a new ReminderService instance.
func NewReminderService(repo ReminderRepository) *ReminderService {
	return &ReminderService{
		repo: repo,
	}
}

// Create creates a new Reminder instance.
// It validates the input and sets empty time/rate if necessary.
func (s *ReminderService) Create(ctx context.Context, channelID, userID uuid.UUID, content string) (*Reminder, error) {
	modelReminder := Reminder{
		ID:        uuid.New(),
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, modelReminder); err != nil {
		return nil, fmt.Errorf("failed to create reminder in repository: %w", err)
	}

	return &modelReminder, nil
}

// Get retrieves a reminder by its ID.
func (s *ReminderService) Get(ctx context.Context, id uuid.UUID) (*Reminder, error) {
	if id == uuid.Nil {
		return nil, errors.New("reminder ID cannot be empty")
	}

	reminder, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return reminder, nil
}

// Delete removes a reminder.
func (s *ReminderService) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("reminder ID cannot be empty")
	}

	// Check existence before deleting to ensure data integrity (optional but good practice)
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("reminder not found: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}

	return nil
}

// GetRemindersForChannel retrieves all active reminders for a specific channel.
// This is useful for clients to fetch pending notifications.
func (s *ReminderService) GetRemindersForChannel(ctx context.Context, channelID uuid.UUID) ([]Reminder, error) {
	if channelID == uuid.Nil {
		return nil, errors.New("channel ID cannot be empty")
	}

	return s.repo.GetRemindersByChannel(ctx, channelID)
}

// ProcessReminders mocks a "Check and Send" behavior.
// In a real backend, this would query for due items and send websocket/webhook events.
func (s *ReminderService) ProcessReminders(ctx context.Context) ([]Reminder, error) {
	// Placeholder for business logic that selects remders based on time
	// Since we don't have a specific Time field in the simplified models/pkg,
	// we return a mock list filtered by channel.
	// Note: This assumes Repository.GetRemindersByChannel returns active all.
	return s.repo.GetRemindersByChannel(ctx, uuid.Nil)
}
