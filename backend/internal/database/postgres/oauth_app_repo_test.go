package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hearth/internal/models"
)

func TestOAuthAppRepository_CreateApp(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	now := time.Now()

	app := &models.OAuthApp{
		ID:           uuid.New(),
		OwnerID:      uuid.New(),
		Name:         "Test App",
		Description:  strPtr("A test application"),
		ClientID:     "test-client-id",
		ClientSecret: "hashed-secret",
		RedirectURIs: []string{"https://example.com/callback"},
		Scopes:       []string{"read", "write"},
		IsPublic:     false,
		IsVerified:   false,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec("INSERT INTO oauth_apps").
		WithArgs(
			app.ID, app.OwnerID, app.Name, app.Description, app.ClientID, app.ClientSecret,
			pq.Array(app.RedirectURIs), pq.Array(app.Scopes),
			nil, nil, nil, nil, // icon, homepage, privacy, terms URLs
			app.IsPublic, app.IsVerified, app.IsActive, app.CreatedAt, app.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateApp(ctx, app)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_GetAppByClientID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	clientID := "test-client-id"
	appID := uuid.New()
	ownerID := uuid.New()
	now := time.Now()

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "owner_id", "name", "description", "client_id", "client_secret_hash",
			"redirect_uris", "scopes", "icon_url", "homepage_url", "privacy_url", "terms_url",
			"is_public", "is_verified", "is_active", "created_at", "updated_at",
		}).AddRow(
			appID, ownerID, "Test App", nil, clientID, "hashed-secret",
			pq.Array([]string{"https://example.com/callback"}), pq.Array([]string{"read"}),
			nil, nil, nil, nil,
			false, false, true, now, now,
		)

		mock.ExpectQuery("SELECT \\* FROM oauth_apps WHERE client_id = \\$1 AND is_active = true").
			WithArgs(clientID).
			WillReturnRows(rows)

		app, err := repo.GetAppByClientID(ctx, clientID)
		assert.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, clientID, app.ClientID)
		assert.Equal(t, "Test App", app.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM oauth_apps WHERE client_id = \\$1 AND is_active = true").
			WithArgs("nonexistent").
			WillReturnError(sql.ErrNoRows)

		app, err := repo.GetAppByClientID(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, app)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOAuthAppRepository_CreateAuthorizationCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	now := time.Now()

	code := &models.OAuthAuthorizationCode{
		ID:          uuid.New(),
		Code:        "hashed-code",
		ClientID:    "test-client",
		UserID:      uuid.New(),
		Scopes:      []string{"read"},
		RedirectURI: "https://example.com/callback",
		ExpiresAt:   now.Add(10 * time.Minute),
		Used:        false,
		CreatedAt:   now,
	}

	mock.ExpectExec("INSERT INTO oauth_authorization_codes").
		WithArgs(
			code.ID, code.Code, code.ClientID, code.UserID, pq.Array(code.Scopes), code.RedirectURI,
			nil, nil, nil, nil, // PKCE and nonce fields
			code.ExpiresAt, code.Used, code.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateAuthorizationCode(ctx, code)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_GetAuthorizationCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	codeHash := "hashed-code"
	codeID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(10 * time.Minute)

	t.Run("found and valid", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "code", "client_id", "user_id", "scopes", "redirect_uri",
			"code_challenge", "code_challenge_method", "nonce", "state",
			"expires_at", "used", "created_at",
		}).AddRow(
			codeID, codeHash, "test-client", userID, pq.Array([]string{"read"}), "https://example.com/callback",
			nil, nil, nil, nil,
			expiresAt, false, now,
		)

		mock.ExpectQuery("SELECT \\* FROM oauth_authorization_codes WHERE code = \\$1 AND used = false AND expires_at > NOW\\(\\)").
			WithArgs(codeHash).
			WillReturnRows(rows)

		code, err := repo.GetAuthorizationCode(ctx, codeHash)
		assert.NoError(t, err)
		assert.NotNil(t, code)
		assert.Equal(t, codeHash, code.Code)
		assert.Equal(t, userID, code.UserID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM oauth_authorization_codes WHERE code = \\$1 AND used = false AND expires_at > NOW\\(\\)").
			WithArgs("invalid-hash").
			WillReturnError(sql.ErrNoRows)

		code, err := repo.GetAuthorizationCode(ctx, "invalid-hash")
		assert.NoError(t, err)
		assert.Nil(t, code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOAuthAppRepository_MarkAuthorizationCodeUsed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	codeID := uuid.New()

	mock.ExpectExec("UPDATE oauth_authorization_codes SET used = true WHERE id = \\$1").
		WithArgs(codeID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.MarkAuthorizationCodeUsed(ctx, codeID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_CreateAccessToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	now := time.Now()

	token := &models.OAuthAccessToken{
		ID:        uuid.New(),
		TokenHash: "hashed-token",
		ClientID:  "test-client",
		UserID:    uuid.New(),
		Scopes:    []string{"read", "write"},
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedAt: now,
	}

	mock.ExpectExec("INSERT INTO oauth_access_tokens").
		WithArgs(
			token.ID, token.TokenHash, token.ClientID, token.UserID,
			pq.Array(token.Scopes), token.ExpiresAt, token.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateAccessToken(ctx, token)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_GetAccessTokenByHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	tokenHash := "hashed-token"
	tokenID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	t.Run("found and valid", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "token_hash", "client_id", "user_id", "scopes", "expires_at", "revoked_at", "created_at",
		}).AddRow(
			tokenID, tokenHash, "test-client", userID, pq.Array([]string{"read"}), expiresAt, nil, now,
		)

		mock.ExpectQuery("SELECT \\* FROM oauth_access_tokens WHERE token_hash = \\$1 AND revoked_at IS NULL AND expires_at > NOW\\(\\)").
			WithArgs(tokenHash).
			WillReturnRows(rows)

		token, err := repo.GetAccessTokenByHash(ctx, tokenHash)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, tokenHash, token.TokenHash)
		assert.Equal(t, userID, token.UserID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOAuthAppRepository_CreateRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	now := time.Now()

	token := &models.OAuthRefreshToken{
		ID:            uuid.New(),
		TokenHash:     "hashed-refresh-token",
		AccessTokenID: uuid.New(),
		ClientID:      "test-client",
		UserID:        uuid.New(),
		Scopes:        []string{"read"},
		ExpiresAt:     now.Add(30 * 24 * time.Hour),
		CreatedAt:     now,
	}

	mock.ExpectExec("INSERT INTO oauth_refresh_tokens").
		WithArgs(
			token.ID, token.TokenHash, token.AccessTokenID, token.ClientID, token.UserID,
			pq.Array(token.Scopes), token.ExpiresAt, token.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateRefreshToken(ctx, token)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_RotateRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	oldID := uuid.New()
	newID := uuid.New()

	mock.ExpectExec("UPDATE oauth_refresh_tokens SET rotated_at = NOW\\(\\), rotated_to_id = \\$1 WHERE id = \\$2").
		WithArgs(newID, oldID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.RotateRefreshToken(ctx, oldID, newID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_RevokeRefreshTokenFamily(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	accessTokenID := uuid.New()
	reason := "reuse_detected"

	mock.ExpectExec("UPDATE oauth_refresh_tokens SET revoked_at = NOW\\(\\), revoked_reason = \\$1 WHERE access_token_id = \\$2 AND revoked_at IS NULL").
		WithArgs(reason, accessTokenID).
		WillReturnResult(sqlmock.NewResult(0, 3)) // Revoked 3 tokens in the family

	err = repo.RevokeRefreshTokenFamily(ctx, accessTokenID, reason)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_CreateOrUpdateUserAuthorization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	now := time.Now()

	auth := &models.OAuthUserAuthorization{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		ClientID:     "test-client",
		Scopes:       []string{"read", "write"},
		AuthorizedAt: now,
		LastUsedAt:   now,
	}

	mock.ExpectExec("INSERT INTO oauth_user_authorizations").
		WithArgs(
			auth.ID, auth.UserID, auth.ClientID, pq.Array(auth.Scopes),
			auth.AuthorizedAt, auth.LastUsedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateOrUpdateUserAuthorization(ctx, auth)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_GetUserAuthorizations(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "client_id", "scopes", "authorized_at", "last_used_at", "revoked_at",
	}).AddRow(
		uuid.New(), userID, "client-1", pq.Array([]string{"read"}), now, now, nil,
	).AddRow(
		uuid.New(), userID, "client-2", pq.Array([]string{"read", "write"}), now, now, nil,
	)

	mock.ExpectQuery("SELECT \\* FROM oauth_user_authorizations WHERE user_id = \\$1 AND revoked_at IS NULL ORDER BY last_used_at DESC").
		WithArgs(userID).
		WillReturnRows(rows)

	auths, err := repo.GetUserAuthorizations(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, auths, 2)
	assert.Equal(t, "client-1", auths[0].ClientID)
	assert.Equal(t, "client-2", auths[1].ClientID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthAppRepository_RevokeUserAuthorization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	userID := uuid.New()
	clientID := "test-client"

	t.Run("successful revocation", func(t *testing.T) {
		mock.ExpectExec("UPDATE oauth_user_authorizations SET revoked_at = NOW\\(\\) WHERE user_id = \\$1 AND client_id = \\$2 AND revoked_at IS NULL").
			WithArgs(userID, clientID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.RevokeUserAuthorization(ctx, userID, clientID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("UPDATE oauth_user_authorizations SET revoked_at = NOW\\(\\) WHERE user_id = \\$1 AND client_id = \\$2 AND revoked_at IS NULL").
			WithArgs(userID, "nonexistent").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.RevokeUserAuthorization(ctx, userID, "nonexistent")
		assert.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOAuthAppRepository_DeleteApp(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthAppRepository(sqlxDB)

	ctx := context.Background()
	appID := uuid.New()
	ownerID := uuid.New()

	t.Run("successful deletion", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM oauth_apps WHERE id = \\$1 AND owner_id = \\$2").
			WithArgs(appID, ownerID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteApp(ctx, appID, ownerID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found or not owner", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM oauth_apps WHERE id = \\$1 AND owner_id = \\$2").
			WithArgs(appID, ownerID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteApp(ctx, appID, ownerID)
		assert.Error(t, err)
		assert.Equal(t, sql.ErrNoRows, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func strPtr(s string) *string {
	return &s
}
