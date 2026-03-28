package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// ForumTagRepository handles forum tag data access
type ForumTagRepository struct {
	db *sqlx.DB
}

// NewForumTagRepository creates a new forum tag repository
func NewForumTagRepository(db *sqlx.DB) *ForumTagRepository {
	return &ForumTagRepository{db: db}
}

// Create creates a new forum tag
func (r *ForumTagRepository) Create(ctx context.Context, tag *models.ForumTag) error {
	query := `
		INSERT INTO forum_tags (id, server_id, channel_id, name, emoji_name, moderated, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		tag.ID, tag.ServerID, tag.ChannelID, tag.Name, tag.EmojiName, tag.Moderated, tag.CreatedAt,
	)
	return err
}

// GetByID retrieves a tag by ID
func (r *ForumTagRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ForumTag, error) {
	var tag models.ForumTag
	query := `SELECT id, server_id, channel_id, name, emoji_name, moderated, created_at FROM forum_tags WHERE id = $1`
	err := r.db.GetContext(ctx, &tag, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tag, err
}

// GetByChannel retrieves all tags for a channel
func (r *ForumTagRepository) GetByChannel(ctx context.Context, channelID uuid.UUID) ([]models.ForumTag, error) {
	var tags []models.ForumTag
	query := `SELECT id, server_id, channel_id, name, emoji_name, moderated, created_at FROM forum_tags WHERE channel_id = $1 ORDER BY name`
	err := r.db.SelectContext(ctx, &tags, query, channelID)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// GetByIDs retrieves tags by their IDs
func (r *ForumTagRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]models.ForumTag, error) {
	if len(ids) == 0 {
		return []models.ForumTag{}, nil
	}
	var tags []models.ForumTag
	query := `SELECT id, server_id, channel_id, name, emoji_name, moderated, created_at FROM forum_tags WHERE id = ANY($1) ORDER BY name`
	err := r.db.SelectContext(ctx, &tags, query, ids)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// Update updates a forum tag
func (r *ForumTagRepository) Update(ctx context.Context, tag *models.ForumTag) error {
	query := `
		UPDATE forum_tags
		SET name = $2, emoji_name = $3, moderated = $4
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, tag.ID, tag.Name, tag.EmojiName, tag.Moderated)
	return err
}

// Delete deletes a forum tag
func (r *ForumTagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM forum_tags WHERE id = $1`, id)
	return err
}

// ApplyTags applies tags to a thread
func (r *ForumTagRepository) ApplyTags(ctx context.Context, threadID uuid.UUID, tagIDs []uuid.UUID) error {
	query := `UPDATE threads SET applied_tags = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, threadID, tagIDs)
	return err
}

// GetThreadTags retrieves tags applied to a thread
func (r *ForumTagRepository) GetThreadTags(ctx context.Context, threadID uuid.UUID) ([]models.ForumTag, error) {
	var tags []models.ForumTag
	query := `
		SELECT ft.id, ft.server_id, ft.channel_id, ft.name, ft.emoji_name, ft.moderated, ft.created_at
		FROM forum_tags ft
		JOIN threads t ON t.applied_tags @> ARRAY[ft.id]
		WHERE t.id = $1
		ORDER BY ft.name
	`
	err := r.db.SelectContext(ctx, &tags, query, threadID)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// FilterThreadsByTags filters threads by tag IDs using the applied_tags array
func (r *ForumTagRepository) FilterThreadsByTags(ctx context.Context, channelID uuid.UUID, tagIDs []uuid.UUID, sortOrder int, limit, offset int) ([]models.Thread, int, error) {
	var threads []models.Thread
	var total int

	countQuery := `SELECT COUNT(*) FROM threads WHERE parent_channel_id = $1`
	threadQuery := `
		SELECT id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count,
		       archived, auto_archive, locked, created_at, archive_timestamp,
		       applied_tags, is_pinned, pin_weight
		FROM threads
		WHERE parent_channel_id = $1
	`

	args := []interface{}{channelID}

	if len(tagIDs) > 0 {
		countQuery += ` AND applied_tags && $2`
		threadQuery += ` AND applied_tags && $2`
		args = append(args, tagIDs)
	}

	// Determine sort order
	var orderClause string
	switch sortOrder {
	case 1: // creation_date
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, created_at DESC`
	case 2: // pin_weight
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	default: // latest_activity (0)
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	}

	threadQuery += orderClause + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)

	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	err = r.db.SelectContext(ctx, &threads, threadQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return threads, total, nil
}
