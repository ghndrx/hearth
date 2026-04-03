package models

import (
	"time"

	"github.com/google/uuid"
)

// ForwardedMessage represents a message that has been forwarded to another channel
type ForwardedMessage struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	OriginalMessageID    uuid.UUID `json:"original_message_id" db:"original_message_id"`
	ForwardedByID        uuid.UUID `json:"forwarded_by_id" db:"forwarded_by_id"`
	DestinationChannelID uuid.UUID `json:"destination_channel_id" db:"destination_channel_id"`
	Comment              string    `json:"comment,omitempty" db:"comment"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

// ForwardMessageRequest represents a request to forward a message
type ForwardMessageRequest struct {
	DestinationChannelID uuid.UUID `json:"destination_channel_id" validate:"required"`
	Comment              string    `json:"comment,omitempty"`
}

// ForwardedMessageWithContext includes the original message context
type ForwardedMessageWithContext struct {
	ForwardedMessage
	OriginalMessage *Message `json:"original_message,omitempty"`
}
