// File: services/comprehensive_service.go
package services

import (
	"context"
	"errors"
	"hearth/internal/models"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnauthorizedAccess = errors.New("unauthorized access")
	ErrInvalidInput       = errors.New("invalid input")
)

// comprehensiveRepository defines the data access methods required by the service.
// This adherence to interfaces ensures the service layer is decoupled from the database implementation.
type comprehensiveRepository interface {
	// User Operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)

	// Server Operations
	CreateServer(ctx context.Context, server *models.Server) error
	GetServerByID(ctx context.Context, serverID uuid.UUID) (*models.Server, error)
	AddMemberToServer(ctx context.Context, serverID, userID uuid.UUID) error
	IsServerMember(ctx context.Context, serverID, userID uuid.UUID) (bool, error)

	// Channel Operations
	CreateChannel(ctx context.Context, channel *models.Channel) error
	GetChannelByID(ctx context.Context, channelID uuid.UUID) (*models.Channel, error)
	GetChannelsByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error)

	// Message Operations
	CreateMessage(ctx context.Context, message *models.Message) error
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*models.Message, error)
	GetMessagesByChannel(ctx context.Context, channelID uuid.UUID, limit int) ([]*models.Message, error)
}

// ComprehensiveService handles business logic for the Hearth application.
type ComprehensiveService struct {
	repo comprehensiveRepository
}

// NewComprehensiveService creates a new service instance.
func NewComprehensiveService(repo comprehensiveRepository) *ComprehensiveService {
	return &ComprehensiveService{
		repo: repo,
	}
}

// --- User Methods ---

// RegisterUser creates a new user account.
func (s *ComprehensiveService) RegisterUser(ctx context.Context, username, email, passwordHash string) (*models.User, error) {
	if username == "" || email == "" || passwordHash == "" {
		return nil, ErrInvalidInput
	}

	// Check if username exists
	existing, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil && err != ErrUserNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	user := &models.User{
		ID:           uuid.New(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser retrieves a user by ID.
func (s *ComprehensiveService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// --- Server Methods ---

// CreateServer creates a new server and assigns the creator as the owner.
func (s *ComprehensiveService) CreateServer(ctx context.Context, name string, ownerID uuid.UUID) (*models.Server, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}

	// Verify owner exists
	_, err := s.repo.GetUserByID(ctx, ownerID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	server := &models.Server{
		ID:        uuid.New(),
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateServer(ctx, server); err != nil {
		return nil, err
	}

	// Add the creator as the first member
	if err := s.repo.AddMemberToServer(ctx, server.ID, ownerID); err != nil {
		// In a real transaction we would roll back the server creation here
		return nil, err
	}

	// Create default "general" text channel
	generalChannel := &models.Channel{
		ID:        uuid.New(),
		ServerID:  server.ID,
		Name:      "general",
		Type:      "text",
		Position:  0,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateChannel(ctx, generalChannel); err != nil {
		// Non-fatal: server is created, just log and continue
		// In production, use proper logging
	}

	return server, nil
}

// JoinServer adds a user to a server.
func (s *ComprehensiveService) JoinServer(ctx context.Context, serverID, userID uuid.UUID) error {
	// Verify user and server exist
	_, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	_, err = s.repo.GetServerByID(ctx, serverID)
	if err != nil {
		return ErrServerNotFound
	}

	// Check if already member
	isMember, err := s.repo.IsServerMember(ctx, serverID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyMember
	}

	return s.repo.AddMemberToServer(ctx, serverID, userID)
}

// --- Channel Methods ---

// CreateChannel creates a new channel within a server.
func (s *ComprehensiveService) CreateChannel(ctx context.Context, serverID uuid.UUID, name, channelType string) (*models.Channel, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}

	// Check if requester has access (simplified: check if server exists)
	_, err := s.repo.GetServerByID(ctx, serverID)
	if err != nil {
		return nil, ErrServerNotFound
	}

	channel := &models.Channel{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      name,
		Type:      models.ChannelType(channelType),
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateChannel(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetChannels retrieves all channels for a server.
func (s *ComprehensiveService) GetChannels(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	channels, err := s.repo.GetChannelsByServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// --- Message Methods ---

// SendMessage sends a message to a channel.
func (s *ComprehensiveService) SendMessage(ctx context.Context, channelID, authorID uuid.UUID, content string) (*models.Message, error) {
	if content == "" {
		return nil, ErrInvalidInput
	}

	// Verify channel exists
	channel, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return nil, ErrChannelNotFound
	}

	// Verify user is a member of the server the channel belongs to
	if channel.ServerID != nil {
		isMember, err := s.repo.IsServerMember(ctx, *channel.ServerID, authorID)
		if err != nil {
			return nil, err
		}
		if !isMember {
			return nil, ErrUnauthorizedAccess
		}
	}

	message := &models.Message{
		ID:        uuid.New(),
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

// GetMessages retrieves recent messages from a channel.
func (s *ComprehensiveService) GetMessages(ctx context.Context, channelID uuid.UUID, limit int) ([]*models.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	messages, err := s.repo.GetMessagesByChannel(ctx, channelID, limit)
	if err != nil {
		return nil, err
	}
	return messages, nil
}
