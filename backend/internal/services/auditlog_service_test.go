package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogService_Log(t *testing.T) {
	svc := NewAuditLogService()
	ctx := context.Background()
	serverID := uuid.New()

	err := svc.Log(ctx, serverID, uuid.New(), "MEMBER_BAN", nil, nil)
	assert.NoError(t, err)

	logs, _ := svc.GetLogs(ctx, serverID, 10)
	assert.Len(t, logs, 1)
	assert.Equal(t, "MEMBER_BAN", logs[0].Action)
}
