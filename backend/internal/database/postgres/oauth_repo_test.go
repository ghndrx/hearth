package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
)

func setupOAuthRepoMock(t *testing.T) (*OAuthRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewOAuthRepository(sqlxDB)
	return repo, mock
}

func TestOAuthRepository_Create(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	provider := &models.OAuthProvider{
		UserID:         uuid.New(),
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
	}

	mock.ExpectExec("INSERT INTO oauth_providers").
		WithArgs(
			sqlmock.AnyArg(), // ID
			provider.UserID,
			provider.Provider,
			provider.ProviderUserID,
			provider.Email,
			sqlmock.AnyArg(), // username
			sqlmock.AnyArg(), // display_name
			sqlmock.AnyArg(), // avatar_url
			sqlmock.AnyArg(), // access_token
			sqlmock.AnyArg(), // refresh_token
			sqlmock.AnyArg(), // token_expires_at
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, provider)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, provider.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_GetByID(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider", "provider_user_id", "email",
		"username", "display_name", "avatar_url", "access_token", "refresh_token",
		"token_expires_at", "created_at", "updated_at",
	}).AddRow(
		providerID, userID, "github", "12345", "test@example.com",
		nil, nil, nil, nil, nil,
		nil, now, now,
	)

	mock.ExpectQuery("SELECT \\* FROM oauth_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnRows(rows)

	provider, err := repo.GetByID(ctx, providerID)
	require.NoError(t, err)
	assert.Equal(t, providerID, provider.ID)
	assert.Equal(t, userID, provider.UserID)
	assert.Equal(t, "github", provider.Provider)
	assert.Equal(t, "12345", provider.ProviderUserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM oauth_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnError(sql.ErrNoRows)

	provider, err := repo.GetByID(ctx, providerID)
	assert.ErrorIs(t, err, services.ErrOAuthProviderNotFound)
	assert.Nil(t, provider)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_GetByUserID(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider", "provider_user_id", "email",
		"username", "display_name", "avatar_url", "access_token", "refresh_token",
		"token_expires_at", "created_at", "updated_at",
	}).AddRow(
		uuid.New(), userID, "github", "12345", "test@example.com",
		nil, nil, nil, nil, nil, nil, now, now,
	).AddRow(
		uuid.New(), userID, "google", "67890", "test@example.com",
		nil, nil, nil, nil, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT \\* FROM oauth_providers WHERE user_id = \\$1 ORDER BY created_at ASC").
		WithArgs(userID).
		WillReturnRows(rows)

	providers, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, providers, 2)
	assert.Equal(t, "github", providers[0].Provider)
	assert.Equal(t, "google", providers[1].Provider)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_GetByProviderUserID(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider", "provider_user_id", "email",
		"username", "display_name", "avatar_url", "access_token", "refresh_token",
		"token_expires_at", "created_at", "updated_at",
	}).AddRow(
		providerID, userID, "github", "12345", "test@example.com",
		nil, nil, nil, nil, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT \\* FROM oauth_providers WHERE provider = \\$1 AND provider_user_id = \\$2").
		WithArgs("github", "12345").
		WillReturnRows(rows)

	provider, err := repo.GetByProviderUserID(ctx, "github", "12345")
	require.NoError(t, err)
	assert.Equal(t, providerID, provider.ID)
	assert.Equal(t, "github", provider.Provider)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_GetByProviderUserID_NotFound(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT \\* FROM oauth_providers WHERE provider = \\$1 AND provider_user_id = \\$2").
		WithArgs("github", "unknown").
		WillReturnError(sql.ErrNoRows)

	provider, err := repo.GetByProviderUserID(ctx, "github", "unknown")
	assert.ErrorIs(t, err, services.ErrOAuthProviderNotFound)
	assert.Nil(t, provider)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_Update(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	username := "updated_user"
	displayName := "Updated User"
	avatarURL := "https://example.com/new-avatar.png"

	provider := &models.OAuthProvider{
		ID:          uuid.New(),
		Email:       "updated@example.com",
		Username:    &username,
		DisplayName: &displayName,
		AvatarURL:   &avatarURL,
	}

	mock.ExpectExec("UPDATE oauth_providers SET").
		WithArgs(
			provider.ID,
			provider.Email,
			provider.Username,
			provider.DisplayName,
			provider.AvatarURL,
			sqlmock.AnyArg(), // access_token
			sqlmock.AnyArg(), // refresh_token
			sqlmock.AnyArg(), // token_expires_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, provider)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_Delete(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()

	mock.ExpectExec("DELETE FROM oauth_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, providerID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_Delete_NotFound(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()

	mock.ExpectExec("DELETE FROM oauth_providers WHERE id = \\$1").
		WithArgs(providerID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(ctx, providerID)
	assert.ErrorIs(t, err, services.ErrOAuthProviderNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_DeleteByUserAndProvider(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM oauth_providers WHERE user_id = \\$1 AND provider = \\$2").
		WithArgs(userID, "github").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteByUserAndProvider(ctx, userID, "github")
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_CountByUserID(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(3)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM oauth_providers WHERE user_id = \\$1").
		WithArgs(userID).
		WillReturnRows(rows)

	count, err := repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_ExistsByProviderUserID(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("github", "12345").
		WillReturnRows(rows)

	exists, err := repo.ExistsByProviderUserID(ctx, "github", "12345")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_GetUserIDByProviderUserID(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	rows := sqlmock.NewRows([]string{"user_id"}).AddRow(userID)

	mock.ExpectQuery("SELECT user_id FROM oauth_providers WHERE provider = \\$1 AND provider_user_id = \\$2").
		WithArgs("github", "12345").
		WillReturnRows(rows)

	resultID, err := repo.GetUserIDByProviderUserID(ctx, "github", "12345")
	require.NoError(t, err)
	assert.Equal(t, userID, resultID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOAuthRepository_GetByUserAndProvider(t *testing.T) {
	repo, mock := setupOAuthRepoMock(t)
	ctx := context.Background()

	providerID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider", "provider_user_id", "email",
		"username", "display_name", "avatar_url", "access_token", "refresh_token",
		"token_expires_at", "created_at", "updated_at",
	}).AddRow(
		providerID, userID, "google", "67890", "test@example.com",
		nil, nil, nil, nil, nil, nil, now, now,
	)

	mock.ExpectQuery("SELECT \\* FROM oauth_providers WHERE user_id = \\$1 AND provider = \\$2").
		WithArgs(userID, "google").
		WillReturnRows(rows)

	provider, err := repo.GetByUserAndProvider(ctx, userID, "google")
	require.NoError(t, err)
	assert.Equal(t, providerID, provider.ID)
	assert.Equal(t, "google", provider.Provider)
	assert.NoError(t, mock.ExpectationsWereMet())
}
