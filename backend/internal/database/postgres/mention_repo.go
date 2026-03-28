package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// MentionRepository handles mention database operations
type MentionRepository struct {
	db *sqlx.DB
}

// NewMentionRepository creates a new mention repository
func NewMentionRepository(db *sqlx.DB) *MentionRepository {
	return &MentionRepository{db: db}
}

// Create creates a new mention record
func (r *MentionRepository) Create(ctx context.Context, mention *models.Mention) error {
	if mention.ID == uuid.Nil {
		mention.ID = uuid.New()
	}
	if mention.CreatedAt.IsZero() {
		mention.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO mentions (
			id, user_id, message_id, mentioned_by, channel_id, guild_id, 
			mention_type, mentioned_role_id, mentioned_channel_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query,
		mention.ID,
		mention.UserID,
		mention.MessageID,
		mention.MentionedBy,
		mention.ChannelID,
		mention.GuildID,
		mention.MentionType,
		mention.MentionedRoleID,
		mention.MentionedChannelID,
		mention.CreatedAt,
	)
	return err
}

// CreateBatch creates multiple mention records efficiently
func (r *MentionRepository) CreateBatch(ctx context.Context, mentions []*models.Mention) error {
	if len(mentions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO mentions (
			id, user_id, message_id, mentioned_by, channel_id, guild_id, 
			mention_type, mentioned_role_id, mentioned_channel_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING
	`

	for _, m := range mentions {
		if m.ID == uuid.Nil {
			m.ID = uuid.New()
		}
		if m.CreatedAt.IsZero() {
			m.CreatedAt = time.Now()
		}

		_, err := tx.ExecContext(ctx, query,
			m.ID,
			m.UserID,
			m.MessageID,
			m.MentionedBy,
			m.ChannelID,
			m.GuildID,
			m.MentionType,
			m.MentionedRoleID,
			m.MentionedChannelID,
			m.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByID retrieves a mention by ID
func (r *MentionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Mention, error) {
	var mention models.Mention
	query := `
		SELECT id, user_id, message_id, mentioned_by, channel_id, guild_id,
			   mention_type, mentioned_role_id, mentioned_channel_id, read_at, created_at
		FROM mentions
		WHERE id = $1
	`
	err := r.db.GetContext(ctx, &mention, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &mention, nil
}

// GetMentionsWithContext retrieves mentions with full context for API responses
func (r *MentionRepository) GetMentionsWithContext(ctx context.Context, filter *models.MentionFilter) ([]models.MentionWithContext, int, error) {
	filter.SetDefaults()

	// Build the base query
	baseQuery := `
		SELECT 
			m.id, m.user_id, m.message_id, m.mentioned_by, m.channel_id, m.guild_id,
			m.mention_type, m.mentioned_role_id, m.mentioned_channel_id, m.read_at, m.created_at,
			s.name as server_name,
			c.name as channel_name,
			u.username as author_name,
			u.avatar as author_avatar,
			COALESCE(SUBSTRING(msg.content, 1, 200), '') as preview
		FROM mentions m
		LEFT JOIN servers s ON m.guild_id = s.id
		LEFT JOIN channels c ON m.channel_id = c.id
		LEFT JOIN users u ON m.mentioned_by = u.id
		LEFT JOIN messages msg ON m.message_id = msg.id
		WHERE m.user_id = $1
	`

	countQuery := `SELECT COUNT(*) FROM mentions m WHERE m.user_id = $1`

	var conditions []string
	args := []interface{}{filter.UserID}
	argIndex := 2

	if filter.Unread != nil {
		if *filter.Unread {
			conditions = append(conditions, "m.read_at IS NULL")
		} else {
			conditions = append(conditions, "m.read_at IS NOT NULL")
		}
	}

	if filter.MentionType != nil {
		conditions = append(conditions, fmt.Sprintf("m.mention_type = $%d", argIndex))
		args = append(args, *filter.MentionType)
		argIndex++
	}

	if filter.ChannelID != nil {
		conditions = append(conditions, fmt.Sprintf("m.channel_id = $%d", argIndex))
		args = append(args, *filter.ChannelID)
		argIndex++
	}

	if filter.GuildID != nil {
		conditions = append(conditions, fmt.Sprintf("m.guild_id = $%d", argIndex))
		args = append(args, *filter.GuildID)
		argIndex++
	}

	if filter.Before != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at < $%d", argIndex))
		args = append(args, *filter.Before)
		argIndex++
	}

	if filter.After != nil {
		conditions = append(conditions, fmt.Sprintf("m.created_at > $%d", argIndex))
		args = append(args, *filter.After)
		argIndex++
	}

	// Build full query
	if len(conditions) > 0 {
		condStr := " AND " + strings.Join(conditions, " AND ")
		baseQuery += condStr
		countQuery += condStr
	}

	baseQuery += " ORDER BY m.created_at DESC"
	baseQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit+1, filter.Offset) // +1 to check for more

	// Execute queries
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	var mentions []models.MentionWithContext
	if err := r.db.SelectContext(ctx, &mentions, baseQuery, args...); err != nil {
		return nil, 0, err
	}

	return mentions, total, nil
}

// GetUnreadCount returns the count of unread mentions for a user
func (r *MentionRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM mentions WHERE user_id = $1 AND read_at IS NULL`
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// GetStats returns mention statistics for a user
func (r *MentionRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.MentionStats, error) {
	stats := &models.MentionStats{}

	// Total count
	err := r.db.GetContext(ctx, &stats.TotalCount,
		`SELECT COUNT(*) FROM mentions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}

	// Unread count
	err = r.db.GetContext(ctx, &stats.UnreadCount,
		`SELECT COUNT(*) FROM mentions WHERE user_id = $1 AND read_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}

	// Today count
	err = r.db.GetContext(ctx, &stats.TodayCount,
		`SELECT COUNT(*) FROM mentions WHERE user_id = $1 AND created_at >= CURRENT_DATE`, userID)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// MarkAsRead marks a mention as read
func (r *MentionRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE mentions SET read_at = $1 WHERE id = $2 AND user_id = $3 AND read_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id, userID)
	return err
}

// MarkAllAsRead marks all mentions as read for a user
func (r *MentionRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `UPDATE mentions SET read_at = $1 WHERE user_id = $2 AND read_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// MarkChannelMentionsAsRead marks all mentions in a channel as read for a user
func (r *MentionRepository) MarkChannelMentionsAsRead(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	query := `UPDATE mentions SET read_at = $1 WHERE user_id = $2 AND channel_id = $3 AND read_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now(), userID, channelID)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// DeleteByMessage deletes all mentions for a message (when message is deleted)
func (r *MentionRepository) DeleteByMessage(ctx context.Context, messageID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mentions WHERE message_id = $1`, messageID)
	return err
}

// DeleteByChannel deletes all mentions in a channel (when channel is deleted)
func (r *MentionRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mentions WHERE channel_id = $1`, channelID)
	return err
}

// DeleteByUser deletes all mentions for a user (when user is deleted)
func (r *MentionRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM mentions WHERE user_id = $1 OR mentioned_by = $1`, userID)
	return err
}

// Search searches mentions by content or author
func (r *MentionRepository) Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]models.MentionWithContext, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	searchQuery := `
		SELECT 
			m.id, m.user_id, m.message_id, m.mentioned_by, m.channel_id, m.guild_id,
			m.mention_type, m.mentioned_role_id, m.mentioned_channel_id, m.read_at, m.created_at,
			s.name as server_name,
			c.name as channel_name,
			u.username as author_name,
			u.avatar as author_avatar,
			COALESCE(SUBSTRING(msg.content, 1, 200), '') as preview
		FROM mentions m
		LEFT JOIN servers s ON m.guild_id = s.id
		LEFT JOIN channels c ON m.channel_id = c.id
		LEFT JOIN users u ON m.mentioned_by = u.id
		LEFT JOIN messages msg ON m.message_id = msg.id
		WHERE m.user_id = $1
		  AND (
			  msg.content ILIKE '%' || $2 || '%'
			  OR u.username ILIKE '%' || $2 || '%'
			  OR c.name ILIKE '%' || $2 || '%'
		  )
		ORDER BY m.created_at DESC
		LIMIT $3
	`

	var mentions []models.MentionWithContext
	err := r.db.SelectContext(ctx, &mentions, searchQuery, userID, query, limit)
	return mentions, err
}
