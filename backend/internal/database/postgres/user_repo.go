package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
	"hearth/internal/services"
)

var ErrUserNotFound = services.ErrUserNotFound

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, discriminator, email, password_hash, avatar_url, banner_url, bio, status, mfa_enabled, verified, flags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Discriminator, user.Email, user.PasswordHash,
		user.AvatarURL, user.BannerURL, user.Bio, user.Status, user.MFAEnabled,
		user.Verified, user.Flags, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE username = $1`
	err := r.db.GetContext(ctx, &user, query, username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users SET
			username = $2, discriminator = $3, email = $4, password_hash = $5,
			avatar_url = $6, banner_url = $7, bio = $8, status = $9, 
			custom_status = $10, mfa_enabled = $11, verified = $12, flags = $13, updated_at = $14
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Discriminator, user.Email, user.PasswordHash,
		user.AvatarURL, user.BannerURL, user.Bio, user.Status, user.CustomStatus,
		user.MFAEnabled, user.Verified, user.Flags, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *UserRepository) UpdateMFA(ctx context.Context, userID uuid.UUID, enabled bool, secret *string) error {
	query := `UPDATE users SET mfa_enabled = $2, mfa_secret = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, enabled, secret)
	return err
}

// Friends

func (r *UserRepository) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON (r.user_id = $1 AND r.target_id = u.id AND r.type = 1)
		OR (r.target_id = $1 AND r.user_id = u.id AND r.type = 1)
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

func (r *UserRepository) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	query := `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 1, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 1
	`
	_, err := r.db.ExecContext(ctx, query, userID, friendID, time.Now())
	return err
}

func (r *UserRepository) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	query := `DELETE FROM relationships WHERE (user_id = $1 AND target_id = $2) OR (user_id = $2 AND target_id = $1)`
	_, err := r.db.ExecContext(ctx, query, userID, friendID)
	return err
}

// GetRelationship gets the relationship between two users
func (r *UserRepository) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	var relType int
	query := `SELECT type FROM relationships WHERE user_id = $1 AND target_id = $2`
	err := r.db.GetContext(ctx, &relType, query, userID, targetID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return relType, err
}

// SendFriendRequest creates a pending friend request from sender to receiver
func (r *UserRepository) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()

	// Create outgoing request for sender
	_, err = tx.ExecContext(ctx, `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 4, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 4, created_at = $3
	`, senderID, receiverID, now)
	if err != nil {
		return err
	}

	// Create incoming request for receiver
	_, err = tx.ExecContext(ctx, `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 3, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 3, created_at = $3
	`, receiverID, senderID, now)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetIncomingFriendRequests gets all pending incoming friend requests for a user
func (r *UserRepository) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON r.user_id = $1 AND r.target_id = u.id AND r.type = 3
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

// GetOutgoingFriendRequests gets all pending outgoing friend requests for a user
func (r *UserRepository) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON r.user_id = $1 AND r.target_id = u.id AND r.type = 4
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

