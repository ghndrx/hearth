package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// SettingsRepository handles user settings data access
type SettingsRepository struct {
	db *sqlx.DB
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *sqlx.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves user settings by user ID
func (r *SettingsRepository) Get(ctx context.Context, userID uuid.UUID) (*models.UserSettings, error) {
	var settings models.UserSettings
	query := `
		SELECT 
			user_id, theme, message_display, compact_mode, developer_mode,
			inline_embeds, inline_attachments, render_reactions, animate_emoji, enable_tts, custom_css,
			COALESCE(notifications_enabled, true) as notifications_enabled,
			COALESCE(notifications_sound, true) as notifications_sound,
			COALESCE(notifications_desktop, true) as notifications_desktop,
			COALESCE(notifications_mentions_only, false) as notifications_mentions_only,
			COALESCE(notifications_dm, true) as notifications_dm,
			COALESCE(notifications_server_defaults, true) as notifications_server_defaults,
			COALESCE(privacy_dm_from_servers, true) as privacy_dm_from_servers,
			COALESCE(privacy_dm_from_friends_only, false) as privacy_dm_from_friends_only,
			COALESCE(privacy_show_activity, true) as privacy_show_activity,
			COALESCE(privacy_friend_requests_all, true) as privacy_friend_requests_all,
			COALESCE(privacy_read_receipts, true) as privacy_read_receipts,
			COALESCE(locale, 'en-US') as locale,
			updated_at
		FROM user_settings 
		WHERE user_id = $1
	`
	err := r.db.GetContext(ctx, &settings, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &settings, err
}

// Create creates new user settings
func (r *SettingsRepository) Create(ctx context.Context, settings *models.UserSettings) error {
	query := `
		INSERT INTO user_settings (
			user_id, theme, message_display, compact_mode, developer_mode,
			inline_embeds, inline_attachments, render_reactions, animate_emoji, enable_tts, custom_css,
			notifications_enabled, notifications_sound, notifications_desktop, notifications_mentions_only,
			notifications_dm, notifications_server_defaults,
			privacy_dm_from_servers, privacy_dm_from_friends_only, privacy_show_activity,
			privacy_friend_requests_all, privacy_read_receipts, locale, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.UserID, settings.Theme, settings.MessageDisplay, settings.CompactMode, settings.DeveloperMode,
		settings.InlineEmbeds, settings.InlineAttachments, settings.RenderReactions, settings.AnimateEmoji, settings.EnableTTS, settings.CustomCSS,
		settings.NotificationsEnabled, settings.NotificationsSound, settings.NotificationsDesktop, settings.NotificationsMentionsOnly,
		settings.NotificationsDM, settings.NotificationsServerDefaults,
		settings.PrivacyDMFromServers, settings.PrivacyDMFromFriendsOnly, settings.PrivacyShowActivity,
		settings.PrivacyFriendRequestsAll, settings.PrivacyReadReceipts, settings.Locale, settings.UpdatedAt,
	)
	return err
}

// Update updates existing user settings
func (r *SettingsRepository) Update(ctx context.Context, settings *models.UserSettings) error {
	query := `
		UPDATE user_settings SET
			theme = $2, message_display = $3, compact_mode = $4, developer_mode = $5,
			inline_embeds = $6, inline_attachments = $7, render_reactions = $8, animate_emoji = $9, enable_tts = $10, custom_css = $11,
			notifications_enabled = $12, notifications_sound = $13, notifications_desktop = $14, notifications_mentions_only = $15,
			notifications_dm = $16, notifications_server_defaults = $17,
			privacy_dm_from_servers = $18, privacy_dm_from_friends_only = $19, privacy_show_activity = $20,
			privacy_friend_requests_all = $21, privacy_read_receipts = $22, locale = $23, updated_at = $24
		WHERE user_id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.UserID, settings.Theme, settings.MessageDisplay, settings.CompactMode, settings.DeveloperMode,
		settings.InlineEmbeds, settings.InlineAttachments, settings.RenderReactions, settings.AnimateEmoji, settings.EnableTTS, settings.CustomCSS,
		settings.NotificationsEnabled, settings.NotificationsSound, settings.NotificationsDesktop, settings.NotificationsMentionsOnly,
		settings.NotificationsDM, settings.NotificationsServerDefaults,
		settings.PrivacyDMFromServers, settings.PrivacyDMFromFriendsOnly, settings.PrivacyShowActivity,
		settings.PrivacyFriendRequestsAll, settings.PrivacyReadReceipts, settings.Locale, settings.UpdatedAt,
	)
	return err
}

// Upsert creates or updates user settings
func (r *SettingsRepository) Upsert(ctx context.Context, settings *models.UserSettings) error {
	settings.UpdatedAt = time.Now()
	query := `
		INSERT INTO user_settings (
			user_id, theme, message_display, compact_mode, developer_mode,
			inline_embeds, inline_attachments, render_reactions, animate_emoji, enable_tts, custom_css,
			notifications_enabled, notifications_sound, notifications_desktop, notifications_mentions_only,
			notifications_dm, notifications_server_defaults,
			privacy_dm_from_servers, privacy_dm_from_friends_only, privacy_show_activity,
			privacy_friend_requests_all, privacy_read_receipts, locale, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
		ON CONFLICT (user_id) DO UPDATE SET
			theme = EXCLUDED.theme,
			message_display = EXCLUDED.message_display,
			compact_mode = EXCLUDED.compact_mode,
			developer_mode = EXCLUDED.developer_mode,
			inline_embeds = EXCLUDED.inline_embeds,
			inline_attachments = EXCLUDED.inline_attachments,
			render_reactions = EXCLUDED.render_reactions,
			animate_emoji = EXCLUDED.animate_emoji,
			enable_tts = EXCLUDED.enable_tts,
			custom_css = EXCLUDED.custom_css,
			notifications_enabled = EXCLUDED.notifications_enabled,
			notifications_sound = EXCLUDED.notifications_sound,
			notifications_desktop = EXCLUDED.notifications_desktop,
			notifications_mentions_only = EXCLUDED.notifications_mentions_only,
			notifications_dm = EXCLUDED.notifications_dm,
			notifications_server_defaults = EXCLUDED.notifications_server_defaults,
			privacy_dm_from_servers = EXCLUDED.privacy_dm_from_servers,
			privacy_dm_from_friends_only = EXCLUDED.privacy_dm_from_friends_only,
			privacy_show_activity = EXCLUDED.privacy_show_activity,
			privacy_friend_requests_all = EXCLUDED.privacy_friend_requests_all,
			privacy_read_receipts = EXCLUDED.privacy_read_receipts,
			locale = EXCLUDED.locale,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.UserID, settings.Theme, settings.MessageDisplay, settings.CompactMode, settings.DeveloperMode,
		settings.InlineEmbeds, settings.InlineAttachments, settings.RenderReactions, settings.AnimateEmoji, settings.EnableTTS, settings.CustomCSS,
		settings.NotificationsEnabled, settings.NotificationsSound, settings.NotificationsDesktop, settings.NotificationsMentionsOnly,
		settings.NotificationsDM, settings.NotificationsServerDefaults,
		settings.PrivacyDMFromServers, settings.PrivacyDMFromFriendsOnly, settings.PrivacyShowActivity,
		settings.PrivacyFriendRequestsAll, settings.PrivacyReadReceipts, settings.Locale, settings.UpdatedAt,
	)
	return err
}

// Delete deletes user settings
func (r *SettingsRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_settings WHERE user_id = $1`, userID)
	return err
}
