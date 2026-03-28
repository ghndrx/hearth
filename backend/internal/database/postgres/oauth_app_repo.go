package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"hearth/internal/models"
)

// OAuthAppRepository handles OAuth application data storage
type OAuthAppRepository struct {
	db *sqlx.DB
}

// NewOAuthAppRepository creates a new OAuth app repository
func NewOAuthAppRepository(db *sqlx.DB) *OAuthAppRepository {
	return &OAuthAppRepository{db: db}
}

// Helper types for database operations with array handling
type oauthAppRow struct {
	ID               uuid.UUID      `db:"id"`
	OwnerID          uuid.UUID      `db:"owner_id"`
	Name             string         `db:"name"`
	Description      *string        `db:"description"`
	ClientID         string         `db:"client_id"`
	ClientSecretHash string         `db:"client_secret_hash"`
	RedirectURIs     pq.StringArray `db:"redirect_uris"`
	Scopes           pq.StringArray `db:"scopes"`
	IconURL          *string        `db:"icon_url"`
	HomepageURL      *string        `db:"homepage_url"`
	PrivacyURL       *string        `db:"privacy_url"`
	TermsURL         *string        `db:"terms_url"`
	IsPublic         bool           `db:"is_public"`
	IsVerified       bool           `db:"is_verified"`
	IsActive         bool           `db:"is_active"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

func (r *oauthAppRow) toModel() *models.OAuthApp {
	return &models.OAuthApp{
		ID:           r.ID,
		OwnerID:      r.OwnerID,
		Name:         r.Name,
		Description:  r.Description,
		ClientID:     r.ClientID,
		ClientSecret: r.ClientSecretHash,
		RedirectURIs: r.RedirectURIs,
		Scopes:       r.Scopes,
		IconURL:      r.IconURL,
		HomepageURL:  r.HomepageURL,
		PrivacyURL:   r.PrivacyURL,
		TermsURL:     r.TermsURL,
		IsPublic:     r.IsPublic,
		IsVerified:   r.IsVerified,
		IsActive:     r.IsActive,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// CreateApp creates a new OAuth application
func (r *OAuthAppRepository) CreateApp(ctx context.Context, app *models.OAuthApp) error {
	query := `
		INSERT INTO oauth_apps (
			id, owner_id, name, description, client_id, client_secret_hash,
			redirect_uris, scopes, icon_url, homepage_url, privacy_url, terms_url,
			is_public, is_verified, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		app.ID, app.OwnerID, app.Name, app.Description, app.ClientID, app.ClientSecret,
		pq.Array(app.RedirectURIs), pq.Array(app.Scopes),
		app.IconURL, app.HomepageURL, app.PrivacyURL, app.TermsURL,
		app.IsPublic, app.IsVerified, app.IsActive, app.CreatedAt, app.UpdatedAt,
	)
	return err
}