// AcceptFriendRequest accepts a pending friend request
func (r *UserRepository) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update both relationships to type 1 (friend)
	_, err = tx.ExecContext(ctx, `
		UPDATE relationships SET type = 1 WHERE user_id = $1 AND target_id = $2 AND type = 3
	`, receiverID, senderID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE relationships SET type = 1 WHERE user_id = $1 AND target_id = $2 AND type = 4
	`, senderID, receiverID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeclineFriendRequest declines/cancels a pending friend request
func (r *UserRepository) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove both pending relationships (works for both declining incoming and canceling outgoing)
	_, err = tx.ExecContext(ctx, `
		DELETE FROM relationships WHERE user_id = $1 AND target_id = $2 AND type IN (3, 4)
	`, userID, otherID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM relationships WHERE user_id = $1 AND target_id = $2 AND type IN (3, 4)
	`, otherID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *UserRepository) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	query := `
		SELECT u.* FROM users u
		INNER JOIN relationships r ON r.user_id = $1 AND r.target_id = u.id AND r.type = 2
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID)
	return users, err
}

func (r *UserRepository) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	query := `
		INSERT INTO relationships (user_id, target_id, type, created_at)
		VALUES ($1, $2, 2, $3)
		ON CONFLICT (user_id, target_id) DO UPDATE SET type = 2
	`
	_, err := r.db.ExecContext(ctx, query, userID, blockedID, time.Now())
	return err
}

func (r *UserRepository) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	query := `DELETE FROM relationships WHERE user_id = $1 AND target_id = $2 AND type = 2`
	_, err := r.db.ExecContext(ctx, query, userID, blockedID)
	return err
}

// Presence

func (r *UserRepository) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	query := `
		INSERT INTO presence (user_id, status, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET status = $2, updated_at = $3
	`
	_, err := r.db.ExecContext(ctx, query, userID, status, time.Now())
	return err
}

func (r *UserRepository) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	var presence models.Presence
	query := `SELECT * FROM presence WHERE user_id = $1`
	err := r.db.GetContext(ctx, &presence, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &presence, err
}

func (r *UserRepository) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*models.Presence), nil
	}

	query, args, err := sqlx.In(`SELECT * FROM presence WHERE user_id IN (?)`, userIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var presences []*models.Presence
	if err := r.db.SelectContext(ctx, &presences, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]*models.Presence)
	for _, p := range presences {
		result[p.UserID] = p
	}
	return result, nil
}

// RecentActivity represents a user's recent activity for profile display
type RecentActivity struct {
	LastMessageAt     *time.Time `db:"last_message_at"`
	LastMessageServer *uuid.UUID `db:"last_message_server"`
	ServerName        *string    `db:"server_name"`
	ChannelName       *string    `db:"channel_name"`
	MessageCount24h   int        `db:"message_count_24h"`
}

// GetRecentActivity gets a user's recent activity (visible to requester via mutual servers)
func (r *UserRepository) GetRecentActivity(ctx context.Context, requesterID, targetID uuid.UUID) (*RecentActivity, error) {
	// Get the most recent message in a mutual server (both users are members)
	activity := &RecentActivity{}

	query := `
		SELECT 
			MAX(m.created_at) as last_message_at,
			(SELECT server_id FROM channels c2 
			 INNER JOIN messages m2 ON m2.channel_id = c2.id 
			 WHERE m2.author_id = $2 AND c2.server_id IS NOT NULL
			 ORDER BY m2.created_at DESC LIMIT 1) as last_message_server
		FROM messages m
		INNER JOIN channels c ON c.id = m.channel_id
		INNER JOIN servers s ON s.id = c.server_id
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		WHERE m.author_id = $2
	`

	err := r.db.GetContext(ctx, activity, query, requesterID, targetID)
	if err == sql.ErrNoRows {
		return activity, nil
	}
	if err != nil {
		return nil, err
	}

	// Get server and channel name for last message if we have server ID
	if activity.LastMessageServer != nil {
		var info struct {
			ServerName  string  `db:"server_name"`
			ChannelName *string `db:"channel_name"`
		}
		infoQuery := `
			SELECT s.name as server_name, c.name as channel_name
			FROM messages m
			INNER JOIN channels c ON c.id = m.channel_id
			INNER JOIN servers s ON s.id = c.server_id
			WHERE m.author_id = $1 AND c.server_id = $2
			ORDER BY m.created_at DESC
			LIMIT 1
		`
		if err := r.db.GetContext(ctx, &info, infoQuery, targetID, *activity.LastMessageServer); err == nil {
			activity.ServerName = &info.ServerName
			activity.ChannelName = info.ChannelName
		}
	}

	// Get message count in last 24 hours (in mutual servers)
	countQuery := `
		SELECT COUNT(*) FROM messages m
		INNER JOIN channels c ON c.id = m.channel_id
		INNER JOIN servers s ON s.id = c.server_id
		INNER JOIN members m1 ON m1.server_id = s.id AND m1.user_id = $1
		INNER JOIN members m2 ON m2.server_id = s.id AND m2.user_id = $2
		WHERE m.author_id = $2 AND m.created_at > NOW() - INTERVAL '24 hours'
	`
	_ = r.db.GetContext(ctx, &activity.MessageCount24h, countQuery, requesterID, targetID)

	return activity, nil
}

// GetMutualFriends gets friends that both users have in common
func (r *UserRepository) GetMutualFriends(ctx context.Context, userID1, userID2 uuid.UUID, limit int) ([]*models.User, int, error) {
	// Get total count
	var total int
	countQuery := `
		SELECT COUNT(DISTINCT u.id) FROM users u
		WHERE EXISTS (
			SELECT 1 FROM relationships r1 
			WHERE r1.user_id = $1 AND r1.target_id = u.id AND r1.type = 1
		)
		AND EXISTS (
			SELECT 1 FROM relationships r2 
			WHERE r2.user_id = $2 AND r2.target_id = u.id AND r2.type = 1
		)
	`
	if err := r.db.GetContext(ctx, &total, countQuery, userID1, userID2); err != nil {
		return nil, 0, err
	}

	// Get mutual friends
	query := `
		SELECT u.* FROM users u
		WHERE EXISTS (
			SELECT 1 FROM relationships r1 
			WHERE r1.user_id = $1 AND r1.target_id = u.id AND r1.type = 1
		)
		AND EXISTS (
			SELECT 1 FROM relationships r2 
			WHERE r2.user_id = $2 AND r2.target_id = u.id AND r2.type = 1
		)
		ORDER BY u.username
		LIMIT $3
	`
	var users []*models.User
	err := r.db.SelectContext(ctx, &users, query, userID1, userID2, limit)
	return users, total, err
}

// GetCustomStatus retrieves a user's custom status
func (r *UserRepository) GetCustomStatus(ctx context.Context, userID uuid.UUID) (*models.UserCustomStatus, error) {
	var status models.UserCustomStatus
	query := `SELECT * FROM user_custom_statuses WHERE user_id = $1`
	err := r.db.GetContext(ctx, &status, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &status, err
}

// SetCustomStatus upserts a user's custom status
func (r *UserRepository) SetCustomStatus(ctx context.Context, status *models.UserCustomStatus) error {
	query := `
		INSERT INTO user_custom_statuses (user_id, custom_text, emoji, emoji_id, emoji_name, clear_after, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			custom_text = $2, emoji = $3, emoji_id = $4, emoji_name = $5, clear_after = $6, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query,
		status.UserID, status.CustomText, status.Emoji, status.EmojiID, status.EmojiName, status.ClearAfter,
	)
	return err
}

