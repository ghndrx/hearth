package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hearth/internal/models"
)

// WebhookHandlers handles webhook-related HTTP requests
type WebhookHandlers struct {
	// webhookRepo would go here
}

// NewWebhookHandlers creates new webhook handlers
func NewWebhookHandlers() *WebhookHandlers {
	return &WebhookHandlers{}
}

// CreateWebhookRequest represents a webhook creation request
type CreateWebhookRequest struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"` // Base64 image data
}

// WebhookResponse represents a webhook in API responses
type WebhookResponse struct {
	ID        string  `json:"id"`
	Type      int     `json:"type"` // 1 = incoming, 2 = channel follower
	ServerID  string  `json:"guild_id,omitempty"`
	ChannelID string  `json:"channel_id"`
	CreatorID *string `json:"user,omitempty"`
	Name      string  `json:"name"`
	Avatar    *string `json:"avatar,omitempty"`
	Token     string  `json:"token,omitempty"`
	URL       string  `json:"url,omitempty"`
}

// CreateWebhook creates a new webhook for a channel
func (h *WebhookHandlers) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Name is required")
		return
	}
	if len(req.Name) > 80 {
		respondError(w, http.StatusBadRequest, "Name must be 80 characters or less")
		return
	}

	// Generate secure token
	token, err := generateWebhookToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	webhook := &models.Webhook{
		ID:        uuid.New(),
		ChannelID: channelID,
		CreatorID: &userID,
		Name:      req.Name,
		Token:     token,
		CreatedAt: time.Now(),
	}

	// TODO: Save webhook to database
	_ = webhook

	response := WebhookResponse{
		ID:        webhook.ID.String(),
		Type:      1,
		ChannelID: channelID.String(),
		Name:      req.Name,
		Token:     token,
		URL:       buildWebhookURL(webhook.ID, token),
	}

	respondJSON(w, http.StatusOK, response)
}

// GetChannelWebhooks gets all webhooks for a channel
func (h *WebhookHandlers) GetChannelWebhooks(w http.ResponseWriter, r *http.Request) {
	_ = getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	// TODO: Get webhooks from database
	_ = channelID

	respondJSON(w, http.StatusOK, []WebhookResponse{})
}

// GetServerWebhooks gets all webhooks for a server
func (h *WebhookHandlers) GetServerWebhooks(w http.ResponseWriter, r *http.Request) {
	_ = getUserID(r)
	serverID, err := uuid.Parse(chi.URLParam(r, "serverID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid server ID")
		return
	}

	// TODO: Get webhooks from database
	_ = serverID

	respondJSON(w, http.StatusOK, []WebhookResponse{})
}

// GetWebhook gets a single webhook
func (h *WebhookHandlers) GetWebhook(w http.ResponseWriter, r *http.Request) {
	_ = getUserID(r)
	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid webhook ID")
		return
	}

	// TODO: Get webhook from database
	_ = webhookID

	respondError(w, http.StatusNotFound, "Webhook not found")
}

// GetWebhookWithToken gets a webhook without auth (using token)
func (h *WebhookHandlers) GetWebhookWithToken(w http.ResponseWriter, r *http.Request) {
	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid webhook ID")
		return
	}
	token := chi.URLParam(r, "token")

	// TODO: Validate token and get webhook
	_, _ = webhookID, token

	respondError(w, http.StatusNotFound, "Webhook not found")
}

// UpdateWebhook updates a webhook
func (h *WebhookHandlers) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	_ = getUserID(r)
	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid webhook ID")
		return
	}

	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Update webhook in database
	_ = webhookID

	respondError(w, http.StatusNotFound, "Webhook not found")
}

// DeleteWebhook deletes a webhook
func (h *WebhookHandlers) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	_ = getUserID(r)
	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid webhook ID")
		return
	}

	// TODO: Delete webhook from database
	_ = webhookID

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteWebhook executes a webhook (sends a message)
func (h *WebhookHandlers) ExecuteWebhook(w http.ResponseWriter, r *http.Request) {
	webhookID, err := uuid.Parse(chi.URLParam(r, "webhookID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid webhook ID")
		return
	}
	token := chi.URLParam(r, "token")

	var req struct {
		Content   string         `json:"content"`
		Username  string         `json:"username,omitempty"`
		AvatarURL string         `json:"avatar_url,omitempty"`
		TTS       bool           `json:"tts"`
		Embeds    []models.Embed `json:"embeds,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content == "" && len(req.Embeds) == 0 {
		respondError(w, http.StatusBadRequest, "Message must have content or embeds")
		return
	}

	// TODO: Validate token, create message
	_, _ = webhookID, token

	// Query param: wait=true returns the message
	wait := r.URL.Query().Get("wait") == "true"
	if wait {
		respondJSON(w, http.StatusOK, MessageResponse{})
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// Helper functions

func generateWebhookToken() (string, error) {
	bytes := make([]byte, 34)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func buildWebhookURL(id uuid.UUID, token string) string {
	// TODO: Use config for base URL
	return "https://api.hearth.example/webhooks/" + id.String() + "/" + token
}
