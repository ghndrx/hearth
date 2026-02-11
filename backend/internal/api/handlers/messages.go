package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"hearth/internal/models"
	"hearth/internal/services"
)

// MessageHandlers handles message-related HTTP requests
type MessageHandlers struct {
	messageService *services.MessageService
	channelService *services.ChannelService
}

// NewMessageHandlers creates new message handlers
func NewMessageHandlers(
	messageService *services.MessageService,
	channelService *services.ChannelService,
) *MessageHandlers {
	return &MessageHandlers{
		messageService: messageService,
		channelService: channelService,
	}
}

// CreateMessageRequest represents a message creation request
type CreateMessageRequest struct {
	Content          string            `json:"content"`
	Nonce            string            `json:"nonce,omitempty"`
	TTS              bool              `json:"tts"`
	Embeds           []models.Embed    `json:"embeds,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	// E2EE fields
	EncryptedContent string `json:"encrypted_content,omitempty"`
	SenderKeyID      int    `json:"sender_key_id,omitempty"`
}

// MessageReference for replies
type MessageReference struct {
	MessageID string `json:"message_id"`
	ChannelID string `json:"channel_id,omitempty"`
	ServerID  string `json:"guild_id,omitempty"`
}

// MessageResponse represents a message in API responses
type MessageResponse struct {
	ID               string                 `json:"id"`
	ChannelID        string                 `json:"channel_id"`
	ServerID         string                 `json:"guild_id,omitempty"`
	Author           *UserResponse          `json:"author"`
	Content          string                 `json:"content"`
	Timestamp        time.Time              `json:"timestamp"`
	EditedTimestamp  *time.Time             `json:"edited_timestamp,omitempty"`
	TTS              bool                   `json:"tts"`
	MentionEveryone  bool                   `json:"mention_everyone"`
	Mentions         []UserResponse         `json:"mentions"`
	MentionRoles     []string               `json:"mention_roles"`
	Attachments      []AttachmentResponse   `json:"attachments"`
	Embeds           []models.Embed         `json:"embeds"`
	Reactions        []ReactionResponse     `json:"reactions,omitempty"`
	Pinned           bool                   `json:"pinned"`
	Type             int                    `json:"type"`
	Flags            int                    `json:"flags,omitempty"`
	ReferencedMessage *MessageResponse      `json:"referenced_message,omitempty"`
	// E2EE
	EncryptedContent string `json:"encrypted_content,omitempty"`
	SenderKeyID      int    `json:"sender_key_id,omitempty"`
}

// AttachmentResponse represents an attachment in API responses
type AttachmentResponse struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
	// E2EE
	Encrypted    bool   `json:"encrypted,omitempty"`
	EncryptedKey string `json:"encrypted_key,omitempty"`
	IV           string `json:"iv,omitempty"`
}

// ReactionResponse represents a reaction summary
type ReactionResponse struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
	Me    bool   `json:"me"`
}

// GetMessages retrieves messages from a channel
func (h *MessageHandlers) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	// Parse query parameters
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var before, after, around *uuid.UUID
	if id := r.URL.Query().Get("before"); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			before = &parsed
		}
	}
	if id := r.URL.Query().Get("after"); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			after = &parsed
		}
	}
	if id := r.URL.Query().Get("around"); id != "" {
		if parsed, err := uuid.Parse(id); err == nil {
			around = &parsed
		}
	}

	messages, err := h.messageService.GetMessages(r.Context(), channelID, userID, limit, before, after, around)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := make([]MessageResponse, len(messages))
	for i, msg := range messages {
		response[i] = toMessageResponse(msg, userID)
	}

	respondJSON(w, http.StatusOK, response)
}

// GetMessage retrieves a single message
func (h *MessageHandlers) GetMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	message, err := h.messageService.GetMessage(r.Context(), messageID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toMessageResponse(message, userID))
}

// CreateMessage creates a new message
func (h *MessageHandlers) CreateMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	var req CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content == "" && req.EncryptedContent == "" && len(req.Embeds) == 0 {
		respondError(w, http.StatusBadRequest, "Message must have content, encrypted_content, or embeds")
		return
	}

	var replyTo *uuid.UUID
	if req.MessageReference != nil && req.MessageReference.MessageID != "" {
		if id, err := uuid.Parse(req.MessageReference.MessageID); err == nil {
			replyTo = &id
		}
	}

	message, err := h.messageService.CreateMessage(r.Context(), &services.CreateMessageInput{
		ChannelID:        channelID,
		AuthorID:         userID,
		Content:          req.Content,
		EncryptedContent: req.EncryptedContent,
		TTS:              req.TTS,
		Embeds:           req.Embeds,
		ReplyToID:        replyTo,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toMessageResponse(message, userID))
}

// UpdateMessage edits a message
func (h *MessageHandlers) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	var req struct {
		Content          string         `json:"content"`
		EncryptedContent string         `json:"encrypted_content,omitempty"`
		Embeds           []models.Embed `json:"embeds,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	message, err := h.messageService.UpdateMessage(r.Context(), messageID, userID, req.Content, req.Embeds)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toMessageResponse(message, userID))
}

