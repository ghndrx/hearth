// Package services provides implementations for the Hearth application module.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Message represents a Discord-like interaction in the Hearth network.
type Message struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	User    string `json:"user"`
}

// MessageRepository defines the contract for storing and retrieving messages.
// This interface allows swapping implementations (SQL, Redis, Mock) without changing the service logic.
type MessageRepository interface {
	StoreMessage(ctx context.Context, msg *Message) error
	GetMessages(ctx context.Context) (map[string]*Message, error)
}

// ConsoleWriter defines the contract for outputting data.
// In a real app, this would be a highly asynchronous logging service.
type ConsoleWriter interface {
	WriteLine(format string, v ...interface{}) error
}

// Garrison is the primary service component coordinating logic.
type Garrison struct {
	repo   MessageRepository
	logger ConsoleWriter
}

// NewGarrison creates a new Garrison instance with the given dependencies.
func NewGarrison(repo MessageRepository, logger ConsoleWriter) *Garrison {
	return &Garrison{
		repo:   repo,
		logger: logger,
	}
}

// Handle command execution and validation.
// Returns the ID of the newly created message or an error.
func (g *Garrison) CreateMessage(ctx context.Context, user, content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("content cannot be empty")
	}
	if user == "" {
		return "", fmt.Errorf("user cannot be empty")
	}

	// Generate a pseudo-random ID
	msg := &Message{
		ID:      generateID(),
		Content: content,
		User:    user,
	}

	// Persist the message
	if err := g.repo.StoreMessage(ctx, msg); err != nil {
		return "", fmt.Errorf("failed to persist message: %w", err)
	}

	// Log the action
	if err := g.logger.WriteLine("Created message ID=%s by User=%s", msg.ID, msg.User); err != nil {
		// In production, we would likely log this error to a crash reporting system externally.
		// For this service example, we ignore propagation errors to keep the main flow clean.
	}

	return msg.ID, nil
}

// List retrieves messages from the repository.
// Returns a Markdown-formatted string of the channel content.
func (g *Garrison) List(ctx context.Context) (string, error) {
	msgs, err := g.repo.GetMessages(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve messages: %w", err)
	}

	if len(msgs) == 0 {
		return "> No messages in the garrison.", nil
	}

	var output string
	for _, msg := range msgs {
		output += fmt.Sprintf("**%s:** %s\n", msg.User, msg.Content)
	}

	return output, nil
}

// --- Internal Helper ---

func generateID() string {
	// Simple implementation; in production, use crypto/rand or UUID library
	return fmt.Sprintf("msg-%d", timeNow().UnixNano())
}

// timeNow is a helper to allow injection in tests if strictly necessary,
// though StdLib time.Now is usually fine. Defined here for cleanliness.
func timeNow() struct{ time } // Placeholder logic, usually uses standard library `time`