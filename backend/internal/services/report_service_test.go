package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReportService_Create(t *testing.T) {
	svc := NewReportService()
	ctx := context.Background()

	r, err := svc.Create(ctx, uuid.New(), uuid.New(), "message", "Spam")
	assert.NoError(t, err)
	assert.Equal(t, "pending", r.Status)
}

func TestReportService_UpdateStatus(t *testing.T) {
	svc := NewReportService()
	ctx := context.Background()

	r, _ := svc.Create(ctx, uuid.New(), uuid.New(), "user", "Harassment")
	svc.UpdateStatus(ctx, r.ID, "resolved")

	pending, _ := svc.GetPending(ctx)
	assert.Len(t, pending, 0)
}
