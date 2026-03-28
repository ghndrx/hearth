package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

// FollowRepository handles followed channel data access
type FollowRepository struct {
	db *sqlx.DB
}

// NewFollowRepository creates a new follow repository
func NewFollowRepository(db *sqlx.DB) *FollowRepository {
	return &FollowRepository{db: db}
}

// Create creates a new channel follow relationship
func (r *FollowRepository) Create(ctx context.Context, follow *models.FollowedChannel) error {
	query := `
		INSERT INTO followed_channels (channel_id, follower_channel_id, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, follow.ChannelID, follow.FollowerChannelID, follow.CreatedAt)
	return err
}

// Delete removes a channel follow relationship
func (r *FollowRepository) Delete(ctx context.Context, channelID, followerChannelID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM followed_channels WHERE channel_id = $1 AND follower_channel_id = $2`,
		channelID, followerChannelID,
	)
	return err
}

// GetByChannelAndFollower retrieves a follow by channel and follower
func (r *FollowRepository) GetByChannelAndFollower(ctx context.Context, channelID, followerChannelID uuid.UUID) (*models.FollowedChannel, error) {
	var follow models.FollowedChannel
	query := `SELECT channel_id, follower_channel_id, created_at FROM followed_channels WHERE channel_id = $1 AND follower_channel_id = $2`
	err := r.db.GetContext(ctx, &follow, query, channelID, followerChannelID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &follow, nil
}

// GetFollowers retrieves all followers of a channel
func (r *FollowRepository) GetFollowers(ctx context.Context, channelID uuid.UUID) ([]models.FollowedChannel, error) {
	var follows []models.FollowedChannel
	query := `SELECT channel_id, follower_channel_id, created_at FROM followed_channels WHERE channel_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &follows, query, channelID)
	if err != nil {
		return nil, err
	}
	return follows, nil
}
