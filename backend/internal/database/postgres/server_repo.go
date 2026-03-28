package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"hearth/internal/models"
)

type ServerRepository struct {
	db *sqlx.DB
}

func NewServerRepository(db *sqlx.DB) *ServerRepository {
	return &ServerRepository{db: db}
}

func (r *ServerRepository) Create(ctx context.Context, server *models.Server) error {
	query := `
		INSERT INTO servers (id, name, icon_url, banner_url, description, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		server.ID, server.Name, server.IconURL, server.BannerURL, server.Description,
		server.OwnerID, server.CreatedAt, server.UpdatedAt,
	)
	return err
}

func (r *ServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	var server models.Server
	query := `
		SELECT 
			id, name, icon_url, banner_url, description,
			owner_id, verification_level, 
			explicit_filter as explicit_content_filter,
			default_notifications, features, vanity_url as vanity_url_code,
			created_at, updated_at
		FROM servers WHERE id = $1
	`
	err := r.db.GetContext(ctx, &server, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &server, err
}

func (r *ServerRepository) Update(ctx context.Context, server *models.Server) error {
	query := `
		UPDATE servers SET
			name = $2, icon_url = $3, banner_url = $4, description = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		server.ID, server.Name, server.IconURL, server.BannerURL, server.Description, server.UpdatedAt,
	)
	return err
}

func (r *ServerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM servers WHERE id = $1`, id)
	return err
}

func (r *ServerRepository) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	query := `UPDATE servers SET owner_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, serverID, newOwnerID)
	return err
}

// Members

func (r *ServerRepository) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	query := `
		SELECT m.server_id, m.user_id, m.nickname, m.joined_at, m.premium_since, m.deaf, m.mute, m.pending, m.temporary
		FROM members m
		WHERE m.server_id = $1
		ORDER BY m.joined_at DESC
		LIMIT $2 OFFSET $3
	`
	var members []*models.Member
	err := r.db.SelectContext(ctx, &members, query, serverID, limit, offset)
	return members, err
}

func (r *ServerRepository) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	limit = models.NormalizeLimit(limit)

	var args []interface{}
	args = append(args, serverID)

	query := `
		SELECT m.server_id, m.user_id, m.nickname, m.joined_at, m.premium_since, m.deaf, m.mute, m.pending, m.temporary
		FROM members m
		WHERE m.server_id = $1
	`

	if cursor != nil {
		query += ` AND (m.joined_at, m.user_id) < ($2, $3)`
		args = append(args, cursor.JoinedAt, cursor.UserID)
	}

	query += ` ORDER BY m.joined_at DESC, m.user_id DESC LIMIT $` + fmt.Sprint(len(args)+1)
	args = append(args, limit+1)

	var members []*models.Member
	err := r.db.SelectContext(ctx, &members, query, args...)
	if err != nil {
		return nil, err
	}

	hasMore := len(members) > limit
	if hasMore {
		members = members[:limit]
	}

	var nextCursor string
	if hasMore && len(members) > 0 {
		lastMember := members[len(members)-1]
		nextCursor = (&models.MemberCursor{
			JoinedAt: lastMember.JoinedAt,
			UserID:   lastMember.UserID,
		}).Encode()
	}

	return &models.PaginatedMembers{
		Members:    members,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (r *ServerRepository) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	var member models.Member
	query := `SELECT server_id, user_id, nickname, joined_at, premium_since, deaf, mute, pending, temporary FROM members WHERE server_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &member, query, serverID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *ServerRepository) AddMember(ctx context.Context, member *models.Member) error {
	query := `
		INSERT INTO members (user_id, server_id, nickname, joined_at, roles)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		member.UserID, member.ServerID, member.Nickname, member.JoinedAt, pq.Array(member.Roles),
	)
	return err
}

func (r *ServerRepository) UpdateMember(ctx context.Context, member *models.Member) error {
	query := `
		UPDATE members SET nickname = $3, roles = $4
		WHERE user_id = $1 AND server_id = $2
	`
	_, err := r.db.ExecContext(ctx, query,
		member.UserID, member.ServerID, member.Nickname, pq.Array(member.Roles),
	)
	return err
}

func (r *ServerRepository) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM members WHERE server_id = $1 AND user_id = $2`, serverID, userID)
	return err
}

func (r *ServerRepository) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM members WHERE server_id = $1`, serverID)
	return count, err
}

// User's servers

func (r *ServerRepository) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	query := `
		SELECT 
			s.id, s.name, s.icon_url, s.banner_url, s.description,
			s.owner_id, s.verification_level, 
			s.explicit_filter as explicit_content_filter,
			s.default_notifications, s.features, s.vanity_url as vanity_url_code,
			s.created_at, s.updated_at
		FROM servers s
		INNER JOIN members m ON m.server_id = s.id
		WHERE m.user_id = $1
		ORDER BY s.name
	`
	var servers []*models.Server
	err := r.db.SelectContext(ctx, &servers, query, userID)
	return servers, err
}

func (r *ServerRepository) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM servers WHERE owner_id = $1`, userID)
	return count, err
}

// Bans

func (r *ServerRepository) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	var ban models.Ban
	query := `SELECT * FROM bans WHERE server_id = $1 AND user_id = $2`
	err := r.db.GetContext(ctx, &ban, query, serverID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &ban, err
}

func (r *ServerRepository) AddBan(ctx context.Context, ban *models.Ban) error {
	query := `
		INSERT INTO bans (server_id, user_id, reason, banned_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		ban.ServerID, ban.UserID, ban.Reason, ban.BannedBy, ban.CreatedAt,
	)
	return err
}

