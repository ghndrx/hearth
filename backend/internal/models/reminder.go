package models

import "time"

type Reminder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	MessageID string    `json:"message_id"`
	ChannelID string    `json:"channel_id"`
	RemindAt  time.Time `json:"remind_at"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateReminderRequest struct {
	MessageID string `json:"message_id"`
	ChannelID string `json:"channel_id"`
	RemindAt  string `json:"remind_at"` // RFC3339 timestamp
}