// GetAppByID retrieves an app by ID
func (r *OAuthAppRepository) GetAppByID(ctx context.Context, id uuid.UUID) (*models.OAuthApp, error) {
	var row oauthAppRow
	query := `SELECT * FROM oauth_apps WHERE id = $1`
	err := r.db.GetContext(ctx, &row, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toModel(), nil
}

// GetAppByClientID retrieves an app by client_id
func (r *OAuthAppRepository) GetAppByClientID(ctx context.Context, clientID string) (*models.OAuthApp, error) {
	var row oauthAppRow
	query := `SELECT * FROM oauth_apps WHERE client_id = $1 AND is_active = true`
	err := r.db.GetContext(ctx, &row, query, clientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toModel(), nil
}

// GetAppsByOwner retrieves all apps owned by a user
func (r *OAuthAppRepository) GetAppsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.OAuthApp, error) {
	var rows []oauthAppRow
	query := `SELECT * FROM oauth_apps WHERE owner_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &rows, query, ownerID)
	if err != nil {
		return nil, err
	}
	apps := make([]*models.OAuthApp, len(rows))
	for i, row := range rows {
		apps[i] = row.toModel()
	}
	return apps, nil
}

// UpdateApp updates an OAuth application
func (r *OAuthAppRepository) UpdateApp(ctx context.Context, app *models.OAuthApp) error {
	query := `
		UPDATE oauth_apps SET
			name = $1, description = $2, redirect_uris = $3, scopes = $4,
			icon_url = $5, homepage_url = $6, privacy_url = $7, terms_url = $8,
			is_public = $9, is_active = $10, updated_at = NOW()
		WHERE id = $11 AND owner_id = $12
	`
	result, err := r.db.ExecContext(ctx, query,
		app.Name, app.Description, pq.Array(app.RedirectURIs), pq.Array(app.Scopes),
		app.IconURL, app.HomepageURL, app.PrivacyURL, app.TermsURL,
		app.IsPublic, app.IsActive, app.ID, app.OwnerID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteApp deletes an OAuth application
func (r *OAuthAppRepository) DeleteApp(ctx context.Context, id, ownerID uuid.UUID) error {
	query := `DELETE FROM oauth_apps WHERE id = $1 AND owner_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, ownerID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RegenerateSecret updates the client secret hash
func (r *OAuthAppRepository) RegenerateSecret(ctx context.Context, id, ownerID uuid.UUID, newSecretHash string) error {
	query := `UPDATE oauth_apps SET client_secret_hash = $1, updated_at = NOW() WHERE id = $2 AND owner_id = $3`
	result, err := r.db.ExecContext(ctx, query, newSecretHash, id, ownerID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Authorization Code operations

type authCodeRow struct {
	ID                  uuid.UUID      `db:"id"`
	Code                string         `db:"code"`
	ClientID            string         `db:"client_id"`
	UserID              uuid.UUID      `db:"user_id"`
	Scopes              pq.StringArray `db:"scopes"`
	RedirectURI         string         `db:"redirect_uri"`
	CodeChallenge       *string        `db:"code_challenge"`
	CodeChallengeMethod *string        `db:"code_challenge_method"`
	Nonce               *string        `db:"nonce"`
	State               *string        `db:"state"`
	ExpiresAt           time.Time      `db:"expires_at"`
	Used                bool           `db:"used"`
	CreatedAt           time.Time      `db:"created_at"`
}

func (r *authCodeRow) toModel() *models.OAuthAuthorizationCode {
	return &models.OAuthAuthorizationCode{
		ID:                  r.ID,
		Code:                r.Code,
		ClientID:            r.ClientID,
		UserID:              r.UserID,
		Scopes:              r.Scopes,
		RedirectURI:         r.RedirectURI,
		CodeChallenge:       r.CodeChallenge,
		CodeChallengeMethod: r.CodeChallengeMethod,
		Nonce:               r.Nonce,
		State:               r.State,
		ExpiresAt:           r.ExpiresAt,
		Used:                r.Used,
		CreatedAt:           r.CreatedAt,
	}
}

// CreateAuthorizationCode stores a new authorization code
func (r *OAuthAppRepository) CreateAuthorizationCode(ctx context.Context, code *models.OAuthAuthorizationCode) error {
	query := `
		INSERT INTO oauth_authorization_codes (
			id, code, client_id, user_id, scopes, redirect_uri,
			code_challenge, code_challenge_method, nonce, state, expires_at, used, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		code.ID, code.Code, code.ClientID, code.UserID, pq.Array(code.Scopes), code.RedirectURI,
		code.CodeChallenge, code.CodeChallengeMethod, code.Nonce, code.State,
		code.ExpiresAt, code.Used, code.CreatedAt,
	)
	return err
}

// GetAuthorizationCode retrieves an authorization code by hash
func (r *OAuthAppRepository) GetAuthorizationCode(ctx context.Context, codeHash string) (*models.OAuthAuthorizationCode, error) {
	var row authCodeRow
	query := `SELECT * FROM oauth_authorization_codes WHERE code = $1 AND used = false AND expires_at > NOW()`
	err := r.db.GetContext(ctx, &row, query, codeHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toModel(), nil
}

// MarkAuthorizationCodeUsed marks an authorization code as used
func (r *OAuthAppRepository) MarkAuthorizationCodeUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE oauth_authorization_codes SET used = true WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// CleanupExpiredAuthCodes removes expired authorization codes
func (r *OAuthAppRepository) CleanupExpiredAuthCodes(ctx context.Context) (int64, error) {
	query := `DELETE FROM oauth_authorization_codes WHERE expires_at < NOW() OR used = true`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Access Token operations

type accessTokenRow struct {
	ID        uuid.UUID      `db:"id"`
	TokenHash string         `db:"token_hash"`
	ClientID  string         `db:"client_id"`
	UserID    uuid.UUID      `db:"user_id"`
	Scopes    pq.StringArray `db:"scopes"`
	ExpiresAt time.Time      `db:"expires_at"`
	RevokedAt *time.Time     `db:"revoked_at"`
	CreatedAt time.Time      `db:"created_at"`
}

func (r *accessTokenRow) toModel() *models.OAuthAccessToken {
	return &models.OAuthAccessToken{
		ID:        r.ID,
		TokenHash: r.TokenHash,
		ClientID:  r.ClientID,
		UserID:    r.UserID,
		Scopes:    r.Scopes,
		ExpiresAt: r.ExpiresAt,
		RevokedAt: r.RevokedAt,
		CreatedAt: r.CreatedAt,
	}
}

// CreateAccessToken stores a new access token
func (r *OAuthAppRepository) CreateAccessToken(ctx context.Context, token *models.OAuthAccessToken) error {
	query := `
		INSERT INTO oauth_access_tokens (
			id, token_hash, client_id, user_id, scopes, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.TokenHash, token.ClientID, token.UserID,
		pq.Array(token.Scopes), token.ExpiresAt, token.CreatedAt,
	)
	return err
}

// GetAccessTokenByHash retrieves an access token by hash
func (r *OAuthAppRepository) GetAccessTokenByHash(ctx context.Context, tokenHash string) (*models.OAuthAccessToken, error) {
	var row accessTokenRow
	query := `SELECT * FROM oauth_access_tokens WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`
	err := r.db.GetContext(ctx, &row, query, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toModel(), nil
}

// RevokeAccessToken revokes an access token
func (r *OAuthAppRepository) RevokeAccessToken(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE oauth_access_tokens SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// RevokeAccessTokensByUser revokes all access tokens for a user and client
func (r *OAuthAppRepository) RevokeAccessTokensByUser(ctx context.Context, userID uuid.UUID, clientID string) error {
	query := `UPDATE oauth_access_tokens SET revoked_at = NOW() WHERE user_id = $1 AND client_id = $2 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID, clientID)
	return err
}

// Refresh Token operations

type refreshTokenRow struct {
	ID            uuid.UUID      `db:"id"`
	TokenHash     string         `db:"token_hash"`
	AccessTokenID uuid.UUID      `db:"access_token_id"`
	ClientID      string         `db:"client_id"`
	UserID        uuid.UUID      `db:"user_id"`
	Scopes        pq.StringArray `db:"scopes"`
	ExpiresAt     time.Time      `db:"expires_at"`
	RotatedAt     *time.Time     `db:"rotated_at"`
	RotatedToID   *uuid.UUID     `db:"rotated_to_id"`
	RevokedAt     *time.Time     `db:"revoked_at"`
	RevokedReason *string        `db:"revoked_reason"`
	CreatedAt     time.Time      `db:"created_at"`
}

func (r *refreshTokenRow) toModel() *models.OAuthRefreshToken {
	return &models.OAuthRefreshToken{
		ID:            r.ID,
		TokenHash:     r.TokenHash,
		AccessTokenID: r.AccessTokenID,
		ClientID:      r.ClientID,
		UserID:        r.UserID,
		Scopes:        r.Scopes,
		ExpiresAt:     r.ExpiresAt,
		RotatedAt:     r.RotatedAt,
		RotatedToID:   r.RotatedToID,
		RevokedAt:     r.RevokedAt,
		RevokedReason: r.RevokedReason,
		CreatedAt:     r.CreatedAt,
	}
}

// CreateRefreshToken stores a new refresh token
func (r *OAuthAppRepository) CreateRefreshToken(ctx context.Context, token *models.OAuthRefreshToken) error {
	query := `
		INSERT INTO oauth_refresh_tokens (
			id, token_hash, access_token_id, client_id, user_id, scopes, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.TokenHash, token.AccessTokenID, token.ClientID, token.UserID,
		pq.Array(token.Scopes), token.ExpiresAt, token.CreatedAt,
	)
	return err
}

// GetRefreshTokenByHash retrieves a refresh token by hash
func (r *OAuthAppRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.OAuthRefreshToken, error) {
	var row refreshTokenRow
	query := `SELECT * FROM oauth_refresh_tokens WHERE token_hash = $1`
	err := r.db.GetContext(ctx, &row, query, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toModel(), nil
}

// RotateRefreshToken marks a refresh token as rotated and links to new token
func (r *OAuthAppRepository) RotateRefreshToken(ctx context.Context, oldID, newID uuid.UUID) error {
	query := `UPDATE oauth_refresh_tokens SET rotated_at = NOW(), rotated_to_id = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, newID, oldID)
	return err
}

// RevokeRefreshToken revokes a refresh token
func (r *OAuthAppRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID, reason string) error {
	query := `UPDATE oauth_refresh_tokens SET revoked_at = NOW(), revoked_reason = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, reason, id)
	return err
}

// RevokeRefreshTokenFamily revokes all refresh tokens in a family (for reuse detection)
func (r *OAuthAppRepository) RevokeRefreshTokenFamily(ctx context.Context, accessTokenID uuid.UUID, reason string) error {
	query := `UPDATE oauth_refresh_tokens SET revoked_at = NOW(), revoked_reason = $1 WHERE access_token_id = $2 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, reason, accessTokenID)
	return err
}

// RevokeRefreshTokensByUser revokes all refresh tokens for a user and client
func (r *OAuthAppRepository) RevokeRefreshTokensByUser(ctx context.Context, userID uuid.UUID, clientID string) error {
	query := `UPDATE oauth_refresh_tokens SET revoked_at = NOW(), revoked_reason = 'user_revoked' WHERE user_id = $1 AND client_id = $2 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID, clientID)
	return err
}

// User Authorization operations

type userAuthRow struct {
	ID           uuid.UUID      `db:"id"`
	UserID       uuid.UUID      `db:"user_id"`
	ClientID     string         `db:"client_id"`
	Scopes       pq.StringArray `db:"scopes"`
	AuthorizedAt time.Time      `db:"authorized_at"`
	LastUsedAt   time.Time      `db:"last_used_at"`
	RevokedAt    *time.Time     `db:"revoked_at"`
}

func (r *userAuthRow) toModel() *models.OAuthUserAuthorization {
	return &models.OAuthUserAuthorization{
		ID:           r.ID,
		UserID:       r.UserID,
		ClientID:     r.ClientID,
		Scopes:       r.Scopes,
		AuthorizedAt: r.AuthorizedAt,
		LastUsedAt:   r.LastUsedAt,
		RevokedAt:    r.RevokedAt,
	}
}

// CreateOrUpdateUserAuthorization creates or updates a user's authorization for an app
func (r *OAuthAppRepository) CreateOrUpdateUserAuthorization(ctx context.Context, auth *models.OAuthUserAuthorization) error {
	query := `
		INSERT INTO oauth_user_authorizations (
			id, user_id, client_id, scopes, authorized_at, last_used_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, client_id) DO UPDATE SET
			scopes = EXCLUDED.scopes,
			last_used_at = EXCLUDED.last_used_at,
			revoked_at = NULL
	`
	_, err := r.db.ExecContext(ctx, query,
		auth.ID, auth.UserID, auth.ClientID, pq.Array(auth.Scopes),
		auth.AuthorizedAt, auth.LastUsedAt,
	)
	return err
}

// GetUserAuthorization retrieves a user's authorization for an app
func (r *OAuthAppRepository) GetUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) (*models.OAuthUserAuthorization, error) {
	var row userAuthRow
	query := `SELECT * FROM oauth_user_authorizations WHERE user_id = $1 AND client_id = $2 AND revoked_at IS NULL`
	err := r.db.GetContext(ctx, &row, query, userID, clientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toModel(), nil
}

// GetUserAuthorizations retrieves all active authorizations for a user
func (r *OAuthAppRepository) GetUserAuthorizations(ctx context.Context, userID uuid.UUID) ([]*models.OAuthUserAuthorization, error) {
	var rows []userAuthRow
	query := `SELECT * FROM oauth_user_authorizations WHERE user_id = $1 AND revoked_at IS NULL ORDER BY last_used_at DESC`
	err := r.db.SelectContext(ctx, &rows, query, userID)
	if err != nil {
		return nil, err
	}
	auths := make([]*models.OAuthUserAuthorization, len(rows))
	for i, row := range rows {
		auths[i] = row.toModel()
	}
	return auths, nil
}

// RevokeUserAuthorization revokes a user's authorization for an app
func (r *OAuthAppRepository) RevokeUserAuthorization(ctx context.Context, userID uuid.UUID, clientID string) error {
	query := `UPDATE oauth_user_authorizations SET revoked_at = NOW() WHERE user_id = $1 AND client_id = $2 AND revoked_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, userID, clientID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *OAuthAppRepository) UpdateLastUsed(ctx context.Context, userID uuid.UUID, clientID string) error {
	query := `UPDATE oauth_user_authorizations SET last_used_at = NOW() WHERE user_id = $1 AND client_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, clientID)
	return err
}
