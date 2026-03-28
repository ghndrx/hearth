package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// ReadStateRepository handles read state database operations
type ReadStateRepository struct {
	db *sqlx.DB
}

// NewReadStateRepository creates a new read state repository
func NewReadStateRepository(db *sqlx.DB) *ReadStateRepository {
	return &ReadStateRepository{db: db}
}

// GetReadState gets the read state for a user in a channel
func (r *ReadStateRepository) GetReadState(ctx context.Context, userID, channelID uuid.UUID) (*models.ReadState, error) {
	var state models.ReadState
	query := `SELECT user_id, channel_id, last_message_id, mention_count, updated_at FROM read_states WHERE user_id = $1 AND channel_id = $2`
	err := r.db.GetContext(ctx, &state, query, userID, channelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// SetReadState sets or updates the read state for a user in a channel
func (r *ReadStateRepository) SetReadState(ctx context.Context, userID, channelID uuid.UUID, lastMessageID *uuid.UUID) error {
	query := `
		INSERT INTO read_states (user_id, channel_id, last_message_id, mention_count, updated_at)
		VALUES ($1, $2, $3, 0, $4)
		ON CONFLICT (user_id, channel_id) DO UPDATE SET
			last_message_id = COALESCE($3, read_states.last_message_id),
			mention_count = 0,
			updated_at = $4
	`
	_, err := r.db.ExecContext(ctx, query, userID, channelID, lastMessageID, time.Now())
	return err
}

// IncrementMentionCount increments the mention count for a user in a channel
func (r *ReadStateRepository) IncrementMentionCount(ctx context.Context, userID, channelID uuid.UUID) error {
	query := `
		INSERT INTO read_states (user_id, channel_id, mention_count, updated_at)
		VALUES ($1, $2, 1, $3)
		ON CONFLICT (user_id, channel_id) DO UPDATE SET
			mention_count = read_states.mention_count + 1,
			updated_at = $3
	`
	_, err := r.db.ExecContext(ctx, query, userID, channelID, time.Now())
	return err
}

// GetUnreadCount returns the count of unread messages for a user in a channel
func (r *ReadStateRepository) GetUnreadCount(ctx context.Context, userID, channelID uuid.UUID) (int, error) {
	// Get the read state first
	state, err := r.GetReadState(ctx, userID, channelID)
	if err != nil {
		return 0, err
	}

	var count int
	if state == nil || state.LastMessageID == nil {
		// No read state - count all messages
		query := `SELECT COUNT(*) FROM messages WHERE channel_id = $1`
		err = r.db.GetContext(ctx, &count, query, channelID)
	} else {
		// Count messages after the last read message
		query := `SELECT COUNT(*) FROM messages WHERE channel_id = $1 AND created_at > (SELECT created_at FROM messages WHERE id = $2)`
		err = r.db.GetContext(ctx, &count, query, channelID, state.LastMessageID)
	}

	return count, err
}

// GetUnreadInfo gets the unread information for a user in a channel
func (r *ReadStateRepository) GetUnreadInfo(ctx context.Context, userID, channelID uuid.UUID) (*models.ChannelUnreadInfo, error) {
	// Get the channel's last message
	var lastMsgID *uuid.UUID
	err := r.db.GetContext(ctx, &lastMsgID, `SELECT last_message_id FROM channels WHERE id = $1`, channelID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get read state
	state, err := r.GetReadState(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}

	info := &models.ChannelUnreadInfo{
		ChannelID:     channelID,
		LastMessageID: lastMsgID,
		MentionCount:  0,
	}

	if state != nil {
		info.MentionCount = state.MentionCount
	}

	// Calculate unread
	if lastMsgID == nil {
		info.UnreadCount = 0
		info.HasUnread = false
	} else if state == nil || state.LastMessageID == nil {
		// Never read - count all messages
		var count int
		err = r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM messages WHERE channel_id = $1`, channelID)
		if err != nil {
			return nil, err
		}
		info.UnreadCount = count
		info.HasUnread = count > 0
	} else {
		// Check if the channel's last message is newer than what we've read
		if *lastMsgID != *state.LastMessageID {
			var count int
			err = r.db.GetContext(ctx, &count, `
				SELECT COUNT(*) FROM messages 
				WHERE channel_id = $1 
				AND created_at > (SELECT created_at FROM messages WHERE id = $2)
			`, channelID, state.LastMessageID)
			if err != nil {
				return nil, err
			}
			info.UnreadCount = count
			info.HasUnread = count > 0
		} else {
			info.UnreadCount = 0
			info.HasUnread = false
		}
	}

	return info, nil
}

// GetUserUnreadSummary gets unread information for all channels a user has access to
func (r *ReadStateRepository) GetUserUnreadSummary(ctx context.Context, userID uuid.UUID) (*models.UnreadSummary, error) {
	// Get all channels the user has access to (server channels + DMs)
	query := `
		SELECT DISTINCT c.id FROM channels c
		LEFT JOIN members m ON c.server_id = m.server_id AND m.user_id = $1
		LEFT JOIN channel_recipients cr ON c.id = cr.channel_id AND cr.user_id = $1
		WHERE m.user_id IS NOT NULL OR cr.user_id IS NOT NULL
	`
	var channelIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &channelIDs, query, userID)
	if err != nil {
		return nil, err
	}

	summary := &models.UnreadSummary{
		Channels: make([]models.ChannelUnreadInfo, 0, len(channelIDs)),
	}

	for _, channelID := range channelIDs {
		info, err := r.GetUnreadInfo(ctx, userID, channelID)
		if err != nil {
			continue
		}
		if info.HasUnread {
			summary.Channels = append(summary.Channels, *info)
			summary.TotalUnread += info.UnreadCount
			summary.TotalMentions += info.MentionCount
		}
	}

	return summary, nil
}

// GetServerUnreadSummary gets unread information for all channels in a server
func (r *ReadStateRepository) GetServerUnreadSummary(ctx context.Context, userID, serverID uuid.UUID) (*models.UnreadSummary, error) {
	// Get all channels in the server
	var channelIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &channelIDs, `SELECT id FROM channels WHERE server_id = $1`, serverID)
	if err != nil {
		return nil, err
	}

	summary := &models.UnreadSummary{
		Channels: make([]models.ChannelUnreadInfo, 0, len(channelIDs)),
	}

	for _, channelID := range channelIDs {
		info, err := r.GetUnreadInfo(ctx, userID, channelID)
		if err != nil {
			continue
		}
		if info.HasUnread {
			summary.Channels = append(summary.Channels, *info)
			summary.TotalUnread += info.UnreadCount
			summary.TotalMentions += info.MentionCount
		}
	}

	return summary, nil
}

// MarkServerAsRead marks all channels in a server as read for a user
func (r *ReadStateRepository) MarkServerAsRead(ctx context.Context, userID, serverID uuid.UUID) error {
	// Get all channels in the server with their last message
	query := `
		INSERT INTO read_states (user_id, channel_id, last_message_id, mention_count, updated_at)
		SELECT $1, c.id, c.last_message_id, 0, $2
		FROM channels c
		WHERE c.server_id = $3 AND c.last_message_id IS NOT NULL
		ON CONFLICT (user_id, channel_id) DO UPDATE SET
			last_message_id = EXCLUDED.last_message_id,
			mention_count = 0,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now(), serverID)
	return err
}

// DeleteReadState deletes the read state for a channel (when channel is deleted)
func (r *ReadStateRepository) DeleteByChannel(ctx context.Context, channelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM read_states WHERE channel_id = $1`, channelID)
	return err
}

// DeleteByUser deletes all read states for a user
func (r *ReadStateRepository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM read_states WHERE user_id = $1`, userID)
	return err
}
