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

func setupRoleRepoMock(t *testing.T) (*RoleRepository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRoleRepository(sqlxDB)
	return repo, mock
}

// --- RoleRepository Tests ---

func TestRoleRepository_Create(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	role := &models.Role{
		ID:          uuid.New(),
		ServerID:    uuid.New(),
		Name:        "TestRole",
		Color:       0xFF5500,
		Hoist:       true,
		Position:    1,
		Permissions: models.PermViewChannels | models.PermSendMessages,
		Mentionable: true,
		IsDefault:   false,
		CreatedAt:   time.Now(),
	}

	mock.ExpectExec("INSERT INTO roles").
		WithArgs(
			role.ID, role.ServerID, role.Name, role.Color, role.Hoist, role.Position,
			role.Permissions, role.Mentionable, role.IsDefault, role.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(ctx, role)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Create_DBError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	role := &models.Role{
		ID:        uuid.New(),
		ServerID:  uuid.New(),
		Name:      "ErrorRole",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO roles").
		WillReturnError(sql.ErrConnDone)

	err := repo.Create(ctx, role)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetByID(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	roleID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "server_id", "name", "color", "hoist", "position", "permissions", "mentionable", "is_default", "created_at"}).
		AddRow(roleID, uuid.New(), "TestRole", 0xFF0000, false, 1, models.PermViewChannels, true, false, time.Now())

	mock.ExpectQuery("SELECT \\* FROM roles WHERE id").
		WithArgs(roleID).
		WillReturnRows(rows)

	role, err := repo.GetByID(ctx, roleID)
	require.NoError(t, err)
	require.NotNil(t, role)
	assert.Equal(t, roleID, role.ID)
	assert.Equal(t, "TestRole", role.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetByID_NotFound(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	roleID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM roles WHERE id").
		WithArgs(roleID).
		WillReturnError(sql.ErrNoRows)

	role, err := repo.GetByID(ctx, roleID)
	require.NoError(t, err)
	assert.Nil(t, role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetByID_DBError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	roleID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM roles WHERE id").
		WithArgs(roleID).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.GetByID(ctx, roleID)
	assert.Error(t, err)
	// Note: when GetContext returns error (not ErrNoRows), role may still be non-nil with partial data
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetByServerID(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "server_id", "name", "color", "hoist", "position", "permissions", "mentionable", "is_default", "created_at"}).
		AddRow(uuid.New(), serverID, "Role1", 0xFF0000, false, 1, models.PermViewChannels, true, false, time.Now()).
		AddRow(uuid.New(), serverID, "Role2", 0x00FF00, true, 2, models.PermSendMessages, false, false, time.Now())

	mock.ExpectQuery("SELECT \\* FROM roles WHERE server_id").
		WithArgs(serverID).
		WillReturnRows(rows)

	roles, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetByServerID_Empty(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "server_id", "name", "color", "hoist", "position", "permissions", "mentionable", "is_default", "created_at"})

	mock.ExpectQuery("SELECT \\* FROM roles WHERE server_id").
		WithArgs(serverID).
		WillReturnRows(rows)

	roles, err := repo.GetByServerID(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, roles, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetByIDs(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	ids := []uuid.UUID{uuid.New(), uuid.New()}

	rows := sqlmock.NewRows([]string{"id", "server_id", "name", "color", "hoist", "position", "permissions", "mentionable", "is_default", "created_at"}).
		AddRow(ids[0], uuid.New(), "Role1", 0xFF0000, false, 1, models.PermViewChannels, true, false, time.Now()).
		AddRow(ids[1], uuid.New(), "Role2", 0x00FF00, true, 2, models.PermSendMessages, false, false, time.Now())

	mock.ExpectQuery("SELECT \\* FROM roles WHERE id IN").
		WithArgs(ids[0], ids[1]).
		WillReturnRows(rows)

	roles, err := repo.GetByIDs(ctx, ids)
	require.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetByIDs_EmptySlice(t *testing.T) {
	repo, _ := setupRoleRepoMock(t)
	ctx := context.Background()

	roles, err := repo.GetByIDs(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Len(t, roles, 0)
}

func TestRoleRepository_Update(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	roleID := uuid.New()

	role := &models.Role{
		ID:          roleID,
		Name:        "UpdatedRole",
		Color:       0x0000FF,
		Hoist:       true,
		Position:    3,
		Permissions: models.PermAdministrator,
		Mentionable: false,
	}

	// Query order: $1=id, $2=name, $3=color, $4=hoist, $5=position, $6=permissions, $7=mentionable
	mock.ExpectExec("UPDATE roles SET").
		WithArgs(role.ID, role.Name, role.Color, role.Hoist, role.Position, role.Permissions, role.Mentionable).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Update(ctx, role)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Update_DBError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	role := &models.Role{
		ID:   uuid.New(),
		Name: "ErrorRole",
	}

	mock.ExpectExec("UPDATE roles SET").
		WillReturnError(sql.ErrConnDone)

	err := repo.Update(ctx, role)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Delete(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	roleID := uuid.New()

	mock.ExpectExec("DELETE FROM roles WHERE id").
		WithArgs(roleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, roleID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_Delete_DBError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	roleID := uuid.New()

	mock.ExpectExec("DELETE FROM roles WHERE id").
		WillReturnError(sql.ErrConnDone)

	err := repo.Delete(ctx, roleID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_AddRoleToMember(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	mock.ExpectExec("UPDATE members SET roles = array_append").
		WithArgs(serverID, userID, roleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.AddRoleToMember(ctx, serverID, userID, roleID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_AddRoleToMember_DBError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE members SET roles = array_append").
		WillReturnError(sql.ErrConnDone)

	err := repo.AddRoleToMember(ctx, uuid.New(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_RemoveRoleFromMember(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	mock.ExpectExec("UPDATE members SET roles = array_remove").
		WithArgs(serverID, userID, roleID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveRoleFromMember(ctx, serverID, userID, roleID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_RemoveRoleFromMember_DBError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	mock.ExpectExec("UPDATE members SET roles = array_remove").
		WillReturnError(sql.ErrConnDone)

	err := repo.RemoveRoleFromMember(ctx, uuid.New(), uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_UpdatePositions(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	roleID1 := uuid.New()
	roleID2 := uuid.New()
	positions := map[uuid.UUID]int{
		roleID1: 1,
		roleID2: 2,
	}

	mock.ExpectBegin()
	// Accept updates in any order since map iteration is non-deterministic
	mock.ExpectExec("UPDATE roles SET position").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE roles SET position").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.UpdatePositions(ctx, serverID, positions)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_UpdatePositions_BeginError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

	err := repo.UpdatePositions(ctx, uuid.New(), map[uuid.UUID]int{uuid.New(): 1})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetMemberRoles(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "server_id", "name", "color", "hoist", "position", "permissions", "mentionable", "is_default", "created_at"}).
		AddRow(uuid.New(), serverID, "Role1", 0xFF0000, false, 1, models.PermViewChannels, true, false, time.Now()).
		AddRow(uuid.New(), serverID, "Role2", 0x00FF00, true, 2, models.PermSendMessages, false, false, time.Now())

	mock.ExpectQuery("SELECT r\\.\\* FROM roles r").
		WithArgs(serverID, userID).
		WillReturnRows(rows)

	roles, err := repo.GetMemberRoles(ctx, serverID, userID)
	require.NoError(t, err)
	assert.Len(t, roles, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetMemberRoles_Empty(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "server_id", "name", "color", "hoist", "position", "permissions", "mentionable", "is_default", "created_at"})

	mock.ExpectQuery("SELECT r\\.\\* FROM roles r").
		WillReturnRows(rows)

	roles, err := repo.GetMemberRoles(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Len(t, roles, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetMemberPermissions(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	expectedPerms := models.PermViewChannels | models.PermSendMessages

	rows := sqlmock.NewRows([]string{"permissions"}).
		AddRow(expectedPerms)

	mock.ExpectQuery("SELECT COALESCE\\(bit_or").
		WithArgs(serverID, userID).
		WillReturnRows(rows)

	perms, err := repo.GetMemberPermissions(ctx, serverID, userID)
	require.NoError(t, err)
	assert.Equal(t, expectedPerms, perms)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetMemberPermissions_DBError(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()

	mock.ExpectQuery("SELECT COALESCE\\(bit_or").
		WillReturnError(sql.ErrConnDone)

	_, err := repo.GetMemberPermissions(ctx, uuid.New(), uuid.New())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetDefaultRole(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "server_id", "name", "color", "hoist", "position", "permissions", "mentionable", "is_default", "created_at"}).
		AddRow(uuid.New(), serverID, "@everyone", 0x000000, false, 0, models.DefaultPermissions, true, true, time.Now())

	mock.ExpectQuery("SELECT \\* FROM roles WHERE server_id").
		WithArgs(serverID).
		WillReturnRows(rows)

	role, err := repo.GetDefaultRole(ctx, serverID)
	require.NoError(t, err)
	require.NotNil(t, role)
	assert.Equal(t, "@everyone", role.Name)
	assert.True(t, role.IsDefault)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRoleRepository_GetDefaultRole_NotFound(t *testing.T) {
	repo, mock := setupRoleRepoMock(t)
	ctx := context.Background()
	serverID := uuid.New()

	mock.ExpectQuery("SELECT \\* FROM roles WHERE server_id").
		WithArgs(serverID).
		WillReturnError(sql.ErrNoRows)

	role, err := repo.GetDefaultRole(ctx, serverID)
	require.NoError(t, err)
	assert.Nil(t, role)
	assert.NoError(t, mock.ExpectationsWereMet())
}
