package services

import (
	"context"
	"errors"
	"hearth/internal/models"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized access")
	ErrInvalidInput   = errors.New("invalid input")
	ErrAlreadyExists  = errors.New("resource already exists")
)

// comprehensiveRepository defines the data access methods required by the service.
// This decouples the service logic from the database implementation.
type comprehensiveRepository interface {
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)

	// Server operations
	CreateServer(ctx context.Context, server *models.Server) (*models.Server, error)
	GetServerByID(ctx context.Context, id uuid.UUID) (*models.Server, error)
	AddMemberToServer(ctx context.Context, serverID, userID uuid.UUID) error
	IsMemberOfServer(ctx context.Context, serverID, userID uuid.UUID) (bool, error)

	// Message operations
	CreateMessage(ctx context.Context, message *models.Message) (*models.Message, error)
	GetMessagesByChannel(ctx context.Context, channelID uuid.UUID, limit, offset int) ([]*models.Message, error)
}

// comprehensiveService handles business logic for the Hearth application.
type comprehensiveService struct {
	repo comprehensiveRepository
}

// NewComprehensiveService creates a new service instance.
func NewComprehensiveService(repo comprehensiveRepository) *comprehensiveService {
	return &comprehensiveService{
		repo: repo,
	}
}

// --- User Methods ---

// RegisterUser creates a new user account.
func (s *comprehensiveService) RegisterUser(ctx context.Context, username, email string) (*models.User, error) {
	if username == "" || email == "" {
		return nil, ErrInvalidInput
	}

	// Check if user exists
	existing, _ := s.repo.GetUserByUsername(ctx, username)
	if existing != nil {
		return nil, ErrAlreadyExists
	}

	newUser := &models.User{
		ID:       uuid.New(),
		Username: username,
		Email:    email,
	}

	if err := s.repo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

// GetUser retrieves a user by ID.
func (s *comprehensiveService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if userID == uuid.Nil {
		return nil, ErrInvalidInput
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrNotFound
	}

	return user, nil
}

// --- Server Methods ---

// CreateServer establishes a new server and assigns the creator as the owner.
func (s *comprehensiveService) CreateServer(ctx context.Context, name string, ownerID uuid.UUID) (*models.Server, error) {
	if name == "" || ownerID == uuid.Nil {
		return nil, ErrInvalidInput
	}

	// Verify owner exists
	_, err := s.repo.GetUserByID(ctx, ownerID)
	if err != nil {
		return nil, ErrNotFound
	}

	newServer := &models.Server{
		ID:      uuid.New(),
		Name:    name,
		OwnerID: ownerID,
	}

	server, err := s.repo.CreateServer(ctx, newServer)
	if err != nil {
		return nil, err
	}

	// Owner is automatically a member
	if err := s.repo.AddMemberToServer(ctx, server.ID, ownerID); err != nil {
		// Rollback logic could go here, but for now we return error
		return nil, err
	}

	return server, nil
}

// JoinServer adds a user to a server.
func (s *comprehensiveService) JoinServer(ctx context.Context, serverID, userID uuid.UUID) error {
	if serverID == uuid.Nil || userID == uuid.Nil {
		return ErrInvalidInput
	}

	// Verify server exists
	_, err := s.repo.GetServerByID(ctx, serverID)
	if err != nil {
		return ErrNotFound
	}

	// Check membership
	isMember, err := s.repo.IsMemberOfServer(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyExists
	}

	return s.repo.AddMemberToServer(ctx, serverID, userID)
}

// --- Message Methods ---

// SendMessage sends a message to a channel.
func (s *comprehensiveService) SendMessage(ctx context.Context, channelID, authorID uuid.UUID, content string) (*models.Message, error) {
	if channelID == uuid.Nil || authorID == uuid.Nil || content == "" {
		return nil, ErrInvalidInput
	}

	// Verify user exists
	if _, err := s.repo.GetUserByID(ctx, authorID); err != nil {
		return nil, ErrNotFound
	}

	// Note: In a real app, we would also verify the user is a member of the server
	// that owns this channel. Assuming channel validation is handled or omitted for brevity.

	msg := &models.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   content,
	}

	return s.repo.CreateMessage(ctx, msg)
}