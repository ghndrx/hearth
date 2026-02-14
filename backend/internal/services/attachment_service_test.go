package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAttachmentService_Upload(t *testing.T) {
	svc := NewAttachmentService()
	ctx := context.Background()

	a, err := svc.Upload(ctx, uuid.New(), "test.png", "image/png", 1024)
	assert.NoError(t, err)
	assert.Equal(t, "test.png", a.Filename)
}

func TestAttachmentService_GetAndDelete(t *testing.T) {
	svc := NewAttachmentService()
	ctx := context.Background()

	a, _ := svc.Upload(ctx, uuid.New(), "test.pdf", "application/pdf", 2048)

	found, err := svc.Get(ctx, a.ID)
	assert.NoError(t, err)
	assert.Equal(t, a.ID, found.ID)

	svc.Delete(ctx, a.ID)
	_, err = svc.Get(ctx, a.ID)
	assert.Error(t, err)
}
