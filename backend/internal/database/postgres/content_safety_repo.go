package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// ContentSafetyRepository defines content safety data access operations
type ContentSafetyRepository interface {
	// Content Filters
	CreateContentFilter(ctx context.Context, filter *models.ContentFilter) error
	GetContentFilterByID(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error)
	GetContentFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error)
	GetContentFiltersByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.ContentFilter, error)
	GetContentFiltersForMessage(ctx context.Context, serverID, channelID uuid.UUID) ([]*models.ContentFilter, error)
	UpdateContentFilter(ctx context.Context, filter *models.ContentFilter) error
	DeleteContentFilter(ctx context.Context, id uuid.UUID) error
	GetEnabledFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error)

	// Age Verification
	CreateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error
	GetAgeVerificationByServerID(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error)
	GetAgeVerificationForChannel(ctx context.Context, serverID, channelID uuid.UUID) (*models.AgeVerificationSetting, error)
	UpdateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error
	DeleteAgeVerification(ctx context.Context, id uuid.UUID) error

	// User Content Preferences
	CreateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error
	GetUserContentPreference(ctx context.Context, userID uuid.UUID) (*models.UserContentPreference, error)
	UpdateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error
	UpsertUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error
}

type contentSafetyRepo struct {
	db *sql.DB
}

// NewContentSafetyRepository creates a new content safety repository
func NewContentSafetyRepository(db *sql.DB) ContentSafetyRepository {
	return &contentSafetyRepo{db: db}
}

// Content Filter methods

