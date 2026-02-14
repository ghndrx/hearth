package services

import (
	"context"
	"errors"
	"hearth/internal/models"

	"github.com/google/uuid"
)

// WebSocketRepository defines the interface for storage operations required by the WebSocket service.
// The concrete implementation (e.g., in the database package) must satisfy this.
type WebSocketRepository interface {
 SaveActiveSession(ctx context.Context, session *models.Session) error
 RemoveActiveSession(ctx context.Context, sessionID uuid.UUID) error
 FindActiveSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, error)
}

// WebSocketService handles WebSocket connections, message routing, and session lifecycle.
type WebSocketService struct {
 repo WebSocketRepository
}

// NewWebSocketService creates a new instance of WebSocketService.
func NewWebSocketService(repo WebSocketRepository) *WebSocketService {
 return &WebSocketService{
  repo: repo,
 }
}

// ConnectSession handles the logic for a user establishing a WebSocket connection.
func (s *WebSocketService) ConnectSession(ctx context.Context, userID uuid.UUID, connID uuid.UUID) (*models.Session, error) {
 // Basic validation
 if userID == uuid.Nil || connID == uuid.Nil {
  return nil, errors.New("invalid user or connection ID")
 }

 session := &models.Session{
  ID:   uuid.New(),
  UserID: userID,
  ConnectionID: connID,
 }

 if err := s.repo.SaveActiveSession(ctx, session); err != nil {
  return nil, errors.New("failed to establish session in repository")
 }

 return session, nil
}

// DisconnectSession handles the cleanup when a user disconnects.
func (s *WebSocketService) DisconnectSession(ctx context.Context, sessionID uuid.UUID) error {
 if sessionID == uuid.Nil {
  return errors.New("invalid session ID")
 }

 _, err := s.repo.FindActiveSession(ctx, sessionID)
 if err != nil {
  // Depending on repo implementation, might be "not found" or an actual db error
  return errors.New("session not found or already disconnected")
 }

 if err := s.repo.RemoveActiveSession(ctx, sessionID); err != nil {
  return errors.New("failed to remove session from repository")
 }

 return nil
}