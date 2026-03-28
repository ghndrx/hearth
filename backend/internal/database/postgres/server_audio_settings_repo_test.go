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

func setupServerAudioSettingsRepoMock(t *testing.T) (*ServerAudioSettingsRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewServerAudioSettingsRepository(sqlxDB)
	return repo, mock
}

func TestServerAudioSettingsRepository_Get(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	serverID := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"user_id", "server_id", "input_device_id", "output_device_id",
		"input_volume", "output_volume", "push_to_talk_enabled", "push_to_talk_key", "updated_at",
	}).AddRow(
		userID, serverID, "input-123", "output-456",
		80, 90, true, "V", now,
	)

	mock.ExpectQuery("SELECT user_id, server_id, input_device_id, output_device_id").
		WithArgs(userID, serverID).
		WillReturnRows(rows)

	settings, err := repo.Get(ctx, userID, serverID)
	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, userID, settings.UserID)
	assert.Equal(t, serverID, settings.ServerID)
	assert.Equal(t, "input-123", settings.InputDeviceID)
	assert.Equal(t, "output-456", settings.OutputDeviceID)
	assert.Equal(t, 80, settings.InputVolume)
	assert.Equal(t, 90, settings.OutputVolume)
	assert.True(t, settings.PushToTalkEnabled)
	assert.Equal(t, "V", settings.PushToTalkKey)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_Get_NotFound(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	serverID := uuid.New()

	mock.ExpectQuery("SELECT user_id, server_id, input_device_id, output_device_id").
		WithArgs(userID, serverID).
		WillReturnError(sql.ErrNoRows)

	settings, err := repo.Get(ctx, userID, serverID)
	assert.NoError(t, err)
	assert.Nil(t, settings)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_Get_Error(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	serverID := uuid.New()

	mock.ExpectQuery("SELECT user_id, server_id, input_device_id, output_device_id").
		WithArgs(userID, serverID).
		WillReturnError(sql.ErrConnDone)

	settings, err := repo.Get(ctx, userID, serverID)
	// The implementation returns the empty struct along with the error
	assert.Error(t, err)
	assert.NotNil(t, settings) // returns empty struct, not nil
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_GetAllForUser(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	serverID1 := uuid.New()
	serverID2 := uuid.New()
	now := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"user_id", "server_id", "input_device_id", "output_device_id",
		"input_volume", "output_volume", "push_to_talk_enabled", "push_to_talk_key", "updated_at",
	}).
		AddRow(userID, serverID1, "input-1", "output-1", 100, 100, false, "", now).
		AddRow(userID, serverID2, "input-2", "output-2", 75, 85, true, "B", now)

	mock.ExpectQuery("SELECT user_id, server_id, input_device_id, output_device_id").
		WithArgs(userID).
		WillReturnRows(rows)

	settings, err := repo.GetAllForUser(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, settings, 2)
	assert.Equal(t, serverID1, settings[0].ServerID)
	assert.Equal(t, serverID2, settings[1].ServerID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_GetAllForUser_Empty(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"user_id", "server_id", "input_device_id", "output_device_id",
		"input_volume", "output_volume", "push_to_talk_enabled", "push_to_talk_key", "updated_at",
	})

	mock.ExpectQuery("SELECT user_id, server_id, input_device_id, output_device_id").
		WithArgs(userID).
		WillReturnRows(rows)

	settings, err := repo.GetAllForUser(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, settings, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_GetAllForUser_Error(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT user_id, server_id, input_device_id, output_device_id").
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	settings, err := repo.GetAllForUser(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, settings)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_Upsert(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	serverID := uuid.New()
	settings := &models.ServerAudioSettings{
		UserID:            userID,
		ServerID:          serverID,
		InputDeviceID:     "mic-usb",
		OutputDeviceID:    "speakers",
		InputVolume:       50,
		OutputVolume:      70,
		PushToTalkEnabled: true,
		PushToTalkKey:     "LeftControl",
		UpdatedAt:         time.Now(),
	}

	mock.ExpectExec("INSERT INTO server_audio_settings").
		WithArgs(
			settings.UserID, settings.ServerID, settings.InputDeviceID, settings.OutputDeviceID,
			settings.InputVolume, settings.OutputVolume, settings.PushToTalkEnabled, settings.PushToTalkKey,
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Upsert(ctx, settings)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_Upsert_Error(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	settings := &models.ServerAudioSettings{
		UserID:            uuid.New(),
		ServerID:          uuid.New(),
		InputDeviceID:     "mic",
		OutputDeviceID:    "spk",
		InputVolume:       100,
		OutputVolume:      100,
		PushToTalkEnabled: false,
		PushToTalkKey:     "",
		UpdatedAt:         time.Now(),
	}

	mock.ExpectExec("INSERT INTO server_audio_settings").
		WillReturnError(sql.ErrConnDone)

	err := repo.Upsert(ctx, settings)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_Delete(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	serverID := uuid.New()

	mock.ExpectExec("DELETE FROM server_audio_settings").
		WithArgs(userID, serverID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, userID, serverID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestServerAudioSettingsRepository_Delete_Error(t *testing.T) {
	repo, mock := setupServerAudioSettingsRepoMock(t)
	ctx := context.Background()

	userID := uuid.New()
	serverID := uuid.New()

	mock.ExpectExec("DELETE FROM server_audio_settings").
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(ctx, userID, serverID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
