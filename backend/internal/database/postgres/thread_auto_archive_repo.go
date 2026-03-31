package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// ThreadAutoArchiveRepository handles database operations for thread auto-archive
type ThreadAutoArchiveRepository struct {
	db *sqlx.DB
}

// NewThreadAutoArchiveRepository creates a new thread auto-archive repository
func NewThreadAutoArchiveRepository(db *sqlx.DB) *ThreadAutoArchiveRepository {
	return &ThreadAutoArchiveRepository{db: db}
}

// CreateServerSettings creates server-level auto-archive settings
func (r *ThreadAutoArchiveRepository) CreateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	query := `
		INSERT INTO thread_auto_archive_settings (id, server_id, default_duration, allow_override, archive_duration_options, require_post_author, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.ID, settings.ServerID, settings.DefaultDuration, settings.AllowOverride,
		settings.ArchiveDurationOptions, settings.RequirePostAuthor, settings.CreatedAt, settings.UpdatedAt,
	)
	return err
}

// GetServerSettings retrieves auto-archive settings for a server
func (r *ThreadAutoArchiveRepository) GetServerSettings(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveSettings, error) {
	var settings models.ThreadAutoArchiveSettings
	query := `
		SELECT id, server_id, default_duration, allow_override, archive_duration_options, require_post_author, created_at, updated_at 
		FROM thread_auto_archive_settings WHERE server_id = $1
	`
	err := r.db.GetContext(ctx, &settings, query, serverID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateServerSettings updates server-level auto-archive settings
func (r *ThreadAutoArchiveRepository) UpdateServerSettings(ctx context.Context, settings *models.ThreadAutoArchiveSettings) error {
	query := `
		UPDATE thread_auto_archive_settings SET
			default_duration = $2, allow_override = $3, archive_duration_options = $4,
			require_post_author = $5, updated_at = $6
		WHERE server_id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.ServerID, settings.DefaultDuration, settings.AllowOverride,
		settings.ArchiveDurationOptions, settings.RequirePostAuthor, time.Now(),
	)
	return err
}

// DeleteServerSettings deletes server-level auto-archive settings
func (r *ThreadAutoArchiveRepository) DeleteServerSettings(ctx context.Context, serverID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM thread_auto_archive_settings WHERE server_id = $1`, serverID)
	return err
}

// SetChannelOverride creates or updates channel-level auto-archive override
func (r *ThreadAutoArchiveRepository) SetChannelOverride(ctx context.Context, override *models.ChannelAutoArchiveOverride) error {
	query := `
		INSERT INTO channel_auto_archive_override (id, channel_id, auto_archive_duration, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (channel_id) DO UPDATE SET
			auto_archive_duration = $3, updated_at = $5
	`
	_, err := r.db.ExecContext(ctx, query,
		override.ID, override.ChannelID, override.AutoArchiveDuration, override.CreatedAt, override.UpdatedAt,
	)
	return err
}

// GetChannelOverride retrieves channel-level auto-archive override
func (r *ThreadAutoArchiveRepository) GetChannelOverride(ctx context.Context, channelID uuid.UUID) (*models.ChannelAutoArchiveOverride, error) {
	var override models.ChannelAutoArchiveOverride
	query := `
		SELECT id, channel_id, auto_archive_duration, created_at, updated_at 
		FROM channel_auto_archive_override WHERE channel_id = $1
	`
	err := r.db.GetContext(ctx, &override, query, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &override, nil
}

// DeleteChannelOverride deletes channel-level auto-archive override
func (r *ThreadAutoArchiveRepository) DeleteChannelOverride(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM channel_auto_archive_override WHERE channel_id = $1`, channelID)
	return err
}

// GetOrCreateThreadMeta retrieves or creates auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) GetOrCreateThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	var meta models.ThreadAutoArchiveMeta
	
	// First try to get existing
	query := `
		SELECT thread_id, last_activity_at, last_activity_message_id, last_activity_user_id, 
		       next_archive_at, archive_eligible, bumped_by_owner, created_at, updated_at 
		FROM thread_auto_archive_meta WHERE thread_id = $1
	`
	err := r.db.GetContext(ctx, &meta, query, threadID)
	if err == sql.ErrNoRows {
		// Create new
		now := time.Now()
		insertQuery := `
			INSERT INTO thread_auto_archive_meta (thread_id, last_activity_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
			RETURNING thread_id, last_activity_at, last_activity_message_id, last_activity_user_id, 
			          next_archive_at, archive_eligible, bumped_by_owner, created_at, updated_at
		`
		err = r.db.GetContext(ctx, &meta, insertQuery, threadID, now, now, now)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	
	return &meta, nil
}

// UpdateThreadMeta updates auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) UpdateThreadMeta(ctx context.Context, meta *models.ThreadAutoArchiveMeta) error {
	query := `
		UPDATE thread_auto_archive_meta SET
			last_activity_at = $2, last_activity_message_id = $3, last_activity_user_id = $4,
			next_archive_at = $5, archive_eligible = $6, bumped_by_owner = $7, updated_at = $8
		WHERE thread_id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		meta.ThreadID, meta.LastActivityAt, meta.LastActivityMessageID, meta.LastActivityUserID,
		meta.NextArchiveAt, meta.ArchiveEligible, meta.BumpedByOwner, time.Now(),
	)
	return err
}

// SetThreadNextArchive sets the next archive time for a thread
func (r *ThreadAutoArchiveRepository) SetThreadNextArchive(ctx context.Context, threadID uuid.UUID, nextArchiveAt *time.Time) error {
	query := `UPDATE thread_auto_archive_meta SET next_archive_at = $2, updated_at = $3 WHERE thread_id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, nextArchiveAt, time.Now())
	return err
}

