package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hearth/internal/models"
)

func TestAuditLogService_Log(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	targetID := uuid.New()

	changes := []models.Change{
		{Key: "name", OldValue: "old", NewValue: "new"},
	}

	err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, &targetID, changes, "spam")
	require.NoError(t, err)

	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, models.AuditLogMemberBan, logs[0].ActionType)
	assert.Equal(t, userID, logs[0].UserID)
	assert.Equal(t, &targetID, logs[0].TargetID)
	assert.Equal(t, "spam", logs[0].Reason)
	assert.Len(t, logs[0].Changes, 1)
}

func TestAuditLogService_GetLogs_FilterByActionType(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create entries with different action types
	err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID, models.AuditLogChannelCreate, nil, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)

	// Filter by action type
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		ActionType: models.AuditLogMemberBan,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, 2, total)
	for _, log := range logs {
		assert.Equal(t, models.AuditLogMemberBan, log.ActionType)
	}
}

func TestAuditLogService_GetLogs_FilterByUserID(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	// Create entries from different users
	err := svc.Log(ctx, serverID, userID1, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID2, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID1, models.AuditLogChannelCreate, nil, nil, "")
	require.NoError(t, err)

	// Filter by user ID
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		UserID: &userID1,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, 2, total)
	for _, log := range logs {
		assert.Equal(t, userID1, log.UserID)
	}
}

func TestAuditLogService_GetLogs_FilterByTargetID(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()
	targetID1 := uuid.New()
	targetID2 := uuid.New()

	// Create entries with different targets
	err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, &targetID1, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, &targetID2, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, &targetID1, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID, models.AuditLogChannelCreate, nil, nil, "") // no target
	require.NoError(t, err)

	// Filter by target ID
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		TargetID: &targetID1,
		Limit:    10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, 2, total)
	for _, log := range logs {
		assert.Equal(t, &targetID1, log.TargetID)
	}
}

func TestAuditLogService_GetLogs_FilterByDateRange(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create entries
	err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	afterTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	err = svc.Log(ctx, serverID, userID, models.AuditLogChannelCreate, nil, nil, "")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	err = svc.Log(ctx, serverID, userID, models.AuditLogRoleCreate, nil, nil, "")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	beforeTime := time.Now()
	time.Sleep(10 * time.Millisecond)

	err = svc.Log(ctx, serverID, userID, models.AuditLogMemberKick, nil, nil, "")
	require.NoError(t, err)

	// Filter by after time (should get entries after first one)
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		After: &afterTime,
		Limit: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total) // CHANNEL_CREATE, ROLE_CREATE, MEMBER_KICK

	// Filter by before time (should exclude last one)
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		Before: &beforeTime,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total) // MEMBER_BAN, CHANNEL_CREATE, ROLE_CREATE

	// Filter by date range
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		After:  &afterTime,
		Before: &beforeTime,
		Limit:  10,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total) // CHANNEL_CREATE, ROLE_CREATE
	assert.Len(t, logs, 2)
}

func TestAuditLogService_GetLogs_Pagination(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create 15 entries
	for i := 0; i < 15; i++ {
		err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, nil, nil, "")
		require.NoError(t, err)
	}

	// Get first page
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit:  5,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 5)
	assert.Equal(t, 15, total)

	// Get second page
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit:  5,
		Offset: 5,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 5)
	assert.Equal(t, 15, total)

	// Get last page
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit:  5,
		Offset: 10,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 5)
	assert.Equal(t, 15, total)

	// Get beyond entries
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit:  5,
		Offset: 15,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 0)
	assert.Equal(t, 15, total)
}

func TestAuditLogService_GetLogs_DefaultLimit(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create 60 entries
	for i := 0; i < 60; i++ {
		err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, nil, nil, "")
		require.NoError(t, err)
	}

	// Get with default limit
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{})
	require.NoError(t, err)
	assert.Len(t, logs, 50) // Default limit is 50
	assert.Equal(t, 60, total)
}

func TestAuditLogService_GetLogs_MaxLimit(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create 150 entries
	for i := 0; i < 150; i++ {
		err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, nil, nil, "")
		require.NoError(t, err)
	}

	// Get with limit > 100
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		Limit: 200,
	})
	require.NoError(t, err)
	assert.Len(t, logs, 100) // Max limit is 100
	assert.Equal(t, 150, total)
}

