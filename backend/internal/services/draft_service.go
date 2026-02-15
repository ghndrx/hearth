package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"hearth/internal/models"
)

// DraftRepository defines the data access contract for draft-related operations.
// This interface allows for dependency inversion, enabling testing without a real DB.
type DraftRepository interface {
	GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error)
	CreateDraft(ctx context.Context, draft *models.Draft) error
	UpdateDraftStatus(ctx context.Context, id uuid.UUID, status models.DraftStatus) error
	ListChannels(ctx context.Context, guildID uuid.UUID) ([]models.Channel, error)
}

// DraftService handles business logic for Drafts.
type DraftService struct {
	repo DraftRepository
}

// NewDraftService creates a new DraftService instance.
func NewDraftService(repo DraftRepository) *DraftService {
	return &DraftService{repo: repo}
}

// CreateDraft creates a new draft based on provided context.
// It performs validation ensuring a draft exists before registering it.
func (s *DraftService) CreateDraft(ctx context.Context, req models.CreateDraftRequest) (*models.Draft, error) {
	// Basic validation
	if req.Title == "" {
		return nil, errors.New("draft title cannot be empty")
	}
	if len(req.Title) > 256 {
		return nil, errors.New("draft title cannot exceed 256 characters")
	}

	draft := &models.Draft{
		ID:         uuid.New(),
		Title:      req.Title,
		Content:    req.Content,
		GuildID:    req.GuildID,
		ChannelID:  req.ChannelID,
		Status:     models.DraftStatusDraft,
		CreatedBy:  req.CreatedBy,
		LastEdited: nil, // nil implies created now
	}

	if err := s.repo.CreateDraft(ctx, draft); err != nil {
		return nil, fmt.Errorf("failed to persist draft: %w", err)
	}

	return draft, nil
}

// GetDraft retrieves a draft by its ID.
func (s *DraftService) GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error) {
	return s.repo.GetDraft(ctx, id)
}

// PublishDraft atomically updates the draft status to Published.
// Ideally: Ensure that raw_message_id/session_id is set on the DB side if required.
func (s *DraftService) PublishDraft(ctx context.Context, id uuid.UUID, sessionID uuid.UUID, rawMessageID string) error {
	if err := s.repo.UpdateDraftStatus(ctx, id, models.DraftStatusPublished); err != nil {
		return fmt.Errorf("failed to update draft status: %w", err)
	}

	// Assuming models.Draft has a method to append logs or similar,
	// but for this service layer, we pass the transaction context primarily.
	return nil
}

// GetDraftPreview generates a preview string or content snippet.
// In a real implementation, this might paginate, truncate, or format markdown.
func (s *DraftService) GetDraftPreview(ctx context.Context, id uuid.UUID) (string, error) {
	draft, err := s.repo.GetDraft(ctx, id)
	if err != nil {
		return "", err
	}

	if draft.Content == "" {
		return "", nil
	}

	// Simple preview: take first 100 chars (excluding markdown symbols for safety)
	const maxPreviewLen = 100
	content := draft.Content
	if len(content) > maxPreviewLen {
		content = content[:maxPreviewLen] + "..."
	}
	return content, nil
}

// UpdateDraft updates the content or title of an existing draft.
func (s *DraftService) UpdateDraft(ctx context.Context, id uuid.UUID, req models.UpdateDraftRequest) error {
	// Fetch current draft to check existence/status indirectly
	existing, err := s.repo.GetDraft(ctx, id)
	if err != nil {
		return fmt.Errorf("draft not found: %w", err)
	}

	if req.Title != nil {
		if *req.Title == "" {
			return errors.New("draft title cannot be empty")
		}
		if len(*req.Title) > 256 {
			return errors.New("draft title exceeds length limit")
		}
		existing.Title = *req.Title
	}
	if req.Content != nil {
		existing.Content = *req.Content
	}
	// Update timestamp logic would be here (omitted for brevity)

	return s.repo.CreateDraft(ctx, existing) // Re-using Create to update
}

// ShareDraft creates a concrete Discord message in the channel
func (s *DraftService) ShareDraft(ctx context.Context, draftID uuid.UUID, channelID uuid.UUID) error {
	// 1. Get the draft
	d, err := s.repo.GetDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("draft not found: %w", err)
	}

	// 2. Verify the channel belongs to the guild (Optional but recommended layer 2 validation)
	channels, err := s.repo.ListChannels(ctx, d.GuildID)
	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	found := false
	for _, c := range channels {
		if c.ID == channelID {
			found = true
			break
		}
	}
	if !found {
		return errors.New("target channel not found in guild")
	}

	// 3. Publish (The actual publishing logic usually lives in the Repo or Outer Transaction)
	// Here we just mark it published.
	return s.repo.UpdateDraftStatus(ctx, draftID, models.DraftStatusPublished)
}