// SetThreadArchiveEligible sets the archive eligibility for a thread
func (r *ThreadAutoArchiveRepository) SetThreadArchiveEligible(ctx context.Context, threadID uuid.UUID, eligible bool) error {
	query := `UPDATE thread_auto_archive_meta SET archive_eligible = $2, updated_at = $3 WHERE thread_id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, eligible, time.Now())
	return err
}

// BumpThreadOwnerActivity marks that the owner has posted in the thread
func (r *ThreadAutoArchiveRepository) BumpThreadOwnerActivity(ctx context.Context, threadID uuid.UUID) error {
	query := `UPDATE thread_auto_archive_meta SET bumped_by_owner = TRUE, updated_at = $2 WHERE thread_id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, time.Now())
	return err
}

// GetThreadsReadyForArchive retrieves threads ready to be archived
func (r *ThreadAutoArchiveRepository) GetThreadsReadyForArchive(ctx context.Context, limit int) ([]*models.ThreadAutoArchiveMeta, error) {
	var metas []*models.ThreadAutoArchiveMeta
	query := `
		SELECT tam.thread_id, tam.last_activity_at, tam.last_activity_message_id, tam.last_activity_user_id, 
		       tam.next_archive_at, tam.archive_eligible, tam.bumped_by_owner, tam.created_at, tam.updated_at
		FROM thread_auto_archive_meta tam
		JOIN threads t ON t.id = tam.thread_id
		WHERE tam.next_archive_at IS NOT NULL 
		  AND tam.next_archive_at <= NOW() 
		  AND tam.archive_eligible = TRUE
		  AND t.archived = FALSE
		ORDER BY tam.next_archive_at ASC
		LIMIT $1
	`
	err := r.db.SelectContext(ctx, &metas, query, limit)
	return metas, err
}

// GetThreadMeta retrieves auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) GetThreadMeta(ctx context.Context, threadID uuid.UUID) (*models.ThreadAutoArchiveMeta, error) {
	var meta models.ThreadAutoArchiveMeta
	query := `
		SELECT thread_id, last_activity_at, last_activity_message_id, last_activity_user_id, 
		       next_archive_at, archive_eligible, bumped_by_owner, created_at, updated_at 
		FROM thread_auto_archive_meta WHERE thread_id = $1
	`
	err := r.db.GetContext(ctx, &meta, query, threadID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// DeleteThreadMeta deletes auto-archive metadata for a thread
func (r *ThreadAutoArchiveRepository) DeleteThreadMeta(ctx context.Context, threadID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM thread_auto_archive_meta WHERE thread_id = $1`, threadID)
	return err
}

// GetServerStats retrieves auto-archive statistics for a server
func (r *ThreadAutoArchiveRepository) GetServerStats(ctx context.Context, serverID uuid.UUID) (*models.ThreadAutoArchiveStats, error) {
	var stats models.ThreadAutoArchiveStats
	stats.ServerID = serverID
	
	// Get total threads in server channels
	query := `
		SELECT COUNT(*) FROM threads t
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1
	`
	err := r.db.GetContext(ctx, &stats.TotalThreads, query, serverID)
	if err != nil {
		return nil, err
	}
	
	// Get archived threads
	query = `
		SELECT COUNT(*) FROM threads t
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1 AND t.archived = TRUE
	`
	err = r.db.GetContext(ctx, &stats.ArchivedThreads, query, serverID)
	if err != nil {
		return nil, err
	}
	
	// Get scheduled threads (have next_archive_at set and not archived)
	query = `
		SELECT COUNT(*) FROM thread_auto_archive_meta tam
		JOIN threads t ON t.id = tam.thread_id
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1 AND tam.next_archive_at IS NOT NULL AND t.archived = FALSE
	`
	err = r.db.GetContext(ctx, &stats.ScheduledThreads, query, serverID)
	if err != nil {
		return nil, err
	}
	
	// Get threads ready to archive
	query = `
		SELECT COUNT(*) FROM thread_auto_archive_meta tam
		JOIN threads t ON t.id = tam.thread_id
		JOIN channels c ON c.id = t.parent_channel_id
		WHERE c.server_id = $1 AND tam.next_archive_at IS NOT NULL 
		  AND tam.next_archive_at <= NOW() AND tam.archive_eligible = TRUE AND t.archived = FALSE
	`
	err = r.db.GetContext(ctx, &stats.ReadyToArchiveThreads, query, serverID)
	if err != nil {
		return nil, err
	}
	
	return &stats, nil
}

// GetChannelDuration gets the effective auto-archive duration for a channel
func (r *ThreadAutoArchiveRepository) GetChannelDuration(ctx context.Context, channelID, serverID uuid.UUID) (int, error) {
	// First check for channel override
	var overrideDuration int
	query := `SELECT auto_archive_duration FROM channel_auto_archive_override WHERE channel_id = $1`
	err := r.db.GetContext(ctx, &overrideDuration, query, channelID)
	if err == nil {
		return overrideDuration, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	
	// Fall back to server default
	var settings models.ThreadAutoArchiveSettings
	query = `SELECT default_duration FROM thread_auto_archive_settings WHERE server_id = $1`
	err = r.db.GetContext(ctx, &settings, query, serverID)
	if err == sql.ErrNoRows {
		return 1440, nil // Default 24 hours
	}
	if err != nil {
		return 0, err
	}
	
	return settings.DefaultDuration, nil
}
