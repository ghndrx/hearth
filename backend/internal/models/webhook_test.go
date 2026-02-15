package models

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestWebhookTypes(t *testing.T) {
	if WebhookTypeIncoming != 1 {
		t.Errorf("expected WebhookTypeIncoming to be 1, got %d", WebhookTypeIncoming)
	}
	if WebhookTypeChannelFollower != 2 {
		t.Errorf("expected WebhookTypeChannelFollower to be 2, got %d", WebhookTypeChannelFollower)
	}
	if WebhookTypeApplication != 3 {
		t.Errorf("expected WebhookTypeApplication to be 3, got %d", WebhookTypeApplication)
	}
}

func TestWebhookStruct(t *testing.T) {
	serverID := uuid.New()
	creatorID := uuid.New()
	appID := uuid.New()
	avatar := "https://example.com/avatar.png"
	sourceServerID := uuid.New()
	sourceChannelID := uuid.New()

	webhook := Webhook{
		ID:              uuid.New(),
		Type:            WebhookTypeIncoming,
		ServerID:        &serverID,
		ChannelID:       uuid.New(),
		CreatorID:       &creatorID,
		Name:            "My Webhook",
		Avatar:          &avatar,
		Token:           "webhook-token-123",
		ApplicationID:   &appID,
		SourceServerID:  &sourceServerID,
		SourceChannelID: &sourceChannelID,
		URL:             "https://api.example.com/webhooks/123/token",
		CreatedAt:       time.Now(),
	}

	if webhook.Name != "My Webhook" {
		t.Errorf("expected name 'My Webhook', got %s", webhook.Name)
	}
	if webhook.Type != WebhookTypeIncoming {
		t.Errorf("expected type %d, got %d", WebhookTypeIncoming, webhook.Type)
	}
	if webhook.Token != "webhook-token-123" {
		t.Errorf("expected token 'webhook-token-123', got %s", webhook.Token)
	}
}

func TestWebhookWithRelations(t *testing.T) {
	user := &User{
		ID:       uuid.New(),
		Username: "creator",
	}

	server := &Server{
		ID:   uuid.New(),
		Name: "Test Server",
	}

	channel := &Channel{
		ID:   uuid.New(),
		Name: "announcements",
	}

	webhook := Webhook{
		ID:        uuid.New(),
		Type:      WebhookTypeIncoming,
		ChannelID: channel.ID,
		Name:      "Announcement Webhook",
		Token:     "token",
		CreatedAt: time.Now(),
		Creator:   user,
		Server:    server,
		Channel:   channel,
	}

	if webhook.Creator == nil {
		t.Error("expected non-nil creator")
	}
	if webhook.Creator.Username != "creator" {
		t.Errorf("expected creator username 'creator', got %s", webhook.Creator.Username)
	}
	if webhook.Server == nil {
		t.Error("expected non-nil server")
	}
	if webhook.Channel == nil {
		t.Error("expected non-nil channel")
	}
}

func TestWebhookMessageValidate(t *testing.T) {
	// Valid message with content
	msg := &WebhookMessage{
		Content: "Hello, world!",
	}
	if err := msg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Valid message with embeds
	msgWithEmbeds := &WebhookMessage{
		Embeds: []Embed{{Type: "rich"}},
	}
	if err := msgWithEmbeds.Validate(); err != nil {
		t.Errorf("expected no error for message with embeds, got %v", err)
	}

	// Valid message with files
	msgWithFiles := &WebhookMessage{
		Files: []interface{}{"file1"},
	}
	if err := msgWithFiles.Validate(); err != nil {
		t.Errorf("expected no error for message with files, got %v", err)
	}
}

func TestWebhookMessageValidateEmpty(t *testing.T) {
	msg := &WebhookMessage{}
	err := msg.Validate()
	if err == nil {
		t.Error("expected error for empty message")
	}
	if err != ErrEmptyMessage {
		t.Errorf("expected ErrEmptyMessage, got %v", err)
	}
}

func TestWebhookMessageValidateContentTooLong(t *testing.T) {
	msg := &WebhookMessage{
		Content: strings.Repeat("a", 2001),
	}
	err := msg.Validate()
	if err == nil {
		t.Error("expected error for too long content")
	}
	if err != ErrContentTooLong {
		t.Errorf("expected ErrContentTooLong, got %v", err)
	}
}

func TestWebhookMessageValidateTooManyEmbeds(t *testing.T) {
	embeds := make([]Embed, 11)
	for i := range embeds {
		embeds[i] = Embed{Type: "rich"}
	}

	msg := &WebhookMessage{
		Embeds: embeds,
	}
	err := msg.Validate()
	if err == nil {
		t.Error("expected error for too many embeds")
	}
	if err != ErrTooManyEmbeds {
		t.Errorf("expected ErrTooManyEmbeds, got %v", err)
	}
}

func TestWebhookMessageFull(t *testing.T) {
	msg := &WebhookMessage{
		Content:   "Test message",
		Username:  "Custom Bot",
		AvatarURL: "https://example.com/avatar.png",
		TTS:       true,
		Embeds: []Embed{
			{
				Type:        "rich",
				Title:       strPtr("Embed Title"),
				Description: strPtr("Embed description"),
			},
		},
		AllowedMentions: &struct {
			Parse       []string `json:"parse,omitempty"`
			Roles       []string `json:"roles,omitempty"`
			Users       []string `json:"users,omitempty"`
			RepliedUser bool     `json:"replied_user,omitempty"`
		}{
			Parse:       []string{"users"},
			Roles:       []string{},
			Users:       []string{"user-123"},
			RepliedUser: true,
		},
		Components: []interface{}{},
		Files:      []interface{}{},
		Flags:      4,
		ThreadName: "New Thread",
	}

	if msg.Username != "Custom Bot" {
		t.Errorf("expected username 'Custom Bot', got %s", msg.Username)
	}
	if !msg.TTS {
		t.Error("expected TTS to be true")
	}
	if msg.ThreadName != "New Thread" {
		t.Errorf("expected thread name 'New Thread', got %s", msg.ThreadName)
	}
	if err := msg.Validate(); err != nil {
		t.Errorf("expected valid message, got error: %v", err)
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("test error message")
	if err.Error() != "test error message" {
		t.Errorf("expected error message 'test error message', got %s", err.Error())
	}
}

func TestValidationErrorTypes(t *testing.T) {
	if ErrEmptyMessage.Error() != "message must have content, embeds, or files" {
		t.Error("ErrEmptyMessage has wrong message")
	}
	if ErrContentTooLong.Error() != "content must be 2000 characters or less" {
		t.Error("ErrContentTooLong has wrong message")
	}
	if ErrTooManyEmbeds.Error() != "maximum 10 embeds allowed" {
		t.Error("ErrTooManyEmbeds has wrong message")
	}
}

func TestWebhookMessageExactlyMaxEmbeds(t *testing.T) {
	// 10 embeds should be valid
	embeds := make([]Embed, 10)
	for i := range embeds {
		embeds[i] = Embed{Type: "rich"}
	}

	msg := &WebhookMessage{
		Embeds: embeds,
	}
	if err := msg.Validate(); err != nil {
		t.Errorf("10 embeds should be valid, got error: %v", err)
	}
}

func TestWebhookMessageExactlyMaxContent(t *testing.T) {
	// 2000 characters should be valid
	msg := &WebhookMessage{
		Content: strings.Repeat("a", 2000),
	}
	if err := msg.Validate(); err != nil {
		t.Errorf("2000 characters should be valid, got error: %v", err)
	}
}
