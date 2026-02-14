package services

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var ErrSvelteNotFound = errors.New("svelte component not found")

// Svelte represents a Svelte component or session resource.
type Svelte struct {
	ID        uuid.UUID
	Name      string
	Component string
	Props     map[string]interface{}
}

// SvelteService manages Svelte component resources.
type SvelteService struct {
	mu         sync.RWMutex
	components map[uuid.UUID]*Svelte
}

// NewSvelteService creates a new instance of the SvelteService.
func NewSvelteService() *SvelteService {
	return &SvelteService{
		components: make(map[uuid.UUID]*Svelte),
	}
}

// Create creates a new Svelte component.
func (s *SvelteService) Create(ctx context.Context, name, component string, props map[string]interface{}) (*Svelte, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if component == "" {
		return nil, errors.New("component cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	svelte := &Svelte{
		ID:        uuid.New(),
		Name:      name,
		Component: component,
		Props:     props,
	}
	s.components[svelte.ID] = svelte
	return svelte, nil
}

// Get retrieves a Svelte component by ID.
func (s *SvelteService) Get(ctx context.Context, id uuid.UUID) (*Svelte, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid UUID")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if svelte, ok := s.components[id]; ok {
		return svelte, nil
	}
	return nil, ErrSvelteNotFound
}

// Update updates an existing Svelte component.
func (s *SvelteService) Update(ctx context.Context, id uuid.UUID, name, component string, props map[string]interface{}) (*Svelte, error) {
	if id == uuid.Nil {
		return nil, errors.New("invalid UUID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.components[id]
	if !ok {
		return nil, ErrSvelteNotFound
	}

	if name != "" {
		existing.Name = name
	}
	if component != "" {
		existing.Component = component
	}
	if props != nil {
		existing.Props = props
	}

	return existing, nil
}

// Delete removes a Svelte component.
func (s *SvelteService) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("invalid UUID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.components[id]; ok {
		delete(s.components, id)
		return nil
	}
	return ErrSvelteNotFound
}

// List returns all Svelte components.
func (s *SvelteService) List(ctx context.Context) ([]*Svelte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Svelte, 0, len(s.components))
	for _, svelte := range s.components {
		result = append(result, svelte)
	}
	return result, nil
}
