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
)

func setupFollowRepoMock(t *testing.T) (*FollowRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewFollowRepository(sqlxDB)
	return repo, mock
}

func TestFollowRepository_Create(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	followerChannelID := uuid.New()
	createdAt := time.Now()

	follow := &models.FollowedChannel{
		ChannelID:         channelID,
		FollowerChannelID: followerChannelID,
		CreatedAt:         createdAt,
	}

	mock.ExpectExec("INSERT INTO followed_channels").
		WithArgs(follow.ChannelID, follow.FollowerChannelID, follow.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, follow)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_Create_Error(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	follow := &models.FollowedChannel{
		ChannelID:         uuid.New(),
		FollowerChannelID: uuid.New(),
		CreatedAt:         time.Now(),
	}

	mock.ExpectExec("INSERT INTO followed_channels").
		WillReturnError(sql.ErrConnDone)

	err := repo.Create(ctx, follow)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_Delete(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	followerChannelID := uuid.New()

	mock.ExpectExec("DELETE FROM followed_channels").
		WithArgs(channelID, followerChannelID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, channelID, followerChannelID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_Delete_Error(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	followerChannelID := uuid.New()

	mock.ExpectExec("DELETE FROM followed_channels").
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(ctx, channelID, followerChannelID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_GetByChannelAndFollower(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	followerChannelID := uuid.New()
	createdAt := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{"channel_id", "follower_channel_id", "created_at"}).
		AddRow(channelID, followerChannelID, createdAt)

	mock.ExpectQuery("SELECT channel_id, follower_channel_id, created_at FROM followed_channels").
		WithArgs(channelID, followerChannelID).
		WillReturnRows(rows)

	follow, err := repo.GetByChannelAndFollower(ctx, channelID, followerChannelID)
	assert.NoError(t, err)
	assert.NotNil(t, follow)
	assert.Equal(t, channelID, follow.ChannelID)
	assert.Equal(t, followerChannelID, follow.FollowerChannelID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_GetByChannelAndFollower_NotFound(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	followerChannelID := uuid.New()

	mock.ExpectQuery("SELECT channel_id, follower_channel_id, created_at FROM followed_channels").
		WithArgs(channelID, followerChannelID).
		WillReturnError(sql.ErrNoRows)

	follow, err := repo.GetByChannelAndFollower(ctx, channelID, followerChannelID)
	assert.NoError(t, err)
	assert.Nil(t, follow)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_GetByChannelAndFollower_Error(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	followerChannelID := uuid.New()

	mock.ExpectQuery("SELECT channel_id, follower_channel_id, created_at FROM followed_channels").
		WithArgs(channelID, followerChannelID).
		WillReturnError(sql.ErrConnDone)

	follow, err := repo.GetByChannelAndFollower(ctx, channelID, followerChannelID)
	assert.Error(t, err)
	assert.Nil(t, follow)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_GetFollowers(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{"channel_id", "follower_channel_id", "created_at"}).
		AddRow(channelID, uuid.New(), now).
		AddRow(channelID, uuid.New(), now.Add(time.Second)).
		AddRow(channelID, uuid.New(), now.Add(2*time.Second))

	mock.ExpectQuery("SELECT channel_id, follower_channel_id, created_at FROM followed_channels").
		WithArgs(channelID).
		WillReturnRows(rows)

	follows, err := repo.GetFollowers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, follows, 3)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_GetFollowers_Empty(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	rows := sqlmock.NewRows([]string{"channel_id", "follower_channel_id", "created_at"})

	mock.ExpectQuery("SELECT channel_id, follower_channel_id, created_at FROM followed_channels").
		WithArgs(channelID).
		WillReturnRows(rows)

	follows, err := repo.GetFollowers(ctx, channelID)
	assert.NoError(t, err)
	assert.Len(t, follows, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFollowRepository_GetFollowers_Error(t *testing.T) {
	repo, mock := setupFollowRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	mock.ExpectQuery("SELECT channel_id, follower_channel_id, created_at FROM followed_channels").
		WithArgs(channelID).
		WillReturnError(sql.ErrConnDone)

	follows, err := repo.GetFollowers(ctx, channelID)
	assert.Error(t, err)
	assert.Nil(t, follows)
	assert.NoError(t, mock.ExpectationsWereMet())
}
