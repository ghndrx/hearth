package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	
	"hearth/internal/models"
)

type ChannelRepository struct {
	db *sqlx.DB
}

func NewChannelRepository(db *sqlx.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

func (r *ChannelRepository) Create(ctx context.Context, channel *models.Channel) error {
	query := `
		INSERT INTO channels (id, server_id, name, topic, type, position, parent_id, slowmode, nsfw, e2ee_enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		channel.ID, channel.ServerID, channel.Name, channel.Topic, channel.Type,
		channel.Position, channel.ParentID, channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
		channel.CreatedAt,
	)
	if err != nil {
		return err
	}
	
	// For DM channels, add recipients
	if len(channel.Recipients) > 0 {
		for _, userID := range channel.Recipients {
			_, _ = r.db.ExecContext(ctx,
				`INSERT INTO channel_recipients (channel_id, user_id) VALUES ($1, $2)`,
				channel.ID, userID,
			)
		}
	}
	
	return nil
}

func (r *ChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	var channel models.Channel
	query := `SELECT * FROM channels WHERE id = $1`
	err := r.db.GetContext(ctx, &channel, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	// Load recipients for DM channels
	if channel.Type == models.ChannelTypeDM || channel.Type == models.ChannelTypeGroupDM {
		var recipients []uuid.UUID
		_ = r.db.SelectContext(ctx, &recipients,
			`SELECT user_id FROM channel_recipients WHERE channel_id = $1`, id)
		channel.Recipients = recipients
	}
	
	return &channel, nil
}

func (r *ChannelRepository) Update(ctx context.Context, channel *models.Channel) error {
	query := `
		UPDATE channels SET
			name = $2, topic = $3, position = $4, parent_id = $5,
			slowmode = $6, nsfw = $7, e2ee_enabled = $8
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		channel.ID, channel.Name, channel.Topic, channel.Position, channel.ParentID,
		channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
	)
	return err
}

func (r *ChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM channels WHERE id = $1`, id)
	return err
}

func (r *ChannelRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	var channels []*models.Channel
	query := `SELECT * FROM channels WHERE server_id = $1 ORDER BY position`
	err := r.db.SelectContext(ctx, &channels, query, serverID)
	return channels, err
}

func (r *ChannelRepository) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	var channelID uuid.UUID
	query := `
		SELECT c.id FROM channels c
		INNER JOIN channel_recipients r1 ON r1.channel_id = c.id AND r1.user_id = $1
		INNER JOIN channel_recipients r2 ON r2.channel_id = c.id AND r2.user_id = $2
		WHERE c.type = $3
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &channelID, query, user1ID, user2ID, models.ChannelTypeDM)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return r.GetByID(ctx, channelID)
}

func (r *ChannelRepository) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	query := `
		SELECT c.* FROM channels c
		INNER JOIN channel_recipients r ON r.channel_id = c.id
		WHERE r.user_id = $1 AND c.type IN ($2, $3)
		ORDER BY c.last_message_at DESC NULLS LAST
	`
	var channels []*models.Channel
	err := r.db.SelectContext(ctx, &channels, query, userID, models.ChannelTypeDM, models.ChannelTypeGroupDM)
	if err != nil {
		return nil, err
	}
	
	// Load recipients for each channel
	for _, ch := range channels {
		var recipients []uuid.UUID
		_ = r.db.SelectContext(ctx, &recipients,
			`SELECT user_id FROM channel_recipients WHERE channel_id = $1`, ch.ID)
		ch.Recipients = recipients
	}
	
	return channels, nil
}

func (r *ChannelRepository) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	query := `UPDATE channels SET last_message_id = $2, last_message_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, channelID, messageID, at)
	return err
}

// CreateDMChannel creates a DM channel between two users
func (r *ChannelRepository) CreateDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	// Check if already exists
	existing, err := r.GetDMChannel(ctx, user1ID, user2ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	
	// Create new DM channel
	channel := &models.Channel{
		ID:          uuid.New(),
		Type:        models.ChannelTypeDM,
		E2EEEnabled: true, // DMs are always encrypted
		Recipients:  []uuid.UUID{user1ID, user2ID},
		CreatedAt:   time.Now(),
	}
	
	if err := r.Create(ctx, channel); err != nil {
		return nil, err
	}
	
	return channel, nil
}

// CreateGroupDM creates a group DM channel
func (r *ChannelRepository) CreateGroupDM(ctx context.Context, ownerID uuid.UUID, name string, recipients []uuid.UUID) (*models.Channel, error) {
	channel := &models.Channel{
		ID:          uuid.New(),
		Name:        name,
		Type:        models.ChannelTypeGroupDM,
		OwnerID:     &ownerID,
		E2EEEnabled: false, // Group DMs are not encrypted by default
		Recipients:  append([]uuid.UUID{ownerID}, recipients...),
		CreatedAt:   time.Now(),
	}
	
	if err := r.Create(ctx, channel); err != nil {
		return nil, err
	}
	
	return channel, nil
}