func TestAuditLogService_GetLogs_CombinedFilters(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	targetID := uuid.New()

	// Create various entries
	err := svc.Log(ctx, serverID, userID1, models.AuditLogMemberBan, &targetID, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID1, models.AuditLogChannelCreate, nil, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID2, models.AuditLogMemberBan, &targetID, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID, userID1, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)

	// Filter by user AND action type
	logs, total, err := svc.GetLogs(ctx, serverID, AuditLogFilter{
		UserID:     &userID1,
		ActionType: models.AuditLogMemberBan,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, logs, 2)
	for _, log := range logs {
		assert.Equal(t, userID1, log.UserID)
		assert.Equal(t, models.AuditLogMemberBan, log.ActionType)
	}

	// Filter by user AND action type AND target
	logs, total, err = svc.GetLogs(ctx, serverID, AuditLogFilter{
		UserID:     &userID1,
		ActionType: models.AuditLogMemberBan,
		TargetID:   &targetID,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, logs, 1)
	assert.Equal(t, userID1, logs[0].UserID)
	assert.Equal(t, models.AuditLogMemberBan, logs[0].ActionType)
	assert.Equal(t, &targetID, logs[0].TargetID)
}

func TestAuditLogService_GetLogs_ServerIsolation(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID1 := uuid.New()
	serverID2 := uuid.New()
	userID := uuid.New()

	// Create entries in different servers
	err := svc.Log(ctx, serverID1, userID, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID2, userID, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)
	err = svc.Log(ctx, serverID1, userID, models.AuditLogChannelCreate, nil, nil, "")
	require.NoError(t, err)

	// Get logs for server 1
	logs, total, err := svc.GetLogs(ctx, serverID1, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, 2, total)
	for _, log := range logs {
		assert.Equal(t, serverID1, log.ServerID)
	}

	// Get logs for server 2
	logs, total, err = svc.GetLogs(ctx, serverID2, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, serverID2, logs[0].ServerID)
}

func TestAuditLogService_GetLogByID(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create an entry
	err := svc.Log(ctx, serverID, userID, models.AuditLogMemberBan, nil, nil, "spam")
	require.NoError(t, err)

	// Get all logs to find the entry ID
	logs, _, err := svc.GetLogs(ctx, serverID, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, logs, 1)

	// Get by ID
	entry, err := svc.GetLogByID(ctx, serverID, logs[0].ID)
	require.NoError(t, err)
	assert.Equal(t, logs[0].ID, entry.ID)
	assert.Equal(t, models.AuditLogMemberBan, entry.ActionType)
	assert.Equal(t, "spam", entry.Reason)
}

func TestAuditLogService_GetLogByID_NotFound(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	randomID := uuid.New()

	// Try to get non-existent entry
	entry, err := svc.GetLogByID(ctx, serverID, randomID)
	assert.ErrorIs(t, err, ErrAuditLogNotFound)
	assert.Nil(t, entry)
}

func TestAuditLogService_GetLogByID_WrongServer(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID1 := uuid.New()
	serverID2 := uuid.New()
	userID := uuid.New()

	// Create an entry in server 1
	err := svc.Log(ctx, serverID1, userID, models.AuditLogMemberBan, nil, nil, "")
	require.NoError(t, err)

	// Get the entry ID
	logs, _, err := svc.GetLogs(ctx, serverID1, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, logs, 1)

	// Try to get entry from server 2 (should fail)
	entry, err := svc.GetLogByID(ctx, serverID2, logs[0].ID)
	assert.ErrorIs(t, err, ErrAuditLogNotFound)
	assert.Nil(t, entry)
}

func TestAuditLogService_GetActionTypes(t *testing.T) {
	svc := NewAuditLogService()
	types := svc.GetActionTypes()

	// Verify all expected action types are present
	expectedTypes := []string{
		models.AuditLogServerUpdate,
		models.AuditLogChannelCreate,
		models.AuditLogChannelUpdate,
		models.AuditLogChannelDelete,
		models.AuditLogMemberKick,
		models.AuditLogMemberBan,
		models.AuditLogMemberUnban,
		models.AuditLogMemberUpdate,
		models.AuditLogRoleCreate,
		models.AuditLogRoleUpdate,
		models.AuditLogRoleDelete,
		models.AuditLogInviteCreate,
		models.AuditLogInviteDelete,
		models.AuditLogWebhookCreate,
		models.AuditLogWebhookUpdate,
		models.AuditLogWebhookDelete,
		models.AuditLogEmojiCreate,
		models.AuditLogEmojiUpdate,
		models.AuditLogEmojiDelete,
		models.AuditLogMessageDelete,
		models.AuditLogMessageBulkDelete,
		models.AuditLogMessagePin,
		models.AuditLogMessageUnpin,
	}

	assert.ElementsMatch(t, expectedTypes, types)
}

func TestAuditLogService_GetLogs_ReturnsInReverseOrder(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	// Create entries
	actions := []string{
		models.AuditLogMemberBan,
		models.AuditLogChannelCreate,
		models.AuditLogRoleCreate,
	}
	for _, action := range actions {
		err := svc.Log(ctx, serverID, userID, action, nil, nil, "")
		require.NoError(t, err)
	}

	// Get logs - should be in reverse order (newest first)
	logs, _, err := svc.GetLogs(ctx, serverID, AuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 3)
	assert.Equal(t, models.AuditLogRoleCreate, logs[0].ActionType)
	assert.Equal(t, models.AuditLogChannelCreate, logs[1].ActionType)
	assert.Equal(t, models.AuditLogMemberBan, logs[2].ActionType)
}
