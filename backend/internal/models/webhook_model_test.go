package models

import (
	"testing"
)

func TestWebhookMessageValidate(t *testing.T) {
	oneEmbed := []Embed{{}}
	elevenEmbeds := make([]Embed, 11)
	tenEmbeds := make([]Embed, 10)

	tests := []struct {
		name      string
		msg       *WebhookMessage
		expectErr error
	}{
		{
			name:      "empty message",
			msg:       &WebhookMessage{},
			expectErr: ErrEmptyMessage,
		},
		{
			name: "valid content",
			msg: &WebhookMessage{
				Content: "Hello, world!",
			},
			expectErr: nil,
		},
		{
			name: "valid with embeds",
			msg: &WebhookMessage{
				Embeds: []Embed{{}, {}},
			},
			expectErr: nil,
		},
		{
			name: "valid with files",
			msg: &WebhookMessage{
				Files: []interface{}{"file1"},
			},
			expectErr: nil,
		},
		{
			name: "content too long",
			msg: &WebhookMessage{
				Content: string(make([]byte, 2001)),
			},
			expectErr: ErrContentTooLong,
		},
		{
			name: "content exactly 2000",
			msg: &WebhookMessage{
				Content: string(make([]byte, 2000)),
			},
			expectErr: nil,
		},
		{
			name: "too many embeds",
			msg: &WebhookMessage{
				Embeds: elevenEmbeds,
			},
			expectErr: ErrTooManyEmbeds,
		},
		{
			name: "exactly 10 embeds",
			msg: &WebhookMessage{
				Embeds: tenEmbeds,
			},
			expectErr: nil,
		},
		{
			name: "content and embeds together",
			msg: &WebhookMessage{
				Content: "Hello",
				Embeds:  oneEmbed,
			},
			expectErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if err != tt.expectErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.expectErr)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("test error message")

	if err.Error() != "test error message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test error message")
	}
}

func TestValidationErrorUnique(t *testing.T) {
	err1 := NewValidationError("error 1")
	err2 := NewValidationError("error 2")

	if err1 == err2 {
		t.Error("expected different ValidationError instances")
	}
	if err1.Error() == err2.Error() {
		t.Error("expected different error messages")
	}
}
