package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// InviteRepo implements InviteRepository
type InviteRepo struct {
	db *sql.DB
}

// NewInviteRepo creates a new invite repository
func NewInviteRepo(db *sql.DB) *InviteRepo {
	return &InviteRepo{db: db}
}

// Create creates a new invite
func (r *InviteRepo) Create(ctx context.Context, invite *models.Invite) error {
	query := `
		INSERT INTO invites (code, server_id, channel_id, creator_id, max_uses, uses, expires_at, temporary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		invite.Code,
		invite.ServerID,
		invite.ChannelID,
		invite.CreatorID,
		invite.MaxUses,
		invite.Uses,
		invite.ExpiresAt,
		invite.Temporary,
		invite.CreatedAt,
	)
	return err
}

// GetByCode retrieves an invite by code
func (r *InviteRepo) GetByCode(ctx context.Context, code string) (*models.Invite, error) {
	query := `
		SELECT code, server_id, channel_id, creator_id, max_uses, uses, expires_at, temporary, created_at
		FROM invites
		WHERE code = $1`

	invite := &models.Invite{}
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&invite.Code,
		&invite.ServerID,
		&invite.ChannelID,
		&invite.CreatorID,
		&invite.MaxUses,
		&invite.Uses,
		&invite.ExpiresAt,
		&invite.Temporary,
		&invite.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return invite, nil
}

// GetByServerID retrieves all invites for a server
func (r *InviteRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	query := `
		SELECT code, server_id, channel_id, creator_id, max_uses, uses, expires_at, temporary, created_at
		FROM invites
		WHERE server_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []*models.Invite
	for rows.Next() {
		invite := &models.Invite{}
		if err := rows.Scan(
			&invite.Code,
			&invite.ServerID,
			&invite.ChannelID,
			&invite.CreatorID,
			&invite.MaxUses,
			&invite.Uses,
			&invite.ExpiresAt,
			&invite.Temporary,
			&invite.CreatedAt,
		); err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}
	return invites, rows.Err()
}

// IncrementUses increments the uses count
func (r *InviteRepo) IncrementUses(ctx context.Context, code string) error {
	query := `UPDATE invites SET uses = uses + 1 WHERE code = $1`
	_, err := r.db.ExecContext(ctx, query, code)
	return err
}

// Delete deletes an invite
func (r *InviteRepo) Delete(ctx context.Context, code string) error {
	query := `DELETE FROM invites WHERE code = $1`
	_, err := r.db.ExecContext(ctx, query, code)
	return err
}

// DeleteExpired deletes all expired invites
func (r *InviteRepo) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM invites WHERE expires_at IS NOT NULL AND expires_at < $1`
	result, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// BanRepo implements BanRepository
type BanRepo struct {
	db *sql.DB
}

// NewBanRepo creates a new ban repository
func NewBanRepo(db *sql.DB) *BanRepo {
	return &BanRepo{db: db}
}

// Create creates a new ban
func (r *BanRepo) Create(ctx context.Context, ban *models.Ban) error {
	query := `
		INSERT INTO bans (server_id, user_id, reason, banned_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (server_id, user_id) DO UPDATE SET
			reason = EXCLUDED.reason,
			banned_by = EXCLUDED.banned_by,
			created_at = EXCLUDED.created_at`

	_, err := r.db.ExecContext(ctx, query,
		ban.ServerID,
		ban.UserID,
		ban.Reason,
		ban.BannedBy,
		ban.CreatedAt,
	)
	return err
}

// GetByServerAndUser retrieves a ban by server and user
func (r *BanRepo) GetByServerAndUser(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	query := `
		SELECT server_id, user_id, reason, banned_by, created_at
		FROM bans
		WHERE server_id = $1 AND user_id = $2`

	ban := &models.Ban{}
	err := r.db.QueryRowContext(ctx, query, serverID, userID).Scan(
		&ban.ServerID,
		&ban.UserID,
		&ban.Reason,
		&ban.BannedBy,
		&ban.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ban, nil
}

// GetByServerID retrieves all bans for a server
func (r *BanRepo) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	query := `
		SELECT server_id, user_id, reason, banned_by, created_at
		FROM bans
		WHERE server_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bans []*models.Ban
	for rows.Next() {
		ban := &models.Ban{}
		if err := rows.Scan(
			&ban.ServerID,
			&ban.UserID,
			&ban.Reason,
			&ban.BannedBy,
			&ban.CreatedAt,
		); err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}
	return bans, rows.Err()
}

// Delete removes a ban
func (r *BanRepo) Delete(ctx context.Context, serverID, userID uuid.UUID) error {
	query := `DELETE FROM bans WHERE server_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, serverID, userID)
	return err
}
