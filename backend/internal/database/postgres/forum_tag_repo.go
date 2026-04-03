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
		INSERT INTO forum_tags (id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		tag.ID, tag.ServerID, tag.ChannelID, tag.Name, tag.Color, tag.EmojiName, tag.Moderated, tag.Position, tag.CreatedAt,
	)
	return err
}

// GetByID retrieves a tag by ID
func (r *ForumTagRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ForumTag, error) {
	var tag models.ForumTag
	query := `SELECT id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at FROM forum_tags WHERE id = $1`
	err := r.db.GetContext(ctx, &tag, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tag, err
}

// GetByChannel retrieves all tags for a channel
func (r *ForumTagRepository) GetByChannel(ctx context.Context, channelID uuid.UUID) ([]models.ForumTag, error) {
	var tags []models.ForumTag
	query := `SELECT id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at FROM forum_tags WHERE channel_id = $1 ORDER BY position, name`
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
	query := `SELECT id, server_id, channel_id, name, color, emoji_name, moderated, position, created_at FROM forum_tags WHERE id = ANY($1) ORDER BY position, name`
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
		SET name = $2, color = $3, emoji_name = $4, moderated = $5, position = $6
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, tag.ID, tag.Name, tag.Color, tag.EmojiName, tag.Moderated, tag.Position)
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
		SELECT ft.id, ft.server_id, ft.channel_id, ft.name, ft.color, ft.emoji_name, ft.moderated, ft.position, ft.created_at
		FROM forum_tags ft
		JOIN threads t ON t.applied_tags @> ARRAY[ft.id]
		WHERE t.id = $1
		ORDER BY ft.position, ft.name
	`
	err := r.db.SelectContext(ctx, &tags, query, threadID)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// FilterThreads filters threads with full ForumPostFilter support
func (r *ForumTagRepository) FilterThreads(ctx context.Context, channelID uuid.UUID, filter *models.ForumPostFilter, limit, offset int) ([]models.Thread, int, error) {
	var threads []models.Thread
	var total int

	countQuery := `SELECT COUNT(*) FROM threads WHERE parent_channel_id = $1`
	threadQuery := `
		SELECT id, parent_channel_id, parent_message_id, owner_id, name, message_count, member_count,
		       archived, auto_archive, locked, created_at, archive_timestamp,
		       applied_tags, is_pinned, pin_weight, is_solved, solved_by, solved_at, solved_message_id
		FROM threads
		WHERE parent_channel_id = $1
	`

	args := []interface{}{channelID}
	argIndex := 2

	// Filter by tags
	if len(filter.TagIDs) > 0 {
		countQuery += fmt.Sprintf(` AND applied_tags && $%d`, argIndex)
		threadQuery += fmt.Sprintf(` AND applied_tags && $%d`, argIndex)
		args = append(args, filter.TagIDs)
		argIndex++
	}

	// Filter by author
	if filter.AuthorID != nil {
		countQuery += fmt.Sprintf(` AND owner_id = $%d`, argIndex)
		threadQuery += fmt.Sprintf(` AND owner_id = $%d`, argIndex)
		args = append(args, *filter.AuthorID)
		argIndex++
	}

	// Filter by pinned only
	if filter.PinnedOnly {
		countQuery += ` AND is_pinned = TRUE`
		threadQuery += ` AND is_pinned = TRUE`
	}

	// Filter by search query (searches in name)
	if filter.SearchQuery != "" {
		searchPattern := "%" + filter.SearchQuery + "%"
		countQuery += fmt.Sprintf(` AND name ILIKE $%d`, argIndex)
		threadQuery += fmt.Sprintf(` AND name ILIKE $%d`, argIndex)
		args = append(args, searchPattern)
		argIndex++
	}

	// Determine sort order
	var orderClause string
	switch filter.SortOrder {
	case 1: // creation_date
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, created_at DESC`
	case 2: // pin_weight
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	case 3: // most_reactions (would need join with reactions table - for now use message_count as proxy)
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, message_count DESC, created_at DESC`
	case 4: // solved_first
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, is_solved DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	default: // latest_activity (0)
		orderClause = ` ORDER BY is_pinned DESC, pin_weight DESC, archive_timestamp DESC NULLS LAST, created_at DESC`
	}

	threadQuery += orderClause + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)

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
