package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

var (
	ErrSvelteNotFound = errors.New("svelte component not found")
	ErrInvalidInput   = errors.New("invalid input provided")
)

// SvelteRepository defines the data access methods for Svelte components.
// This decouples the service layer from the database implementation.
type SvelteRepository interface {
	Create(ctx context.Context, component *models.SvelteComponent) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SvelteComponent, error)
	Update(ctx context.Context, component *models.SvelteComponent) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.SvelteComponent, error)
}

// SvelteService handles business logic related to Svelte components
// within the Hearth application.
type SvelteService struct {
	repo SvelteRepository
}

// NewSvelteService creates a new SvelteService instance.
func NewSvelteService(repo SvelteRepository) *SvelteService {
	return &SvelteService{
		repo: repo,
	}
}

// CreateComponent attempts to register a new Svelte component.
func (s *SvelteService) CreateComponent(ctx context.Context, name, content string, userID uuid.UUID) (*models.SvelteComponent, error) {
	if name == "" || content == "" {
		return nil, ErrInvalidInput
	}

	component := &models.SvelteComponent{
		ID:        uuid.New(),
		Name:      name,
		Content:   content,
		AuthorID:  userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, component); err != nil {
		return nil, err
	}

	return component, nil
}

// GetComponent retrieves a specific component by its ID.
func (s *SvelteService) GetComponent(ctx context.Context, componentID uuid.UUID) (*models.SvelteComponent, error) {
	component, err := s.repo.GetByID(ctx, componentID)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, ErrSvelteNotFound
	}
	return component, nil
}

// UpdateComponent modifies an existing component's content.
func (s *SvelteService) UpdateComponent(ctx context.Context, componentID uuid.UUID, newContent string) (*models.SvelteComponent, error) {
	if newContent == "" {
		return nil, ErrInvalidInput
	}

	component, err := s.GetComponent(ctx, componentID)
	if err != nil {
		return nil, err
	}

	component.Content = newContent
	component.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, component); err != nil {
		return nil, err
	}

	return component, nil
}

// DeleteComponent removes a component from the system.
func (s *SvelteService) DeleteComponent(ctx context.Context, componentID uuid.UUID) error {
	// Check existence first usually good practice, or rely on DB count
	_, err := s.repo.GetByID(ctx, componentID)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, componentID)
}