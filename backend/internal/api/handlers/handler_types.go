package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"hearth/internal/models"
)

// Handler provides a minimal type definition for legacy handler methods.
// TODO: refactor follow.go, pins.go, reminders.go, soundboard.go to use proper service dependencies.
type Handler struct {
	db    HandlerDB
	wsHub WsHubInterface
}

// HandlerDB defines the database interface used by legacy handler methods.
type HandlerDB interface {
	CreateFollowedChannel(ctx context.Context, follow *models.FollowedChannel) error
	DeleteFollowedChannel(ctx context.Context, channelID, followerChannelID string) error
	GetFollowedChannels(ctx context.Context, channelID string) ([]models.FollowedChannel, error)
	GetPinnedMessages(ctx context.Context, channelID string) ([]*models.Message, error)
	CreatePin(ctx context.Context, pin *models.Pin) error
	DeletePin(ctx context.Context, messageID string) error
	CreateReminder(ctx context.Context, reminder *models.Reminder) error
	GetUserReminders(ctx context.Context, userID string) ([]models.Reminder, error)
	DeleteReminder(ctx context.Context, reminderID, userID string) error
	GetSounds(ctx context.Context, serverID string) ([]models.SoundboardSound, error)
	CreateSound(ctx context.Context, sound *models.Sound) error
	DeleteSound(ctx context.Context, soundID string) error
	GetSound(ctx context.Context, soundID string) (*models.Sound, error)
}

// WsHubInterface defines the WebSocket hub interface used by soundboard handlers.
type WsHubInterface interface {
	BroadcastToChannel(serverID string, payload map[string]interface{})
}

// generateID returns a new UUID string for ID fields.
func generateID() string {
	return uuid.New().String()
}

// decodeJSON decodes a JSON request body into the given value.
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// jsonResponse writes a JSON response with the given status code.
func jsonResponse(w http.ResponseWriter, v interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// userFromRequest extracts a minimal user from the request context.
// TODO: implement proper user extraction from auth middleware context.
type minimalUser struct {
	ID string
}

func getUser(r *http.Request) *minimalUser {
	// Stub: in production this would extract user from auth context
	return &minimalUser{ID: uuid.New().String()}
}