func (r *contentSafetyRepo) CreateContentFilter(ctx context.Context, filter *models.ContentFilter) error {
	filter.ID = uuid.New()
	filter.CreatedAt = time.Now()
	filter.UpdatedAt = time.Now()

	query := `
		INSERT INTO content_filters (id, server_id, channel_id, type, name, enabled, threshold, action, filter_data, exempt_roles, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := r.db.ExecContext(ctx, query,
		filter.ID,
		filter.ServerID,
		filter.ChannelID,
		filter.Type,
		filter.Name,
		filter.Enabled,
		filter.Threshold,
		filter.Action,
		filter.FilterData,
		filter.ExemptRoles,
		filter.CreatedBy,
		filter.CreatedAt,
		filter.UpdatedAt,
	)
	return err
}

func (r *contentSafetyRepo) GetContentFilterByID(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error) {
	query := `
		SELECT id, server_id, channel_id, type, name, enabled, threshold, action, filter_data, exempt_roles, created_by, created_at, updated_at
		FROM content_filters
		WHERE id = $1
	`

	filter := &models.ContentFilter{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&filter.ID,
		&filter.ServerID,
		&filter.ChannelID,
		&filter.Type,
		&filter.Name,
		&filter.Enabled,
		&filter.Threshold,
		&filter.Action,
		&filter.FilterData,
		&filter.ExemptRoles,
		&filter.CreatedBy,
		&filter.CreatedAt,
		&filter.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return filter, nil
}

func (r *contentSafetyRepo) GetContentFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error) {
	query := `
		SELECT id, server_id, channel_id, type, name, enabled, threshold, action, filter_data, exempt_roles, created_by, created_at, updated_at
		FROM content_filters
		WHERE server_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []*models.ContentFilter
	for rows.Next() {
		filter := &models.ContentFilter{}
		err := rows.Scan(
			&filter.ID,
			&filter.ServerID,
			&filter.ChannelID,
			&filter.Type,
			&filter.Name,
			&filter.Enabled,
			&filter.Threshold,
			&filter.Action,
			&filter.FilterData,
			&filter.ExemptRoles,
			&filter.CreatedBy,
			&filter.CreatedAt,
			&filter.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func (r *contentSafetyRepo) GetContentFiltersByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.ContentFilter, error) {
	query := `
		SELECT id, server_id, channel_id, type, name, enabled, threshold, action, filter_data, exempt_roles, created_by, created_at, updated_at
		FROM content_filters
		WHERE channel_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []*models.ContentFilter
	for rows.Next() {
		filter := &models.ContentFilter{}
		err := rows.Scan(
			&filter.ID,
			&filter.ServerID,
			&filter.ChannelID,
			&filter.Type,
			&filter.Name,
			&filter.Enabled,
			&filter.Threshold,
			&filter.Action,
			&filter.FilterData,
			&filter.ExemptRoles,
			&filter.CreatedBy,
			&filter.CreatedAt,
			&filter.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func (r *contentSafetyRepo) GetContentFiltersForMessage(ctx context.Context, serverID, channelID uuid.UUID) ([]*models.ContentFilter, error) {
	// Get both server-wide and channel-specific filters
	query := `
		SELECT id, server_id, channel_id, type, name, enabled, threshold, action, filter_data, exempt_roles, created_by, created_at, updated_at
		FROM content_filters
		WHERE server_id = $1 
		  AND (channel_id IS NULL OR channel_id = $2)
		  AND enabled = true
		ORDER BY 
			CASE WHEN channel_id IS NOT NULL THEN 0 ELSE 1 END,
			created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []*models.ContentFilter
	for rows.Next() {
		filter := &models.ContentFilter{}
		err := rows.Scan(
			&filter.ID,
			&filter.ServerID,
			&filter.ChannelID,
			&filter.Type,
			&filter.Name,
			&filter.Enabled,
			&filter.Threshold,
			&filter.Action,
			&filter.FilterData,
			&filter.ExemptRoles,
			&filter.CreatedBy,
			&filter.CreatedAt,
			&filter.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func (r *contentSafetyRepo) GetEnabledFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error) {
	query := `
		SELECT id, server_id, channel_id, type, name, enabled, threshold, action, filter_data, exempt_roles, created_by, created_at, updated_at
		FROM content_filters
		WHERE server_id = $1 AND enabled = true
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []*models.ContentFilter
	for rows.Next() {
		filter := &models.ContentFilter{}
		err := rows.Scan(
			&filter.ID,
			&filter.ServerID,
			&filter.ChannelID,
			&filter.Type,
			&filter.Name,
			&filter.Enabled,
			&filter.Threshold,
			&filter.Action,
			&filter.FilterData,
			&filter.ExemptRoles,
			&filter.CreatedBy,
			&filter.CreatedAt,
			&filter.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func (r *contentSafetyRepo) UpdateContentFilter(ctx context.Context, filter *models.ContentFilter) error {
	filter.UpdatedAt = time.Now()

	query := `
		UPDATE content_filters
		SET name = $2, enabled = $3, threshold = $4, action = $5, filter_data = $6, exempt_roles = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		filter.ID,
		filter.Name,
		filter.Enabled,
		filter.Threshold,
		filter.Action,
		filter.FilterData,
		filter.ExemptRoles,
		filter.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *contentSafetyRepo) DeleteContentFilter(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM content_filters WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Age Verification methods

func (r *contentSafetyRepo) CreateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error {
	settings.ID = uuid.New()
	settings.CreatedAt = time.Now()
	settings.UpdatedAt = time.Now()

	query := `
		INSERT INTO age_verification_settings (id, server_id, channel_id, enabled, required_age, verification_type, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.ID,
		settings.ServerID,
		settings.ChannelID,
		settings.Enabled,
		settings.RequiredAge,
		settings.VerificationType,
		settings.CreatedBy,
		settings.CreatedAt,
		settings.UpdatedAt,
	)
	return err
}

func (r *contentSafetyRepo) GetAgeVerificationByServerID(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error) {
	query := `
		SELECT id, server_id, channel_id, enabled, required_age, verification_type, created_by, created_at, updated_at
		FROM age_verification_settings
		WHERE server_id = $1 AND channel_id IS NULL
	`

	settings := &models.AgeVerificationSetting{}
	err := r.db.QueryRowContext(ctx, query, serverID).Scan(
		&settings.ID,
		&settings.ServerID,
		&settings.ChannelID,
		&settings.Enabled,
		&settings.RequiredAge,
		&settings.VerificationType,
		&settings.CreatedBy,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *contentSafetyRepo) GetAgeVerificationForChannel(ctx context.Context, serverID, channelID uuid.UUID) (*models.AgeVerificationSetting, error) {
	// First check channel-specific settings
	query := `
		SELECT id, server_id, channel_id, enabled, required_age, verification_type, created_by, created_at, updated_at
		FROM age_verification_settings
		WHERE channel_id = $1 AND server_id = $2
	`

	settings := &models.AgeVerificationSetting{}
	err := r.db.QueryRowContext(ctx, query, channelID, serverID).Scan(
		&settings.ID,
		&settings.ServerID,
		&settings.ChannelID,
		&settings.Enabled,
		&settings.RequiredAge,
		&settings.VerificationType,
		&settings.CreatedBy,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Fall back to server-wide settings
		return r.GetAgeVerificationByServerID(ctx, serverID)
	}
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *contentSafetyRepo) UpdateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error {
	settings.UpdatedAt = time.Now()

	query := `
		UPDATE age_verification_settings
		SET enabled = $2, required_age = $3, verification_type = $4, updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		settings.ID,
		settings.Enabled,
		settings.RequiredAge,
		settings.VerificationType,
		settings.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *contentSafetyRepo) DeleteAgeVerification(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM age_verification_settings WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// User Content Preference methods

func (r *contentSafetyRepo) CreateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error {
	prefs.ID = uuid.New()
	prefs.UpdatedAt = time.Now()

	query := `
		INSERT INTO user_content_preferences (id, user_id, nsfw_filter_level, hide_nsfw_content, hide_explicit_content, auto_collapse_nsfw, allow_age_verified_channels, trusted_servers, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		prefs.ID,
		prefs.UserID,
		prefs.NSFWFilterLevel,
		prefs.HideNSFWContent,
		prefs.HideExplicitContent,
		prefs.AutoCollapseNSFW,
		prefs.AllowAgeVerifiedChannels,
		prefs.TrustedServers,
		prefs.UpdatedAt,
	)
	return err
}

func (r *contentSafetyRepo) GetUserContentPreference(ctx context.Context, userID uuid.UUID) (*models.UserContentPreference, error) {
	query := `
		SELECT id, user_id, nsfw_filter_level, hide_nsfw_content, hide_explicit_content, auto_collapse_nsfw, allow_age_verified_channels, trusted_servers, updated_at
		FROM user_content_preferences
		WHERE user_id = $1
	`

	prefs := &models.UserContentPreference{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&prefs.ID,
		&prefs.UserID,
		&prefs.NSFWFilterLevel,
		&prefs.HideNSFWContent,
		&prefs.HideExplicitContent,
		&prefs.AutoCollapseNSFW,
		&prefs.AllowAgeVerifiedChannels,
		&prefs.TrustedServers,
		&prefs.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return prefs, nil
}

func (r *contentSafetyRepo) UpdateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error {
	prefs.UpdatedAt = time.Now()

	query := `
		UPDATE user_content_preferences
		SET nsfw_filter_level = $2, hide_nsfw_content = $3, hide_explicit_content = $4, auto_collapse_nsfw = $5, allow_age_verified_channels = $6, trusted_servers = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		prefs.ID,
		prefs.NSFWFilterLevel,
		prefs.HideNSFWContent,
		prefs.HideExplicitContent,
		prefs.AutoCollapseNSFW,
		prefs.AllowAgeVerifiedChannels,
		prefs.TrustedServers,
		prefs.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *contentSafetyRepo) UpsertUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error {
	prefs.UpdatedAt = time.Now()

	query := `
		INSERT INTO user_content_preferences (id, user_id, nsfw_filter_level, hide_nsfw_content, hide_explicit_content, auto_collapse_nsfw, allow_age_verified_channels, trusted_servers, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			nsfw_filter_level = EXCLUDED.nsfw_filter_level,
			hide_nsfw_content = EXCLUDED.hide_nsfw_content,
			hide_explicit_content = EXCLUDED.hide_explicit_content,
			auto_collapse_nsfw = EXCLUDED.auto_collapse_nsfw,
			allow_age_verified_channels = EXCLUDED.allow_age_verified_channels,
			trusted_servers = EXCLUDED.trusted_servers,
			updated_at = EXCLUDED.updated_at
	`

	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx, query,
		prefs.ID,
		prefs.UserID,
		prefs.NSFWFilterLevel,
		prefs.HideNSFWContent,
		prefs.HideExplicitContent,
		prefs.AutoCollapseNSFW,
		prefs.AllowAgeVerifiedChannels,
		prefs.TrustedServers,
		prefs.UpdatedAt,
	)
	return err
}
