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

func setupWebhookRepoMock(t *testing.T) (*WebhookRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewWebhookRepository(sqlxDB)
	return repo, mock
}

func TestWebhookRepository_Create(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhook := &models.Webhook{
		ID:        uuid.New(),
		Type:      models.WebhookTypeIncoming,
		ChannelID: uuid.New(),
		Name:      "test-webhook",
		Token:     "test-token",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO webhooks").
		WithArgs(
			webhook.ID, webhook.Type, webhook.ServerID, webhook.ChannelID,
			webhook.CreatorID, webhook.Name, webhook.Avatar, webhook.Token,
			webhook.ApplicationID, webhook.SourceServerID, webhook.SourceChannelID,
			webhook.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_Create_WithOptionalFields(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	creatorID := uuid.New()
	avatar := "default avatar"
	webhook := &models.Webhook{
		ID:              uuid.New(),
		Type:            models.WebhookTypeIncoming,
		ServerID:        &serverID,
		ChannelID:       uuid.New(),
		CreatorID:       &creatorID,
		Name:            "test-webhook",
		Avatar:          &avatar,
		Token:           "test-token",
		ApplicationID:   nil,
		SourceServerID:  nil,
		SourceChannelID: nil,
		CreatedAt:       time.Now(),
	}

	mock.ExpectExec("INSERT INTO webhooks").
		WithArgs(
			webhook.ID, webhook.Type, webhook.ServerID, webhook.ChannelID,
			webhook.CreatorID, webhook.Name, webhook.Avatar, webhook.Token,
			webhook.ApplicationID, webhook.SourceServerID, webhook.SourceChannelID,
			webhook.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, webhook)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByID(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhookID := uuid.New()
	channelID := uuid.New()
	createdAt := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "type", "server_id", "channel_id", "creator_id",
		"name", "avatar", "token", "application_id",
		"source_server_id", "source_channel_id", "created_at",
	}).AddRow(
		webhookID, models.WebhookTypeIncoming, nil, channelID,
		nil, "test-webhook", nil, "test-token",
		nil, nil, nil, createdAt,
	)

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE id").
		WithArgs(webhookID).
		WillReturnRows(rows)

	webhook, err := repo.GetByID(ctx, webhookID)
	require.NoError(t, err)
	require.NotNil(t, webhook)
	assert.Equal(t, webhookID, webhook.ID)
	assert.Equal(t, "test-webhook", webhook.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhookID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE id").
		WithArgs(webhookID).
		WillReturnError(sql.ErrNoRows)

	webhook, err := repo.GetByID(ctx, webhookID)
	require.NoError(t, err)
	require.Nil(t, webhook)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByID_Error(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhookID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE id").
		WithArgs(webhookID).
		WillReturnError(sql.ErrConnDone)

	webhook, err := repo.GetByID(ctx, webhookID)
	require.Error(t, err)
	require.Nil(t, webhook)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByChannelID(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	webhookID1 := uuid.New()
	webhookID2 := uuid.New()
	createdAt := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "type", "server_id", "channel_id", "creator_id",
		"name", "avatar", "token", "application_id",
		"source_server_id", "source_channel_id", "created_at",
	}).
		AddRow(webhookID1, models.WebhookTypeIncoming, nil, channelID, nil, "webhook-1", nil, "token-1", nil, nil, nil, createdAt).
		AddRow(webhookID2, models.WebhookTypeApplication, nil, channelID, nil, "webhook-2", nil, "token-2", nil, nil, nil, createdAt)

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE channel_id").
		WithArgs(channelID).
		WillReturnRows(rows)

	webhooks, err := repo.GetByChannelID(ctx, channelID)
	require.NoError(t, err)
	assert.Len(t, webhooks, 2)
	assert.Equal(t, "webhook-1", webhooks[0].Name)
	assert.Equal(t, "webhook-2", webhooks[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByChannelID_Empty(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "type", "server_id", "channel_id", "creator_id",
		"name", "avatar", "token", "application_id",
		"source_server_id", "source_channel_id", "created_at",
	})

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE channel_id").
		WithArgs(channelID).
		WillReturnRows(rows)

	webhooks, err := repo.GetByChannelID(ctx, channelID)
	require.NoError(t, err)
	assert.Len(t, webhooks, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByServerID(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	webhookID := uuid.New()
	createdAt := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "type", "server_id", "channel_id", "creator_id",
		"name", "avatar", "token", "application_id",
		"source_server_id", "source_channel_id", "created_at",
	}).AddRow(webhookID, models.WebhookTypeIncoming, serverID, channelID, nil, "server-webhook", nil, "token", nil, nil, nil, createdAt)

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE server_id").
		WithArgs(serverID).
		WillReturnRows(rows)

	webhooks, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, webhooks, 1)
	assert.Equal(t, webhookID, webhooks[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_Update(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhookID := uuid.New()
	channelID := uuid.New()
	newName := "updated-webhook"
	newAvatar := "new-avatar"

	webhook := &models.Webhook{
		ID:        webhookID,
		Name:      newName,
		Avatar:    &newAvatar,
		ChannelID: channelID,
	}

	mock.ExpectExec("UPDATE webhooks").
		WithArgs(webhook.ID, webhook.Name, webhook.Avatar, webhook.ChannelID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, webhook)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_Update_NotFound(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhook := &models.Webhook{
		ID:        uuid.New(),
		Name:      "updated-webhook",
		ChannelID: uuid.New(),
	}

	mock.ExpectExec("UPDATE webhooks").
		WithArgs(webhook.ID, webhook.Name, webhook.Avatar, webhook.ChannelID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Update(ctx, webhook)
	require.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_Delete(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhookID := uuid.New()

	mock.ExpectExec("DELETE FROM webhooks WHERE id").
		WithArgs(webhookID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, webhookID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_Delete_NotFound(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhookID := uuid.New()

	mock.ExpectExec("DELETE FROM webhooks WHERE id").
		WithArgs(webhookID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(ctx, webhookID)
	require.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_CountByChannelID(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(channelID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.CountByChannelID(ctx, channelID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByToken(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	webhookID := uuid.New()
	channelID := uuid.New()
	token := "test-token-123"
	createdAt := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "type", "server_id", "channel_id", "creator_id",
		"name", "avatar", "token", "application_id",
		"source_server_id", "source_channel_id", "created_at",
	}).AddRow(webhookID, models.WebhookTypeIncoming, nil, channelID, nil, "token-webhook", nil, token, nil, nil, nil, createdAt)

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE token").
		WithArgs(token).
		WillReturnRows(rows)

	webhook, err := repo.GetByToken(ctx, token)
	require.NoError(t, err)
	require.NotNil(t, webhook)
	assert.Equal(t, webhookID, webhook.ID)
	assert.Equal(t, token, webhook.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByToken_NotFound(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	token := "invalid-token"

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE token").
		WithArgs(token).
		WillReturnError(sql.ErrNoRows)

	webhook, err := repo.GetByToken(ctx, token)
	require.NoError(t, err)
	require.Nil(t, webhook)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByChannelID_Error(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE channel_id").
		WithArgs(channelID).
		WillReturnError(sql.ErrConnDone)

	webhooks, err := repo.GetByChannelID(ctx, channelID)
	require.Error(t, err)
	require.Nil(t, webhooks)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRepository_GetByServerID_Error(t *testing.T) {
	repo, mock := setupWebhookRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM webhooks WHERE server_id").
		WithArgs(serverID).
		WillReturnError(sql.ErrConnDone)

	webhooks, err := repo.GetByServerID(ctx, serverID)
	require.Error(t, err)
	require.Nil(t, webhooks)
	assert.NoError(t, mock.ExpectationsWereMet())
}
