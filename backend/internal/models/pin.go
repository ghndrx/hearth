package models

import "time"

type Pin struct {
    ID        string    `json:"id"`
    MessageID string    `json:"message_id"`
    ChannelID string    `json:"channel_id"`
    UserID    string    `json:"user_id"`
    PinnedAt  time.Time `json:"pinned_at"`
}

type PinnedMessage struct {
    Pin
    Message *Message `json:"message,omitempty"`
}
