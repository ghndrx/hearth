package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"hearth/internal/models"
)

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
