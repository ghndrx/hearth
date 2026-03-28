package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func setupChannelRepoMock(t *testing.T) (*ChannelRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewChannelRepository(sqlxDB)
	return repo, mock
}

func TestChannelRepository_Create(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	channel := &models.Channel{
		ID:          uuid.New(),
		ServerID:    &serverID,
		Name:        "general",
		Topic:       "General discussion",
		Type:        models.ChannelTypeText,
		Position:    0,
		Slowmode:    0,
		NSFW:        false,
		E2EEEnabled: false,
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO channels").
		WithArgs(
			channel.ID, channel.ServerID, channel.Name, channel.Topic, channel.Type,
			channel.Position, channel.ParentID, channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
			channel.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, channel)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_Create_WithRecipients(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	channel := &models.Channel{
		ID:          uuid.New(),
		Type:        models.ChannelTypeDM,
		E2EEEnabled: true,
		Recipients:  []uuid.UUID{user1, user2},
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO channels").
		WithArgs(
			channel.ID, channel.ServerID, channel.Name, channel.Topic, channel.Type,
			channel.Position, channel.ParentID, channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
			channel.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO channel_recipients").
		WithArgs(channel.ID, user1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO channel_recipients").
		WithArgs(channel.ID, user2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, channel)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_Create_RecipientInsertFails(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	channel := &models.Channel{
		ID:          uuid.New(),
		Type:        models.ChannelTypeDM,
		E2EEEnabled: true,
		Recipients:  []uuid.UUID{user1, user2},
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO channels").
		WithArgs(
			channel.ID, channel.ServerID, channel.Name, channel.Topic, channel.Type,
			channel.Position, channel.ParentID, channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
			channel.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// First recipient succeeds
	mock.ExpectExec("INSERT INTO channel_recipients").
		WithArgs(channel.ID, user1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second recipient fails
	mock.ExpectExec("INSERT INTO channel_recipients").
		WithArgs(channel.ID, user2).
		WillReturnError(fmt.Errorf("unique constraint violation"))

	err := repo.Create(ctx, channel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 recipient insert(s) failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_Create_AllRecipientsInsertFail(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	channel := &models.Channel{
		ID:          uuid.New(),
		Type:        models.ChannelTypeDM,
		E2EEEnabled: true,
		Recipients:  []uuid.UUID{user1, user2},
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO channels").
		WithArgs(
			channel.ID, channel.ServerID, channel.Name, channel.Topic, channel.Type,
			channel.Position, channel.ParentID, channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
			channel.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO channel_recipients").
		WithArgs(channel.ID, user1).
		WillReturnError(fmt.Errorf("db error"))

	mock.ExpectExec("INSERT INTO channel_recipients").
		WithArgs(channel.ID, user2).
		WillReturnError(fmt.Errorf("db error"))

	err := repo.Create(ctx, channel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 recipient insert(s) failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_Create_ChannelInsertFails(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channel := &models.Channel{
		ID:        uuid.New(),
		Type:      models.ChannelTypeText,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO channels").
		WithArgs(
			channel.ID, channel.ServerID, channel.Name, channel.Topic, channel.Type,
			channel.Position, channel.ParentID, channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
			channel.CreatedAt,
		).
		WillReturnError(fmt.Errorf("insert failed"))

	err := repo.Create(ctx, channel)
	require.Error(t, err)
	assert.Equal(t, "insert failed", err.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

var channelColumns = []string{
	"id", "server_id", "parent_id", "owner_id", "type", "name", "topic",
	"position", "slowmode", "nsfw", "e2ee_enabled", "bitrate", "user_limit",
	"rtc_region", "last_message_id", "created_at",
}

func TestChannelRepository_GetByID(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	serverID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(channelColumns).AddRow(
		channelID, serverID, nil, nil, models.ChannelTypeText, "general", "General chat",
		0, 0, false, false, nil, nil,
		nil, nil, now,
	)

	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnRows(rows)

	channel, err := repo.GetByID(ctx, channelID)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, channelID, channel.ID)
	assert.Equal(t, "general", channel.Name)
	assert.Equal(t, models.ChannelTypeText, channel.Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnError(sql.ErrNoRows)

	channel, err := repo.GetByID(ctx, channelID)
	assert.NoError(t, err)
	assert.Nil(t, channel)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetByID_DBError(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnError(fmt.Errorf("connection refused"))

	channel, err := repo.GetByID(ctx, channelID)
	require.Error(t, err)
	assert.Nil(t, channel)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetByID_DMLoadsRecipients(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	now := time.Now()

	channelRows := sqlmock.NewRows(channelColumns).AddRow(
		channelID, nil, nil, nil, models.ChannelTypeDM, "", "",
		0, 0, false, true, nil, nil,
		nil, nil, now,
	)

	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnRows(channelRows)

	recipientRows := sqlmock.NewRows([]string{"user_id"}).
		AddRow(user1).
		AddRow(user2)

	mock.ExpectQuery("SELECT user_id FROM channel_recipients WHERE channel_id = \\$1").
		WithArgs(channelID).
		WillReturnRows(recipientRows)

	channel, err := repo.GetByID(ctx, channelID)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, models.ChannelTypeDM, channel.Type)
	assert.Len(t, channel.Recipients, 2)
	assert.Contains(t, channel.Recipients, user1)
	assert.Contains(t, channel.Recipients, user2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetByID_GroupDMLoadsRecipients(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	ownerID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	now := time.Now()

	channelRows := sqlmock.NewRows(channelColumns).AddRow(
		channelID, nil, nil, &ownerID, models.ChannelTypeGroupDM, "Group Chat", "",
		0, 0, false, false, nil, nil,
		nil, nil, now,
	)

	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnRows(channelRows)

	recipientRows := sqlmock.NewRows([]string{"user_id"}).
		AddRow(ownerID).
		AddRow(user1).
		AddRow(user2)

	mock.ExpectQuery("SELECT user_id FROM channel_recipients WHERE channel_id = \\$1").
		WithArgs(channelID).
		WillReturnRows(recipientRows)

	channel, err := repo.GetByID(ctx, channelID)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, models.ChannelTypeGroupDM, channel.Type)
	assert.Len(t, channel.Recipients, 3)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetByID_RecipientLoadFails(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	now := time.Now()

	channelRows := sqlmock.NewRows(channelColumns).AddRow(
		channelID, nil, nil, nil, models.ChannelTypeDM, "", "",
		0, 0, false, true, nil, nil,
		nil, nil, now,
	)

	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnRows(channelRows)

	mock.ExpectQuery("SELECT user_id FROM channel_recipients WHERE channel_id = \\$1").
		WithArgs(channelID).
		WillReturnError(fmt.Errorf("recipient table missing"))

	channel, err := repo.GetByID(ctx, channelID)
	require.Error(t, err)
	assert.Nil(t, channel)
	assert.Contains(t, err.Error(), "failed to load recipients")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetByID_TextChannelSkipsRecipients(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	serverID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(channelColumns).AddRow(
		channelID, serverID, nil, nil, models.ChannelTypeText, "general", "",
		0, 0, false, false, nil, nil,
		nil, nil, now,
	)

	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnRows(rows)

	// No recipient query expected for text channels

	channel, err := repo.GetByID(ctx, channelID)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Empty(t, channel.Recipients)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_Update(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channel := &models.Channel{
		ID:          uuid.New(),
		Name:        "updated-name",
		Topic:       "new topic",
		Position:    3,
		Slowmode:    5,
		NSFW:        true,
		E2EEEnabled: true,
	}

	mock.ExpectExec("UPDATE channels SET").
		WithArgs(
			channel.ID, channel.Name, channel.Topic, channel.Position, channel.ParentID,
			channel.Slowmode, channel.NSFW, channel.E2EEEnabled,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, channel)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_Delete(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()

	mock.ExpectExec("DELETE FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, channelID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetByServerID(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	ch1 := uuid.New()
	ch2 := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows(channelColumns).
		AddRow(ch1, serverID, nil, nil, models.ChannelTypeText, "general", "", 0, 0, false, false, nil, nil, nil, nil, now).
		AddRow(ch2, serverID, nil, nil, models.ChannelTypeText, "random", "", 1, 0, false, false, nil, nil, nil, nil, now)

	mock.ExpectQuery("SELECT \\* FROM channels WHERE server_id = \\$1 ORDER BY position").
		WithArgs(serverID).
		WillReturnRows(rows)

	channels, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, channels, 2)
	assert.Equal(t, "general", channels[0].Name)
	assert.Equal(t, "random", channels[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetUserDMs(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	ch1 := uuid.New()
	ch2 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	now := time.Now()

	dmColumns := []string{
		"id", "server_id", "parent_id", "owner_id", "type", "name", "topic",
		"position", "slowmode", "nsfw", "e2ee_enabled", "bitrate", "user_limit",
		"rtc_region", "last_message_id", "created_at",
	}

	channelRows := sqlmock.NewRows(dmColumns).
		AddRow(ch1, nil, nil, nil, models.ChannelTypeDM, "", "", 0, 0, false, true, nil, nil, nil, nil, now).
		AddRow(ch2, nil, nil, nil, models.ChannelTypeGroupDM, "Group", "", 0, 0, false, false, nil, nil, nil, nil, now)

	mock.ExpectQuery("SELECT .+ FROM channels c").
		WithArgs(userID, models.ChannelTypeDM, models.ChannelTypeGroupDM).
		WillReturnRows(channelRows)

	// Recipients for ch1
	recipientRows1 := sqlmock.NewRows([]string{"user_id"}).
		AddRow(userID).
		AddRow(user2)

	mock.ExpectQuery("SELECT user_id FROM channel_recipients WHERE channel_id = \\$1").
		WithArgs(ch1).
		WillReturnRows(recipientRows1)

	// Recipients for ch2
	recipientRows2 := sqlmock.NewRows([]string{"user_id"}).
		AddRow(userID).
		AddRow(user2).
		AddRow(user3)

	mock.ExpectQuery("SELECT user_id FROM channel_recipients WHERE channel_id = \\$1").
		WithArgs(ch2).
		WillReturnRows(recipientRows2)

	channels, err := repo.GetUserDMs(ctx, userID)
	require.NoError(t, err)
	require.Len(t, channels, 2)
	assert.Len(t, channels[0].Recipients, 2)
	assert.Len(t, channels[1].Recipients, 3)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetUserDMs_RecipientLoadFails(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	ch1 := uuid.New()
	now := time.Now()

	dmColumns := []string{
		"id", "server_id", "parent_id", "owner_id", "type", "name", "topic",
		"position", "slowmode", "nsfw", "e2ee_enabled", "bitrate", "user_limit",
		"rtc_region", "last_message_id", "created_at",
	}

	channelRows := sqlmock.NewRows(dmColumns).
		AddRow(ch1, nil, nil, nil, models.ChannelTypeDM, "", "", 0, 0, false, true, nil, nil, nil, nil, now)

	mock.ExpectQuery("SELECT .+ FROM channels c").
		WithArgs(userID, models.ChannelTypeDM, models.ChannelTypeGroupDM).
		WillReturnRows(channelRows)

	mock.ExpectQuery("SELECT user_id FROM channel_recipients WHERE channel_id = \\$1").
		WithArgs(ch1).
		WillReturnError(fmt.Errorf("connection lost"))

	channels, err := repo.GetUserDMs(ctx, userID)
	require.Error(t, err)
	assert.Nil(t, channels)
	assert.Contains(t, err.Error(), "failed to load recipients")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetUserDMs_QueryFails(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM channels c").
		WithArgs(userID, models.ChannelTypeDM, models.ChannelTypeGroupDM).
		WillReturnError(fmt.Errorf("query error"))

	channels, err := repo.GetUserDMs(ctx, userID)
	require.Error(t, err)
	assert.Nil(t, channels)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_UpdateLastMessage(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	messageID := uuid.New()
	now := time.Now()

	mock.ExpectExec("UPDATE channels SET last_message_id").
		WithArgs(channelID, messageID, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastMessage(ctx, channelID, messageID, now)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetDMChannel(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()
	channelID := uuid.New()
	now := time.Now()

	// First query: find channel ID
	idRows := sqlmock.NewRows([]string{"id"}).AddRow(channelID)
	mock.ExpectQuery("SELECT c.id FROM channels c").
		WithArgs(user1, user2, models.ChannelTypeDM).
		WillReturnRows(idRows)

	// Second query: GetByID
	channelRows := sqlmock.NewRows(channelColumns).AddRow(
		channelID, nil, nil, nil, models.ChannelTypeDM, "", "",
		0, 0, false, true, nil, nil,
		nil, nil, now,
	)
	mock.ExpectQuery("SELECT \\* FROM channels WHERE id = \\$1").
		WithArgs(channelID).
		WillReturnRows(channelRows)

	// Load recipients for DM
	recipientRows := sqlmock.NewRows([]string{"user_id"}).
		AddRow(user1).
		AddRow(user2)
	mock.ExpectQuery("SELECT user_id FROM channel_recipients WHERE channel_id = \\$1").
		WithArgs(channelID).
		WillReturnRows(recipientRows)

	channel, err := repo.GetDMChannel(ctx, user1, user2)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, channelID, channel.ID)
	assert.Len(t, channel.Recipients, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetDMChannel_NotFound(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	mock.ExpectQuery("SELECT c.id FROM channels c").
		WithArgs(user1, user2, models.ChannelTypeDM).
		WillReturnError(sql.ErrNoRows)

	channel, err := repo.GetDMChannel(ctx, user1, user2)
	assert.NoError(t, err)
	assert.Nil(t, channel)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelRepository_GetPermissionOverrides(t *testing.T) {
	repo, mock := setupChannelRepoMock(t)
	ctx := context.Background()

	channelID := uuid.New()
	targetID := uuid.New()

	rows := sqlmock.NewRows([]string{"channel_id", "target_type", "target_id", "allow", "deny"}).
		AddRow(channelID, "role", targetID, int64(1024), int64(0))

	mock.ExpectQuery("SELECT channel_id, target_type, target_id, allow, deny FROM permission_overrides WHERE channel_id = \\$1").
		WithArgs(channelID).
		WillReturnRows(rows)

	overrides, err := repo.GetPermissionOverrides(ctx, channelID)
	require.NoError(t, err)
	require.Len(t, overrides, 1)
	assert.Equal(t, channelID, overrides[0].ChannelID)
	assert.Equal(t, "role", overrides[0].TargetType)
	assert.Equal(t, int64(1024), overrides[0].Allow)
	assert.NoError(t, mock.ExpectationsWereMet())
}
