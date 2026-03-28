package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
)

func setupInviteRepoMock(t *testing.T) (*InviteRepo, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	repo := NewInviteRepo(db)
	return repo, mock
}

func setupBanRepoMock(t *testing.T) (*BanRepo, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	repo := NewBanRepo(db)
	return repo, mock
}

// --- InviteRepo Tests ---

func TestInviteRepo_Create(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	invite := &models.Invite{
		Code:      "TESTCODE",
		ServerID:  uuid.New(),
		ChannelID: uuid.New(),
		CreatorID: uuid.New(),
		MaxUses:   10,
		Uses:      0,
		Temporary: false,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO invites").
		WithArgs(
			invite.Code, invite.ServerID, invite.ChannelID, invite.CreatorID,
			invite.MaxUses, invite.Uses, invite.ExpiresAt, invite.Temporary, invite.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, invite)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_Create_WithExpiresAt(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	expiresAt := time.Now().Add(24 * time.Hour)
	invite := &models.Invite{
		Code:      "EXPIRING",
		ServerID:  uuid.New(),
		ChannelID: uuid.New(),
		CreatorID: uuid.New(),
		MaxUses:   5,
		Uses:      0,
		Temporary: true,
		ExpiresAt: &expiresAt,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO invites").
		WithArgs(
			invite.Code, invite.ServerID, invite.ChannelID, invite.CreatorID,
			invite.MaxUses, invite.Uses, invite.ExpiresAt, invite.Temporary, invite.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, invite)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_Create_DBError(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	invite := &models.Invite{
		Code:      "ERROR",
		ServerID:  uuid.New(),
		ChannelID: uuid.New(),
		CreatorID: uuid.New(),
		MaxUses:   10,
		Uses:      0,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO invites").
		WillReturnError(sql.ErrConnDone)

	err := repo.Create(ctx, invite)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_GetByCode(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "FINDME"
	serverID := uuid.New()
	channelID := uuid.New()
	creatorID := uuid.New()
	createdAt := time.Now()

	rows := sqlmock.NewRows([]string{
		"code", "server_id", "channel_id", "creator_id",
		"max_uses", "uses", "expires_at", "temporary", "created_at",
	}).AddRow(code, serverID, channelID, creatorID, 10, 2, nil, false, createdAt)

	mock.ExpectQuery("SELECT .+ FROM invites WHERE code = \\$1").
		WithArgs(code).
		WillReturnRows(rows)

	invite, err := repo.GetByCode(ctx, code)
	require.NoError(t, err)
	require.NotNil(t, invite)
	assert.Equal(t, code, invite.Code)
	assert.Equal(t, serverID, invite.ServerID)
	assert.Equal(t, channelID, invite.ChannelID)
	assert.Equal(t, creatorID, invite.CreatorID)
	assert.Equal(t, 10, invite.MaxUses)
	assert.Equal(t, 2, invite.Uses)
	assert.False(t, invite.Temporary)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_GetByCode_NotFound(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "NOTFOUND"
	mock.ExpectQuery("SELECT .+ FROM invites WHERE code = \\$1").
		WithArgs(code).
		WillReturnError(sql.ErrNoRows)

	invite, err := repo.GetByCode(ctx, code)
	require.NoError(t, err)
	assert.Nil(t, invite)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_GetByCode_DBError(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "ERROR"
	mock.ExpectQuery("SELECT .+ FROM invites WHERE code = \\$1").
		WithArgs(code).
		WillReturnError(sql.ErrConnDone)

	invite, err := repo.GetByCode(ctx, code)
	assert.Error(t, err)
	assert.Nil(t, invite)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_GetByServerID(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	rows := sqlmock.NewRows([]string{
		"code", "server_id", "channel_id", "creator_id",
		"max_uses", "uses", "expires_at", "temporary", "created_at",
	}).
		AddRow("CODE1", serverID, uuid.New(), uuid.New(), 10, 1, nil, false, time.Now()).
		AddRow("CODE2", serverID, uuid.New(), uuid.New(), 5, 0, nil, true, time.Now())

	mock.ExpectQuery("SELECT .+ FROM invites WHERE server_id = \\$1").
		WithArgs(serverID).
		WillReturnRows(rows)

	invites, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, invites, 2)
	assert.Equal(t, "CODE1", invites[0].Code)
	assert.Equal(t, "CODE2", invites[1].Code)
	assert.Equal(t, 1, invites[0].Uses)
	assert.Equal(t, 0, invites[1].Uses)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_GetByServerID_Empty(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	rows := sqlmock.NewRows([]string{
		"code", "server_id", "channel_id", "creator_id",
		"max_uses", "uses", "expires_at", "temporary", "created_at",
	})

	mock.ExpectQuery("SELECT .+ FROM invites WHERE server_id = \\$1").
		WithArgs(serverID).
		WillReturnRows(rows)

	invites, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, invites, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_GetByServerID_DBError(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	mock.ExpectQuery("SELECT .+ FROM invites WHERE server_id = \\$1").
		WithArgs(serverID).
		WillReturnError(sql.ErrConnDone)

	invites, err := repo.GetByServerID(ctx, serverID)
	assert.Error(t, err)
	assert.Nil(t, invites)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_IncrementUses(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "INCR"
	mock.ExpectExec("UPDATE invites SET uses = uses \\+ 1 WHERE code = \\$1").
		WithArgs(code).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.IncrementUses(ctx, code)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_IncrementUses_DBError(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "ERROR"
	mock.ExpectExec("UPDATE invites SET uses = uses \\+ 1 WHERE code = \\$1").
		WithArgs(code).
		WillReturnError(sql.ErrConnDone)

	err := repo.IncrementUses(ctx, code)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_Delete(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "DELETEME"
	mock.ExpectExec("DELETE FROM invites WHERE code = \\$1").
		WithArgs(code).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, code)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_Delete_NotFound(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "NOTFOUND"
	mock.ExpectExec("DELETE FROM invites WHERE code = \\$1").
		WithArgs(code).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(ctx, code)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_Delete_DBError(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	code := "ERROR"
	mock.ExpectExec("DELETE FROM invites WHERE code = \\$1").
		WithArgs(code).
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(ctx, code)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_DeleteExpired(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM invites WHERE expires_at IS NOT NULL AND expires_at < \\$1").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepo_DeleteExpired_DBError(t *testing.T) {
	repo, mock := setupInviteRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM invites WHERE expires_at IS NOT NULL AND expires_at < \\$1").
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	count, err := repo.DeleteExpired(ctx)
	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- BanRepo Tests ---

func TestBanRepo_Create(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	reason := "Spam"
	bannedBy := uuid.New()
	ban := &models.Ban{
		ServerID:  uuid.New(),
		UserID:    uuid.New(),
		Reason:    &reason,
		BannedBy:  &bannedBy,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO bans").
		WithArgs(ban.ServerID, ban.UserID, ban.Reason, ban.BannedBy, ban.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, ban)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_Create_Conflict(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	reason := "Updated reason"
	bannedBy := uuid.New()
	ban := &models.Ban{
		ServerID:  uuid.New(),
		UserID:    uuid.New(),
		Reason:    &reason,
		BannedBy:  &bannedBy,
		CreatedAt: time.Now(),
	}

	// ON CONFLICT triggers an UPDATE
	mock.ExpectExec("INSERT INTO bans").
		WithArgs(ban.ServerID, ban.UserID, ban.Reason, ban.BannedBy, ban.CreatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Create(ctx, ban)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_Create_DBError(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	reason := "Spam"
	bannedBy := uuid.New()
	ban := &models.Ban{
		ServerID:  uuid.New(),
		UserID:    uuid.New(),
		Reason:    &reason,
		BannedBy:  &bannedBy,
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO bans").
		WillReturnError(sql.ErrConnDone)

	err := repo.Create(ctx, ban)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_GetByServerAndUser(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()
	bannedBy := uuid.New()
	createdAt := time.Now()
	reason := "Violation of ToS"

	rows := sqlmock.NewRows([]string{
		"server_id", "user_id", "reason", "banned_by", "created_at",
	}).AddRow(serverID, userID, reason, bannedBy, createdAt)

	mock.ExpectQuery("SELECT .+ FROM bans WHERE server_id = \\$1 AND user_id = \\$2").
		WithArgs(serverID, userID).
		WillReturnRows(rows)

	ban, err := repo.GetByServerAndUser(ctx, serverID, userID)
	require.NoError(t, err)
	require.NotNil(t, ban)
	assert.Equal(t, serverID, ban.ServerID)
	assert.Equal(t, userID, ban.UserID)
	assert.Equal(t, reason, *ban.Reason)
	assert.Equal(t, bannedBy, *ban.BannedBy)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_GetByServerAndUser_NotFound(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM bans WHERE server_id = \\$1 AND user_id = \\$2").
		WithArgs(serverID, userID).
		WillReturnError(sql.ErrNoRows)

	ban, err := repo.GetByServerAndUser(ctx, serverID, userID)
	require.NoError(t, err)
	assert.Nil(t, ban)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_GetByServerAndUser_DBError(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM bans WHERE server_id = \\$1 AND user_id = \\$2").
		WithArgs(serverID, userID).
		WillReturnError(sql.ErrConnDone)

	ban, err := repo.GetByServerAndUser(ctx, serverID, userID)
	assert.Error(t, err)
	assert.Nil(t, ban)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_GetByServerID(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	reason1 := "Spam"
	reason2 := "Harassment"
	rows := sqlmock.NewRows([]string{
		"server_id", "user_id", "reason", "banned_by", "created_at",
	}).
		AddRow(serverID, uuid.New(), reason1, uuid.New(), time.Now()).
		AddRow(serverID, uuid.New(), reason2, uuid.New(), time.Now())

	mock.ExpectQuery("SELECT .+ FROM bans WHERE server_id = \\$1").
		WithArgs(serverID).
		WillReturnRows(rows)

	bans, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, bans, 2)
	assert.Equal(t, reason1, *bans[0].Reason)
	assert.Equal(t, reason2, *bans[1].Reason)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_GetByServerID_Empty(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	rows := sqlmock.NewRows([]string{
		"server_id", "user_id", "reason", "banned_by", "created_at",
	})

	mock.ExpectQuery("SELECT .+ FROM bans WHERE server_id = \\$1").
		WithArgs(serverID).
		WillReturnRows(rows)

	bans, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, bans, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_GetByServerID_DBError(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	mock.ExpectQuery("SELECT .+ FROM bans WHERE server_id = \\$1").
		WithArgs(serverID).
		WillReturnError(sql.ErrConnDone)

	bans, err := repo.GetByServerID(ctx, serverID)
	assert.Error(t, err)
	assert.Nil(t, bans)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_Delete(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM bans WHERE server_id = \\$1 AND user_id = \\$2").
		WithArgs(serverID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, serverID, userID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_Delete_NotFound(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM bans WHERE server_id = \\$1 AND user_id = \\$2").
		WithArgs(serverID, userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.Delete(ctx, serverID, userID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBanRepo_Delete_DBError(t *testing.T) {
	repo, mock := setupBanRepoMock(t)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	mock.ExpectExec("DELETE FROM bans WHERE server_id = \\$1 AND user_id = \\$2").
		WithArgs(serverID, userID).
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(ctx, serverID, userID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
