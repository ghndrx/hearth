package postgres

import (
	"context"
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

func setupDiscoverableServerRepoMock(t *testing.T) (*DiscoverableServerRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewDiscoverableServerRepository(sqlxDB)
	return repo, mock
}

func TestDiscoverableServerRepository_GetByID(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "name", "description", "category",
		"icon_url", "banner_url", "tags", "member_count",
		"is_verified", "is_public", "is_featured", "created_at", "updated_at",
	}).AddRow(
		id, serverID, "Test Server", "A test server", models.DiscoveryCategoryGaming,
		"https://example.com/icon.png", "https://example.com/banner.png", pq.StringArray{"test"}, 100,
		false, true, false, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(id).WillReturnRows(rows)

	retrieved, err := repo.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, id, retrieved.ID)
	assert.Equal(t, serverID, retrieved.ServerID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscoverableServerRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()
	id := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "name", "description", "category",
		"icon_url", "banner_url", "tags", "member_count",
		"is_verified", "is_public", "is_featured", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT").WithArgs(id).WillReturnRows(rows)

	retrieved, err := repo.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscoverableServerRepository_GetByServerID(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()
	id := uuid.New()
	serverID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "name", "description", "category",
		"icon_url", "banner_url", "tags", "member_count",
		"is_verified", "is_public", "is_featured", "created_at", "updated_at",
	}).AddRow(
		id, serverID, "Test Server", "A test server", models.DiscoveryCategoryMusic,
		"https://example.com/icon.png", nil, pq.StringArray{"music"}, 50,
		false, true, false, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(serverID).WillReturnRows(rows)

	retrieved, err := repo.GetByServerID(ctx, serverID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, serverID, retrieved.ServerID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscoverableServerRepository_GetByServerID_NotFound(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "name", "description", "category",
		"icon_url", "banner_url", "tags", "member_count",
		"is_verified", "is_public", "is_featured", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT").WithArgs(serverID).WillReturnRows(rows)

	retrieved, err := repo.GetByServerID(ctx, serverID)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscoverableServerRepository_GetFeaturedServers(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "name", "description", "category",
		"icon_url", "banner_url", "member_count",
		"is_verified", "featured_at", "created_at",
	}).AddRow(
		uuid.New(), uuid.New(), "Featured Server", "A featured server", models.DiscoveryCategoryGaming,
		"https://example.com/icon.png", nil, 500,
		true, now, now,
	)

	mock.ExpectQuery("SELECT").WithArgs(10).WillReturnRows(rows)

	featured, err := repo.GetFeaturedServers(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, featured, 1)
	assert.Equal(t, "Featured Server", featured[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscoverableServerRepository_GetFeaturedServers_Empty(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{
		"id", "server_id", "name", "description", "category",
		"icon_url", "banner_url", "member_count",
		"is_verified", "featured_at", "created_at",
	})

	mock.ExpectQuery("SELECT").WithArgs(5).WillReturnRows(rows)

	featured, err := repo.GetFeaturedServers(ctx, 5)
	assert.NoError(t, err)
	assert.Len(t, featured, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscoverableServerRepository_GetInviteCode(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"code",
	}).AddRow("ABC123")

	mock.ExpectQuery("SELECT").WithArgs(serverID).WillReturnRows(rows)

	code, err := repo.GetInviteCode(ctx, serverID)
	assert.NoError(t, err)
	assert.Equal(t, "ABC123", code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDiscoverableServerRepository_GetInviteCode_NotFound(t *testing.T) {
	repo, mock := setupDiscoverableServerRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"code",
	})

	mock.ExpectQuery("SELECT").WithArgs(serverID).WillReturnRows(rows)

	code, err := repo.GetInviteCode(ctx, serverID)
	assert.NoError(t, err)
	assert.Equal(t, "", code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
