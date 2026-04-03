package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
		INSERT INTO channels (id, server_id, name, topic, type, position, parent_id, slowmode, nsfw, e2ee_enabled, icon, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		channel.ID, channel.ServerID, channel.Name, channel.Topic, channel.Type,
		channel.Position, channel.ParentID, channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
		channel.Icon, channel.CreatedAt,
	)
	if err != nil {
		return err
	}

	// For DM channels, add recipients
	var errs []error
	if len(channel.Recipients) > 0 {
		for _, userID := range channel.Recipients {
			_, err = r.db.ExecContext(ctx,
				`INSERT INTO channel_recipients (channel_id, user_id) VALUES ($1, $2)`,
				channel.ID, userID,
			)
			if err != nil {
				log.Printf("failed to insert recipient for channel %s, user %s: %v", channel.ID, userID, err)
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("channel created but %d recipient insert(s) failed: %w", len(errs), errs[0])
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
		if err := r.db.SelectContext(ctx, &recipients,
			`SELECT user_id FROM channel_recipients WHERE channel_id = $1`, id); err != nil {
			return nil, fmt.Errorf("failed to load recipients for channel %s: %w", id, err)
		}
		channel.Recipients = recipients
	}

	return &channel, nil
}

func (r *ChannelRepository) Update(ctx context.Context, channel *models.Channel) error {
	query := `
		UPDATE channels SET
			name = $2, topic = $3, position = $4, parent_id = $5,
			slowmode_seconds = $6, nsfw = $7, e2ee_enabled = $8, icon = $9
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		channel.ID, channel.Name, channel.Topic, channel.Position, channel.ParentID,
		channel.Slowmode, channel.NSFW, channel.E2EEEnabled, channel.Icon,
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
		SELECT c.id, c.server_id, c.parent_id, c.owner_id, c.type, c.name, c.topic, 
		       c.position, c.slowmode, c.nsfw, c.e2ee_enabled, c.bitrate, c.user_limit, 
		       c.rtc_region, c.last_message_id, c.created_at
		FROM channels c
		INNER JOIN channel_recipients r ON r.channel_id = c.id
		WHERE r.user_id = $1 AND c.type IN ($2, $3)
		ORDER BY c.created_at DESC NULLS LAST
	`
	var channels []*models.Channel
	err := r.db.SelectContext(ctx, &channels, query, userID, models.ChannelTypeDM, models.ChannelTypeGroupDM)
	if err != nil {
		return nil, err
	}

	// Load recipients for each channel
	for _, ch := range channels {
		var recipients []uuid.UUID
		if err := r.db.SelectContext(ctx, &recipients,
			`SELECT user_id FROM channel_recipients WHERE channel_id = $1`, ch.ID); err != nil {
			return nil, fmt.Errorf("failed to load recipients for channel %s: %w", ch.ID, err)
		}
		ch.Recipients = recipients
	}

	return channels, nil
}

func (r *ChannelRepository) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	query := `UPDATE channels SET last_message_id = $2, last_message_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, channelID, messageID, at)
	return err
}

func (r *ChannelRepository) UpdateForumConfig(ctx context.Context, channelID uuid.UUID, configJSON []byte) error {
	query := `UPDATE channels SET forum_config = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, channelID, configJSON)
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

// GetSharedChannels returns channels that both users have access to (in mutual servers)
// This includes text channels in servers where both users are members
func (r *ChannelRepository) GetSharedChannels(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.Channel, int, error) {
	// Get total count first
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT c.id) FROM channels c
		INNER JOIN servers s ON c.server_id = s.id
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		WHERE c.type IN ('text', 'announcement')
	`
	if err := r.db.GetContext(ctx, &total, countQuery, userID1, userID2); err != nil {
		return nil, 0, err
	}

	// Get channels with server info, ordered by last message activity
	query := `
		SELECT c.* FROM channels c
		INNER JOIN servers s ON c.server_id = s.id
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		WHERE c.type IN ('text', 'announcement')
		ORDER BY c.created_at DESC NULLS LAST, c.created_at DESC
		LIMIT $3
	`
	var channels []*models.Channel
	err := r.db.SelectContext(ctx, &channels, query, userID1, userID2, limit)
	return channels, total, err
}

// GetSharedChannelsWithServerNames returns shared channels including server names for display
type ChannelWithServer struct {
	models.Channel
	ServerName string  `db:"server_name"`
	ServerIcon *string `db:"server_icon"`
}

func (r *ChannelRepository) GetSharedChannelsWithServerNames(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*ChannelWithServer, int, error) {
	// Get total count first
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT c.id) FROM channels c
		INNER JOIN servers s ON c.server_id = s.id
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		WHERE c.type IN ('text', 'announcement')
	`
	if err := r.db.GetContext(ctx, &total, countQuery, userID1, userID2); err != nil {
		return nil, 0, err
	}

	// Get channels with server info
	query := `
		SELECT c.*, s.name as server_name, s.icon_url as server_icon 
		FROM channels c
		INNER JOIN servers s ON c.server_id = s.id
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		WHERE c.type IN ('text', 'announcement')
		ORDER BY c.created_at DESC NULLS LAST, c.created_at DESC
		LIMIT $3
	`
	var channels []*ChannelWithServer
	err := r.db.SelectContext(ctx, &channels, query, userID1, userID2, limit)
	return channels, total, err
}

// AddRecipient adds a user to a DM channel's recipients
func (r *ChannelRepository) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO channel_recipients (channel_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		channelID, userID,
	)
	return err
}

// RemoveRecipient removes a user from a DM channel's recipients
func (r *ChannelRepository) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM channel_recipients WHERE channel_id = $1 AND user_id = $2`,
		channelID, userID,
	)
	return err
}

// CountRecipients returns the number of recipients in a channel
func (r *ChannelRepository) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM channel_recipients WHERE channel_id = $1`, channelID)
	return count, err
}

// BulkUpdatePositions updates position and parent_id for multiple channels in a single transaction
func (r *ChannelRepository) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE channels SET position = $2, parent_id = $3 WHERE id = $1`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		_, err := stmt.ExecContext(ctx, entry.ID, entry.Position, entry.CategoryID)
		if err != nil {
			return fmt.Errorf("failed to update channel %s: %w", entry.ID, err)
		}
	}

	return tx.Commit()
}

// GetPermissionOverrides returns all permission overrides for a channel
func (r *ChannelRepository) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	var overrides []models.PermissionOverride
	query := `SELECT channel_id, target_type, target_id, allow, deny FROM channel_overrides WHERE channel_id = $1`
	err := r.db.SelectContext(ctx, &overrides, query, channelID)
	if err != nil {
		return nil, err
	}
	return overrides, nil
}

// UpsertPermissionOverride creates or updates a permission override for a channel
func (r *ChannelRepository) UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error {
	query := `
		INSERT INTO channel_overrides (channel_id, target_type, target_id, allow, deny)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (channel_id, target_type, target_id)
		DO UPDATE SET allow = $4, deny = $5
	`
	_, err := r.db.ExecContext(ctx, query,
		override.ChannelID, override.TargetType, override.TargetID, override.Allow, override.Deny,
	)
	return err
}

// DeletePermissionOverride removes a permission override from a channel
func (r *ChannelRepository) DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error {
	query := `DELETE FROM channel_overrides WHERE channel_id = $1 AND target_type = $2 AND target_id = $3`
	_, err := r.db.ExecContext(ctx, query, channelID, targetType, targetID)
	return err
}