// DeleteCustomStatus removes a user's custom status
func (r *UserRepository) DeleteCustomStatus(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_custom_statuses WHERE user_id = $1`, userID)
	return err
}

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrTokenAlreadyUsed     = errors.New("refresh token already used")
	ErrTokenRevoked         = errors.New("refresh token revoked")
	ErrTokenExpired         = errors.New("refresh token expired")
)

// SessionRepository handles session and refresh token persistence
type SessionRepository struct {
	db *sqlx.DB
}

// NewSessionRepository creates a new SessionRepository
func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// --- Session Operations ---

// CreateSession creates a new session with device info
func (r *SessionRepository) CreateSession(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (
			id, user_id, token_hash, device, device_name, device_type,
			browser, browser_version, os, os_version,
			ip_address, user_agent, location_city, location_country,
			is_current, last_used, expires_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.TokenHash, session.Device,
		session.DeviceName, session.DeviceType, session.Browser, session.BrowserVersion,
		session.OS, session.OSVersion, session.IPAddress, session.UserAgent,
		session.LocationCity, session.LocationCountry, session.IsCurrent,
		session.LastUsed, session.ExpiresAt, session.CreatedAt,
	)
	return err
}

// GetSessionByID retrieves a session by ID
func (r *SessionRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	var session models.Session
	query := `
		SELECT id, user_id, token_hash, device, device_name, device_type,
			browser, browser_version, os, os_version,
			ip_address, user_agent, location_city, location_country,
			is_current, last_used, expires_at, created_at
		FROM sessions
		WHERE id = $1 AND expires_at > NOW()
	`
	err := r.db.GetContext(ctx, &session, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	return &session, err
}

// GetUserSessions retrieves all active sessions for a user
func (r *SessionRepository) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	var sessions []*models.Session
	query := `
		SELECT id, user_id, token_hash, device, device_name, device_type,
			browser, browser_version, os, os_version,
			ip_address, user_agent, location_city, location_country,
			is_current, last_used, expires_at, created_at
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY last_used DESC NULLS LAST, created_at DESC
	`
	err := r.db.SelectContext(ctx, &sessions, query, userID)
	return sessions, err
}

// UpdateSessionActivity updates the last_used timestamp
func (r *SessionRepository) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	query := `UPDATE sessions SET last_used = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	return err
}

