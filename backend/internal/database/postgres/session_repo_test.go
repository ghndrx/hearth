package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func setupSessionRepoMock(t *testing.T) (*SessionRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewSessionRepository(sqlxDB)
	return repo, mock
}

func TestSessionRepo_CreateSession(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()

	now := time.Now()
	session := &models.Session{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		TokenHash:       "hash123",
		Device:          ptrString("Chrome on Windows"),
		DeviceName:      ptrString("My Laptop"),
		DeviceType:      models.DeviceTypeDesktop,
		Browser:         ptrString("Chrome"),
		BrowserVersion:  ptrString("120.0"),
		OS:              ptrString("Windows"),
		OSVersion:       ptrString("11"),
		IPAddress:       ptrString("192.168.1.1"),
		UserAgent:       ptrString("Mozilla/5.0"),
		LocationCity:    ptrString("New York"),
		LocationCountry: ptrString("US"),
		IsCurrent:       true,
		LastUsed:        &now,
		ExpiresAt:       now.Add(24 * time.Hour),
		CreatedAt:      now,
	}

	mock.ExpectExec("INSERT INTO sessions").
		WithArgs(
			session.ID, session.UserID, session.TokenHash, session.Device,
			session.DeviceName, session.DeviceType, session.Browser, session.BrowserVersion,
			session.OS, session.OSVersion, session.IPAddress, session.UserAgent,
			session.LocationCity, session.LocationCountry, session.IsCurrent,
			session.LastUsed, session.ExpiresAt, session.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateSession(ctx, session)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetSessionByID(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	sessionID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "token_hash", "device", "device_name", "device_type",
		"browser", "browser_version", "os", "os_version",
		"ip_address", "user_agent", "location_city", "location_country",
		"is_current", "last_used", "expires_at", "created_at",
	}).AddRow(
		sessionID, uuid.New(), "hash", "Chrome", ptrString("Laptop"),
		models.DeviceTypeDesktop, ptrString("Chrome"), ptrString("120"),
		ptrString("Windows"), ptrString("11"), ptrString("1.1"), ptrString("UA"),
		ptrString("NYC"), ptrString("US"), true, &now, now.Add(24*time.Hour), now,
	)

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE id = \\$1").
		WithArgs(sessionID).
		WillReturnRows(rows)

	session, err := repo.GetSessionByID(ctx, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, sessionID, session.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetSessionByID_NotFound(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	sessionID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE id = \\$1").
		WithArgs(sessionID).
		WillReturnError(sql.ErrNoRows)

	session, err := repo.GetSessionByID(ctx, sessionID)
	assert.Nil(t, session)
	assert.ErrorIs(t, err, ErrSessionNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetUserSessions(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "token_hash", "device", "device_name", "device_type",
		"browser", "browser_version", "os", "os_version",
		"ip_address", "user_agent", "location_city", "location_country",
		"is_current", "last_used", "expires_at", "created_at",
	}).
		AddRow(uuid.New(), userID, "hash1", "Chrome", nil, models.DeviceTypeDesktop, nil, nil, nil, nil, nil, nil, nil, nil, true, &now, now.Add(24*time.Hour), now).
		AddRow(uuid.New(), userID, "hash2", "Safari", nil, models.DeviceTypeMobile, nil, nil, nil, nil, nil, nil, nil, nil, false, &now, now.Add(24*time.Hour), now)

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE user_id = \\$1").
		WithArgs(userID).
		WillReturnRows(rows)

	sessions, err := repo.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetUserSessions_Empty(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "token_hash", "device", "device_name", "device_type",
		"browser", "browser_version", "os", "os_version",
		"ip_address", "user_agent", "location_city", "location_country",
		"is_current", "last_used", "expires_at", "created_at",
	})

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE user_id = \\$1").
		WithArgs(userID).
		WillReturnRows(rows)

	sessions, err := repo.GetUserSessions(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_UpdateSessionActivity(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	sessionID := uuid.New()

	mock.ExpectExec("UPDATE sessions SET last_used").
		WithArgs(sessionID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateSessionActivity(ctx, sessionID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_SetCurrentSession(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()
	sessionID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE sessions SET is_current = FALSE").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec("UPDATE sessions SET is_current = TRUE").
		WithArgs(sessionID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.SetCurrentSession(ctx, userID, sessionID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_SetCurrentSession_BeginError(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()
	sessionID := uuid.New()

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err := repo.SetCurrentSession(ctx, userID, sessionID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RevokeSession(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	sessionID := uuid.New()

	mock.ExpectExec("UPDATE sessions SET expires_at").
		WithArgs(sessionID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RevokeSession(ctx, sessionID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RevokeSession_NotFound(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	sessionID := uuid.New()

	mock.ExpectExec("UPDATE sessions SET expires_at").
		WithArgs(sessionID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.RevokeSession(ctx, sessionID)
	assert.ErrorIs(t, err, ErrSessionNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RevokeAllUserSessions(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()
	exceptID := uuid.New()

	mock.ExpectExec("UPDATE sessions SET expires_at = NOW\\(\\) WHERE user_id = \\$1 AND id != \\$2").
		WithArgs(userID, exceptID).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := repo.RevokeAllUserSessions(ctx, userID, &exceptID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RevokeAllUserSessions_NoException(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectExec("UPDATE sessions SET expires_at = NOW\\(\\) WHERE user_id = \\$1 AND expires_at > NOW\\(\\)").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 5))

	err := repo.RevokeAllUserSessions(ctx, userID, nil)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_DeleteExpiredSessions(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	olderThan := 30 * 24 * time.Hour // 30 days

	mock.ExpectExec("DELETE FROM sessions WHERE expires_at < NOW\\(\\) - \\$1::interval").
		WithArgs(olderThan.String()).
		WillReturnResult(sqlmock.NewResult(0, 10))

	count, err := repo.DeleteExpiredSessions(ctx, olderThan)
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Refresh Token Tests ---

func TestSessionRepo_CreateRefreshToken(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	now := time.Now()

	token := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "tokenhash123",
		FamilyID:  uuid.New(),
		SessionID: uuid.New(),
		Used:      false,
		Revoked:   false,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		CreatedAt: now,
	}

	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs(
			token.ID, token.UserID, token.TokenHash, token.FamilyID,
			token.SessionID, token.Used, token.Revoked, token.ExpiresAt, token.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateRefreshToken(ctx, token)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetRefreshTokenByHash(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	now := time.Now()
	tokenID := uuid.New()
	familyID := uuid.New()
	sessionID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "token_hash", "family_id", "session_id",
		"used", "used_at", "revoked", "revoked_at", "expires_at", "created_at",
	}).AddRow(tokenID, uuid.New(), "hash", familyID, sessionID, false, nil, false, nil, now.Add(30*24*time.Hour), now)

	mock.ExpectQuery("SELECT .+ FROM refresh_tokens WHERE token_hash = \\$1").
		WithArgs("hash").
		WillReturnRows(rows)

	token, err := repo.GetRefreshTokenByHash(ctx, "hash")
	require.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, tokenID, token.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetRefreshTokenByHash_NotFound(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT .+ FROM refresh_tokens WHERE token_hash = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	token, err := repo.GetRefreshTokenByHash(ctx, "nonexistent")
	assert.Nil(t, token)
	assert.ErrorIs(t, err, ErrRefreshTokenNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_MarkRefreshTokenUsed(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	tokenID := uuid.New()

	mock.ExpectExec("UPDATE refresh_tokens SET used = TRUE").
		WithArgs(tokenID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkRefreshTokenUsed(ctx, tokenID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RevokeTokenFamily(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	familyID := uuid.New()

	mock.ExpectExec("UPDATE refresh_tokens SET revoked = TRUE").
		WithArgs(familyID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := repo.RevokeTokenFamily(ctx, familyID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RevokeAllUserTokens(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()
	exceptFamily := uuid.New()

	mock.ExpectExec("UPDATE refresh_tokens SET revoked = TRUE").
		WithArgs(userID, exceptFamily).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := repo.RevokeAllUserTokens(ctx, userID, &exceptFamily)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RevokeAllUserTokens_NoException(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectExec("UPDATE refresh_tokens SET revoked = TRUE, revoked_at = NOW\\(\\) WHERE user_id = \\$1 AND revoked = FALSE").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 4))

	err := repo.RevokeAllUserTokens(ctx, userID, nil)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetLatestTokenInFamily(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	now := time.Now()
	tokenID := uuid.New()
	familyID := uuid.New()
	sessionID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "token_hash", "family_id", "session_id",
		"used", "used_at", "revoked", "revoked_at", "expires_at", "created_at",
	}).AddRow(tokenID, uuid.New(), "hash", familyID, sessionID, false, nil, false, nil, now.Add(30*24*time.Hour), now)

	mock.ExpectQuery("SELECT .+ FROM refresh_tokens WHERE family_id = \\$1 AND revoked = FALSE").
		WithArgs(familyID).
		WillReturnRows(rows)

	token, err := repo.GetLatestTokenInFamily(ctx, familyID)
	require.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, tokenID, token.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetLatestTokenInFamily_NotFound(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	familyID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM refresh_tokens WHERE family_id = \\$1 AND revoked = FALSE").
		WithArgs(familyID).
		WillReturnError(sql.ErrNoRows)

	token, err := repo.GetLatestTokenInFamily(ctx, familyID)
	assert.Nil(t, token)
	assert.ErrorIs(t, err, ErrRefreshTokenNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_DeleteExpiredTokens(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	olderThan := 30 * 24 * time.Hour // 30 days

	mock.ExpectExec("DELETE FROM refresh_tokens").
		WithArgs(olderThan.String()).
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.DeleteExpiredTokens(ctx, olderThan)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RotateRefreshToken(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	now := time.Now()
	oldTokenID := uuid.New()

	newToken := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "newtokenhash",
		FamilyID:  uuid.New(),
		SessionID: uuid.New(),
		Used:      false,
		Revoked:   false,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		CreatedAt: now,
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE refresh_tokens SET used = TRUE").
		WithArgs(oldTokenID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs(
			newToken.ID, newToken.UserID, newToken.TokenHash, newToken.FamilyID,
			newToken.SessionID, newToken.Used, newToken.Revoked, newToken.ExpiresAt, newToken.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.RotateRefreshToken(ctx, oldTokenID, newToken)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_RotateRefreshToken_Rollback(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	oldTokenID := uuid.New()
	now := time.Now()

	newToken := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "newtokenhash",
		FamilyID:  uuid.New(),
		SessionID: uuid.New(),
		Used:      false,
		Revoked:   false,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
		CreatedAt: now,
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE refresh_tokens SET used = TRUE").
		WithArgs(oldTokenID).
		WillReturnError(errors.New("update failed"))
	mock.ExpectRollback()

	err := repo.RotateRefreshToken(ctx, oldTokenID, newToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

func TestSessionRepo_WithTransaction_Commit(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return nil
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_WithTransaction_Rollback(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()

	mock.ExpectBegin()
	mock.ExpectRollback()

	err := repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return errors.New("operation failed")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_WithTransaction_BeginError(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(errors.New("begin error"))

	err := repo.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "begin error")
}

func TestSessionRepo_GetUserSessions_DBError(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	userID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE user_id = \\$1").
		WithArgs(userID).
		WillReturnError(errors.New("database connection error"))

	sessions, err := repo.GetUserSessions(ctx, userID)
	assert.Nil(t, sessions)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepo_GetSessionByID_DBError(t *testing.T) {
	repo, mock := setupSessionRepoMock(t)
	ctx := context.Background()
	sessionID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM sessions WHERE id = \\$1").
		WithArgs(sessionID).
		WillReturnError(errors.New("database error"))

	_, err := repo.GetSessionByID(ctx, sessionID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefreshToken_IsValid(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	tests := []struct {
		name     string
		token    models.RefreshToken
		expected bool
	}{
		{
			name: "valid token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   false,
				ExpiresAt: future,
			},
			expected: true,
		},
		{
			name: "used token",
			token: models.RefreshToken{
				Used:      true,
				Revoked:   false,
				ExpiresAt: future,
			},
			expected: false,
		},
		{
			name: "revoked token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   true,
				ExpiresAt: future,
			},
			expected: false,
		},
		{
			name: "expired token",
			token: models.RefreshToken{
				Used:      false,
				Revoked:   false,
				ExpiresAt: past,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.IsValid()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestHashToken(t *testing.T) {
	token := "my-secret-token"
	hash1 := models.HashToken(token)
	hash2 := models.HashToken(token)

	assert.Equal(t, hash1, hash2, "same token should produce same hash")
	assert.Len(t, hash1, 64, "SHA-256 hash should be 64 hex characters")
}

func TestGenerateTokenFamily(t *testing.T) {
	family1 := models.GenerateTokenFamily()
	family2 := models.GenerateTokenFamily()

	assert.NotEqual(t, family1, family2, "each call should generate unique family ID")
	assert.NotEqual(t, uuid.Nil, family1)
	assert.NotEqual(t, uuid.Nil, family2)
}

func ptrString(s string) *string {
	return &s
}
