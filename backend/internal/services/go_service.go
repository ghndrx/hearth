package services

import (
	"context"
	"errors"
	"hearth/internal/models"

	"github.com/google/uuid"
)

var (
	ErrServerNotFound = errors.New("server not found")
	ErrUserNotFound   = errors.New("user not found")
)

// ServerRepository defines the data access methods for Servers.
type ServerRepository interface {
	Create(ctx context.Context, server *models.Server) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error)
	GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Server, error)
	Update(ctx context.Context, server *models.Server) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ServerService handles business logic for Discord servers.
type ServerService struct {
	repo ServerRepository
}

// NewServerService creates a new ServerService.
func NewServerService(repo ServerRepository) *ServerService {
	return &ServerService{
		repo: repo,
	}
}

// CreateServer creates a new server.
func (s *ServerService) CreateServer(ctx context.Context, name string, ownerID uuid.UUID) (*models.Server, error) {
	if name == "" {
		return nil, errors.New("server name cannot be empty")
	}

	server := &models.Server{
		ID:      uuid.New(),
		Name:    name,
		OwnerID: ownerID,
		// Default settings can be applied here
	}

	if err := s.repo.Create(ctx, server); err != nil {
		return nil, err
	}

	return server, nil
}

// GetServer retrieves a server by its ID.
func (s *ServerService) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	server, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// In a real scenario, we might check if the error is a "not found" error from the driver
		// and return ErrServerNotFound specifically.
		return nil, ErrServerNotFound
	}
	return server, nil
}

// GetUserServers retrieves all servers owned by a specific user.
func (s *ServerService) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	return s.repo.GetByOwner(ctx, userID)
}

// DeleteServer deletes a server by its ID.
func (s *ServerService) DeleteServer(ctx context.Context, id uuid.UUID) error {
	// Verify existence before deletion (optional logic check, can be done in repo)
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			return ErrServerNotFound
		}
		return err
	}

	return s.repo.Delete(ctx, id)
}