// SetCurrentSession marks a session as current and unmarks others
func (r *SessionRepository) SetCurrentSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Unmark all sessions as current
	_, err = tx.ExecContext(ctx, `UPDATE sessions SET is_current = FALSE WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	// Mark the specified session as current
	_, err = tx.ExecContext(ctx, `UPDATE sessions SET is_current = TRUE, last_used = NOW() WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RevokeSession revokes a specific session
func (r *SessionRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `UPDATE sessions SET expires_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// RevokeAllUserSessions revokes all sessions for a user except the current one
func (r *SessionRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	var query string
	var args []interface{}

	if exceptSessionID != nil {
		query = `UPDATE sessions SET expires_at = NOW() WHERE user_id = $1 AND id != $2 AND expires_at > NOW()`
		args = []interface{}{userID, *exceptSessionID}
	} else {
		query = `UPDATE sessions SET expires_at = NOW() WHERE user_id = $1 AND expires_at > NOW()`
		args = []interface{}{userID}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteExpiredSessions removes expired sessions
func (r *SessionRepository) DeleteExpiredSessions(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < NOW() - $1::interval`
	result, err := r.db.ExecContext(ctx, query, olderThan.String())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- Refresh Token Operations ---

// CreateRefreshToken creates a new refresh token
func (r *SessionRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (
			id, user_id, token_hash, family_id, session_id,
			used, revoked, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.TokenHash, token.FamilyID,
		token.SessionID, token.Used, token.Revoked, token.ExpiresAt, token.CreatedAt,
	)
	return err
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *SessionRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `
		SELECT id, user_id, token_hash, family_id, session_id,
			used, used_at, revoked, revoked_at, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	err := r.db.GetContext(ctx, &token, query, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRefreshTokenNotFound
	}
	return &token, err
}

// MarkRefreshTokenUsed marks a token as used
func (r *SessionRepository) MarkRefreshTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET used = TRUE, used_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, tokenID)
	return err
}

// RevokeTokenFamily revokes all tokens in a family
func (r *SessionRepository) RevokeTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked = TRUE, revoked_at = NOW() WHERE family_id = $1 AND revoked = FALSE`
	_, err := r.db.ExecContext(ctx, query, familyID)
	return err
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *SessionRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, exceptFamilyID *uuid.UUID) error {
	var query string
	var args []interface{}

	if exceptFamilyID != nil {
		query = `UPDATE refresh_tokens SET revoked = TRUE, revoked_at = NOW() WHERE user_id = $1 AND family_id != $2 AND revoked = FALSE`
		args = []interface{}{userID, *exceptFamilyID}
	} else {
		query = `UPDATE refresh_tokens SET revoked = TRUE, revoked_at = NOW() WHERE user_id = $1 AND revoked = FALSE`
		args = []interface{}{userID}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetLatestTokenInFamily gets the most recent unused token in a family
func (r *SessionRepository) GetLatestTokenInFamily(ctx context.Context, familyID uuid.UUID) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `
		SELECT id, user_id, token_hash, family_id, session_id,
			used, used_at, revoked, revoked_at, expires_at, created_at
		FROM refresh_tokens
		WHERE family_id = $1 AND revoked = FALSE
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &token, query, familyID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRefreshTokenNotFound
	}
	return &token, err
}

// DeleteExpiredTokens removes expired refresh tokens
func (r *SessionRepository) DeleteExpiredTokens(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM refresh_tokens 
		WHERE expires_at < NOW() - $1::interval
		   OR (revoked = TRUE AND revoked_at < NOW() - $1::interval)
	`
	result, err := r.db.ExecContext(ctx, query, olderThan.String())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- Transaction Support ---

// WithTransaction executes a function within a transaction
func (r *SessionRepository) WithTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// RotateRefreshToken atomically marks old token as used and creates new one
func (r *SessionRepository) RotateRefreshToken(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error {
	return r.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		// Mark old token as used
		_, err := tx.ExecContext(ctx, `UPDATE refresh_tokens SET used = TRUE, used_at = NOW() WHERE id = $1`, oldTokenID)
		if err != nil {
			return err
		}

		// Create new token
		query := `
			INSERT INTO refresh_tokens (
				id, user_id, token_hash, family_id, session_id,
				used, revoked, expires_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err = tx.ExecContext(ctx, query,
			newToken.ID, newToken.UserID, newToken.TokenHash, newToken.FamilyID,
			newToken.SessionID, newToken.Used, newToken.Revoked, newToken.ExpiresAt, newToken.CreatedAt,
		)
		return err
	})
}

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