// DeleteMessage deletes a message
func (h *MessageHandlers) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	if err := h.messageService.DeleteMessage(r.Context(), messageID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkDeleteMessages deletes multiple messages
func (h *MessageHandlers) BulkDeleteMessages(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	var req struct {
		Messages []string `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Messages) < 2 || len(req.Messages) > 100 {
		respondError(w, http.StatusBadRequest, "Must delete between 2 and 100 messages")
		return
	}

	messageIDs := make([]uuid.UUID, 0, len(req.Messages))
	for _, id := range req.Messages {
		if parsed, err := uuid.Parse(id); err == nil {
			messageIDs = append(messageIDs, parsed)
		}
	}

	if err := h.messageService.BulkDeleteMessages(r.Context(), channelID, messageIDs, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Reactions

// AddReaction adds a reaction to a message
func (h *MessageHandlers) AddReaction(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}
	emoji := chi.URLParam(r, "emoji")
	if emoji == "" {
		respondError(w, http.StatusBadRequest, "Invalid emoji")
		return
	}

	if err := h.messageService.AddReaction(r.Context(), messageID, userID, emoji); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveReaction removes a reaction from a message
func (h *MessageHandlers) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}
	emoji := chi.URLParam(r, "emoji")

	targetUserID := userID
	if targetUser := chi.URLParam(r, "userID"); targetUser != "" && targetUser != "@me" {
		if parsed, err := uuid.Parse(targetUser); err == nil {
			targetUserID = parsed
		}
	}

	if err := h.messageService.RemoveReaction(r.Context(), messageID, targetUserID, emoji, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Pins

// GetPinnedMessages gets pinned messages in a channel
func (h *MessageHandlers) GetPinnedMessages(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	messages, err := h.messageService.GetPinnedMessages(r.Context(), channelID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := make([]MessageResponse, len(messages))
	for i, msg := range messages {
		response[i] = toMessageResponse(msg, userID)
	}

	respondJSON(w, http.StatusOK, response)
}

// PinMessage pins a message
func (h *MessageHandlers) PinMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	if err := h.messageService.PinMessage(r.Context(), messageID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UnpinMessage unpins a message
func (h *MessageHandlers) UnpinMessage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid message ID")
		return
	}

	if err := h.messageService.UnpinMessage(r.Context(), messageID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Typing indicator

// StartTyping triggers typing indicator
func (h *MessageHandlers) StartTyping(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid channel ID")
		return
	}

	if err := h.messageService.StartTyping(r.Context(), channelID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper

func toMessageResponse(msg *models.Message, requesterID uuid.UUID) MessageResponse {
	resp := MessageResponse{
		ID:              msg.ID.String(),
		ChannelID:       msg.ChannelID.String(),
		Content:         msg.Content,
		Timestamp:       msg.CreatedAt,
		EditedTimestamp: msg.EditedAt,
		TTS:             msg.TTS,
		MentionEveryone: msg.MentionsEveryone,
		Mentions:        []UserResponse{},
		MentionRoles:    []string{},
		Attachments:     []AttachmentResponse{},
		Embeds:          msg.Embeds,
		Pinned:          msg.Pinned,
		Type:            int(msg.Type),
		Flags:           msg.Flags,
		EncryptedContent: msg.EncryptedContent,
	}

	if msg.ServerID != nil {
		resp.ServerID = msg.ServerID.String()
	}

	// TODO: Populate author, attachments, reactions, etc.

	return resp
}