func (r *ServerRepository) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM bans WHERE server_id = $1 AND user_id = $2`, serverID, userID)
	return err
}

func (r *ServerRepository) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	var bans []*models.Ban
	err := r.db.SelectContext(ctx, &bans, `SELECT * FROM bans WHERE server_id = $1`, serverID)
	return bans, err
}

// Invites

func (r *ServerRepository) CreateInvite(ctx context.Context, invite *models.Invite) error {
	query := `
		INSERT INTO invites (code, server_id, channel_id, creator_id, max_uses, uses, expires_at, is_vanity, vanity_code, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		invite.Code, invite.ServerID, invite.ChannelID, invite.CreatorID,
		invite.MaxUses, invite.Uses, invite.ExpiresAt,
		invite.IsVanity, invite.VanityCode, invite.CreatedAt,
	)
	return err
}

func (r *ServerRepository) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	var invite models.Invite
	query := `SELECT code, server_id, channel_id, creator_id, max_uses, uses, expires_at, temporary, is_vanity, vanity_code, created_at FROM invites WHERE code = $1`
	err := r.db.GetContext(ctx, &invite, query, code)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &invite, err
}

func (r *ServerRepository) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	var invite models.Invite
	query := `SELECT code, server_id, channel_id, creator_id, max_uses, uses, expires_at, temporary, is_vanity, vanity_code, created_at FROM invites WHERE vanity_code = $1`
	err := r.db.GetContext(ctx, &invite, query, vanityCode)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &invite, err
}

func (r *ServerRepository) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	var invites []*models.Invite
	err := r.db.SelectContext(ctx, &invites, `SELECT code, server_id, channel_id, creator_id, max_uses, uses, expires_at, temporary, is_vanity, vanity_code, created_at FROM invites WHERE server_id = $1`, serverID)
	return invites, err
}

func (r *ServerRepository) DeleteInvite(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM invites WHERE code = $1`, code)
	return err
}

func (r *ServerRepository) IncrementInviteUses(ctx context.Context, code string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE invites SET uses = uses + 1 WHERE code = $1`, code)
	return err
}

// Invite analytics

func (r *ServerRepository) LogInviteUse(ctx context.Context, logEntry *models.InviteUseLog) error {
	query := `
		INSERT INTO invite_use_logs (invite_code, server_id, user_id, joined_at, account_created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		logEntry.InviteCode, logEntry.ServerID, logEntry.UserID,
		logEntry.JoinedAt, logEntry.AccountCreatedAt,
	)
	return err
}

func (r *ServerRepository) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	var logs []models.InviteUseLog
	query := `SELECT id, invite_code, server_id, user_id, joined_at, account_created_at, account_age_days FROM invite_use_logs WHERE invite_code = $1 ORDER BY joined_at DESC`
	err := r.db.SelectContext(ctx, &logs, query, inviteCode)
	return logs, err
}

func (r *ServerRepository) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	var logs []models.InviteUseLog
	query := `SELECT id, invite_code, server_id, user_id, joined_at, account_created_at, account_age_days FROM invite_use_logs WHERE server_id = $1 ORDER BY joined_at DESC`
	err := r.db.SelectContext(ctx, &logs, query, serverID)
	return logs, err
}

// GetMembersWithRole returns all members who have a specific role
func (r *ServerRepository) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	query := `
		SELECT m.server_id, m.user_id, m.nickname, m.joined_at, m.premium_since, m.deaf, m.mute, m.pending, m.temporary
		FROM members m
		WHERE m.server_id = $1 AND $2 = ANY(m.roles)
	`
	var members []*models.Member
	err := r.db.SelectContext(ctx, &members, query, serverID, roleID)
	return members, err
}

// GetMutualServers returns servers that both users are members of
func (r *ServerRepository) GetMutualServers(ctx context.Context, userID1, userID2 uuid.UUID) ([]*models.Server, error) {
	query := `
		SELECT s.* FROM servers s
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		ORDER BY s.name
	`
	var servers []*models.Server
	err := r.db.SelectContext(ctx, &servers, query, userID1, userID2)
	return servers, err
}

// GetMutualServersLimited returns mutual servers with a limit (for popout display)
func (r *ServerRepository) GetMutualServersLimited(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.Server, int, error) {
	// Get total count first
	var total int
	countQuery := `
		SELECT COUNT(*) FROM servers s
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
	`
	if err := r.db.GetContext(ctx, &total, countQuery, userID1, userID2); err != nil {
		return nil, 0, err
	}

	// Get limited results
	query := `
		SELECT s.* FROM servers s
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		ORDER BY s.name
		LIMIT $3
	`
	var servers []*models.Server
	err := r.db.SelectContext(ctx, &servers, query, userID1, userID2, limit)
	return servers, total, err
}

// GetOnlineCount returns the approximate number of online members in a server
func (r *ServerRepository) GetOnlineCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	// This is a simplified implementation - in production you'd use presence data
	query := `
		SELECT COUNT(*) FROM members m
		LEFT JOIN user_presences p ON p.user_id = m.user_id
		WHERE m.server_id = $1 AND (p.status != 'offline' OR p.status IS NULL)
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, serverID)
	return count, err
}

// GetPublicInviteCode returns a public invite code for a server (if one exists)
func (r *ServerRepository) GetPublicInviteCode(ctx context.Context, serverID uuid.UUID) (string, error) {
	query := `
		SELECT code FROM invites 
		WHERE server_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY uses DESC
		LIMIT 1
	`
	var code string
	err := r.db.GetContext(ctx, &code, query, serverID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return code, err
}
