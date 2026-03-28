package models

import (
	"time"

	"github.com/google/uuid"
)

// UserSettings represents a user's application settings
type UserSettings struct {
	UserID uuid.UUID `json:"user_id" db:"user_id"`

	// Theme settings
	Theme          string `json:"theme" db:"theme"`                     // "dark", "light", "system"
	MessageDisplay string `json:"message_display" db:"message_display"` // "cozy", "compact"
	CompactMode    bool   `json:"compact_mode" db:"compact_mode"`
	DeveloperMode  bool   `json:"developer_mode" db:"developer_mode"`

	// Display settings
	InlineEmbeds      bool    `json:"inline_embeds" db:"inline_embeds"`
	InlineAttachments bool    `json:"inline_attachments" db:"inline_attachments"`
	RenderReactions   bool    `json:"render_reactions" db:"render_reactions"`
	AnimateEmoji      bool    `json:"animate_emoji" db:"animate_emoji"`
	EnableTTS         bool    `json:"enable_tts" db:"enable_tts"`
	CustomCSS         *string `json:"custom_css,omitempty" db:"custom_css"`

	// Notification settings
	NotificationsEnabled        bool `json:"notifications_enabled" db:"notifications_enabled"`
	NotificationsSound          bool `json:"notifications_sound" db:"notifications_sound"`
	NotificationsDesktop        bool `json:"notifications_desktop" db:"notifications_desktop"`
	NotificationsMentionsOnly   bool `json:"notifications_mentions_only" db:"notifications_mentions_only"`
	NotificationsDM             bool `json:"notifications_dm" db:"notifications_dm"`
	NotificationsServerDefaults bool `json:"notifications_server_defaults" db:"notifications_server_defaults"`

	// Privacy settings
	PrivacyDMFromServers     bool `json:"privacy_dm_from_servers" db:"privacy_dm_from_servers"`
	PrivacyDMFromFriendsOnly bool `json:"privacy_dm_from_friends_only" db:"privacy_dm_from_friends_only"`
	PrivacyShowActivity      bool `json:"privacy_show_activity" db:"privacy_show_activity"`
	PrivacyFriendRequestsAll bool `json:"privacy_friend_requests_all" db:"privacy_friend_requests_all"`
	PrivacyReadReceipts      bool `json:"privacy_read_receipts" db:"privacy_read_receipts"`

	// Locale settings
	Locale string `json:"locale" db:"locale"` // e.g., "en-US", "es", "fr"

	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DefaultUserSettings returns the default settings for a new user
func DefaultUserSettings(userID uuid.UUID) *UserSettings {
	return &UserSettings{
		UserID: userID,

		// Theme defaults
		Theme:          "dark",
		MessageDisplay: "cozy",
		CompactMode:    false,
		DeveloperMode:  false,

		// Display defaults
		InlineEmbeds:      true,
		InlineAttachments: true,
		RenderReactions:   true,
		AnimateEmoji:      true,
		EnableTTS:         true,
		CustomCSS:         nil,

		// Notification defaults
		NotificationsEnabled:        true,
		NotificationsSound:          true,
		NotificationsDesktop:        true,
		NotificationsMentionsOnly:   false,
		NotificationsDM:             true,
		NotificationsServerDefaults: true,

		// Privacy defaults
		PrivacyDMFromServers:     true,
		PrivacyDMFromFriendsOnly: false,
		PrivacyShowActivity:      true,
		PrivacyFriendRequestsAll: true,
		PrivacyReadReceipts:      true,

		// Locale default
		Locale: "en-US",

		UpdatedAt: time.Now(),
	}
}

// UpdateUserSettingsRequest represents a request to update user settings
type UpdateUserSettingsRequest struct {
	// Theme settings
	Theme          *string `json:"theme,omitempty" validate:"omitempty,oneof=dark light system"`
	MessageDisplay *string `json:"message_display,omitempty" validate:"omitempty,oneof=cozy compact"`
	CompactMode    *bool   `json:"compact_mode,omitempty"`
	DeveloperMode  *bool   `json:"developer_mode,omitempty"`

	// Display settings
	InlineEmbeds      *bool   `json:"inline_embeds,omitempty"`
	InlineAttachments *bool   `json:"inline_attachments,omitempty"`
	RenderReactions   *bool   `json:"render_reactions,omitempty"`
	AnimateEmoji      *bool   `json:"animate_emoji,omitempty"`
	EnableTTS         *bool   `json:"enable_tts,omitempty"`
	CustomCSS         *string `json:"custom_css,omitempty"`

	// Notification settings
	NotificationsEnabled        *bool `json:"notifications_enabled,omitempty"`
	NotificationsSound          *bool `json:"notifications_sound,omitempty"`
	NotificationsDesktop        *bool `json:"notifications_desktop,omitempty"`
	NotificationsMentionsOnly   *bool `json:"notifications_mentions_only,omitempty"`
	NotificationsDM             *bool `json:"notifications_dm,omitempty"`
	NotificationsServerDefaults *bool `json:"notifications_server_defaults,omitempty"`

	// Privacy settings
	PrivacyDMFromServers     *bool `json:"privacy_dm_from_servers,omitempty"`
	PrivacyDMFromFriendsOnly *bool `json:"privacy_dm_from_friends_only,omitempty"`
	PrivacyShowActivity      *bool `json:"privacy_show_activity,omitempty"`
	PrivacyFriendRequestsAll *bool `json:"privacy_friend_requests_all,omitempty"`
	PrivacyReadReceipts      *bool `json:"privacy_read_receipts,omitempty"`

	// Locale settings
	Locale *string `json:"locale,omitempty" validate:"omitempty,min=2,max=10"`
}
