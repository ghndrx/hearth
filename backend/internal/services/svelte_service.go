package services

import (
	"context"
	"errors"
	"hearth/internal/models"
	"github.com/google/uuid"
)

var (
	ErrSvelteNotFound = errors.New("svelte component not found")
	ErrInvalidInput   = errors.New("invalid input provided")
)

// SvelteRepository defines the data access methods required by the service.
// This adheres to the rule of not importing internal database packages directly.
type SvelteRepository interface {
	Create(ctx context.Context, svelte *models.Svelte) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Svelte, error)
	Update(ctx context.Context, svelte *models.Svelte) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SvelteService handles business logic for Svelte components.
type SvelteService struct {
	repo SvelteRepository
}

// NewSvelteService creates a new instance of SvelteService.
func NewSvelteService(repo SvelteRepository) *SvelteService {
	return &SvelteService{
		repo: repo,
	}
}

// CreateComponent validates and creates a new Svelte component instance.
func (s *SvelteService) CreateComponent(ctx context.Context, name, code string) (*models.Svelte, error) {
	if name == "" || code == "" {
		return nil, ErrInvalidInput
	}

	newSvelte := &models.Svelte{
		ID:   uuid.New(),
		Name: name,
		Code: code,
	}

	if err := s.repo.Create(ctx, newSvelte); err != nil {
		return nil, err
	}

	return newSvelte, nil
}

// GetComponent retrieves a component by its ID.
func (s *SvelteService) GetComponent(ctx context.Context, id uuid.UUID) (*models.Svelte, error) {
	component, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, ErrSvelteNotFound
	}
	return component, nil
}

// UpdateComponent modifies an existing component.
func (s *SvelteService) UpdateComponent(ctx context.Context, id uuid.UUID, name, code string) error {
	if name == "" || code == "" {
		return ErrInvalidInput
	}

	// Check existence first
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSvelteNotFound
	}

	// Update fields
	existing.Name = name
	existing.Code = code

	return s.repo.Update(ctx, existing)
}

// DeleteComponent removes a component by its ID.
func (s *SvelteService) DeleteComponent(ctx context.Context, id uuid.UUID) error {
	// Check existence first
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSvelteNotFound
	}

	return s.repo.Delete(ctx, id)
}