package services

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// mockStickerRepo is a mock implementation of StickerRepository for testing
type mockStickerRepo struct {
	stickers map[uuid.UUID]*models.Sticker
}

func newMockStickerRepo() *mockStickerRepo {
	return &mockStickerRepo{
		stickers: make(map[uuid.UUID]*models.Sticker),
	}
}

func (m *mockStickerRepo) Create(ctx context.Context, sticker *models.Sticker) error {
	m.stickers[sticker.ID] = sticker
	return nil
}

func (m *mockStickerRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Sticker, error) {
	sticker, ok := m.stickers[id]
	if !ok {
		return nil, nil
	}
	return sticker, nil
}

func (m *mockStickerRepo) Update(ctx context.Context, sticker *models.Sticker) error {
	m.stickers[sticker.ID] = sticker
	return nil
}

func (m *mockStickerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.stickers, id)
	return nil
}

func (m *mockStickerRepo) GetByServer(ctx context.Context, serverID uuid.UUID) ([]*models.Sticker, error) {
	var result []*models.Sticker
	for _, s := range m.stickers {
		if s.ServerID != nil && *s.ServerID == serverID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockStickerRepo) GetGlobal(ctx context.Context) ([]*models.Sticker, error) {
	var result []*models.Sticker
	for _, s := range m.stickers {
		if s.ServerID == nil {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockStickerRepo) GetAvailable(ctx context.Context, serverID *uuid.UUID) ([]*models.Sticker, error) {
	var result []*models.Sticker
	for _, s := range m.stickers {
		if s.ServerID == nil {
			result = append(result, s)
		} else if serverID != nil && *s.ServerID == *serverID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockStickerRepo) Search(ctx context.Context, query string, serverID *uuid.UUID) ([]*models.Sticker, error) {
	var result []*models.Sticker
	for _, s := range m.stickers {
		if serverID != nil {
			if s.ServerID == nil || *s.ServerID != *serverID {
				continue
			}
		}
		// Simple name search
		if s.Name == query {
			result = append(result, s)
		}
	}
	return result, nil
}

func TestStickerService_ValidateStickerUpload(t *testing.T) {
	service := &StickerService{}

	testCases := []struct {
		name      string
		size      int64
		mimeType  string
		expectErr error
	}{
		{
			name:      "valid PNG",
			size:      100 * 1024, // 100KB
			mimeType:  "image/png",
			expectErr: nil,
		},
		{
			name:      "valid APNG",
			size:      200 * 1024, // 200KB
			mimeType:  "image/apng",
			expectErr: nil,
		},
		{
			name:      "valid GIF",
			size:      300 * 1024, // 300KB
			mimeType:  "image/gif",
			expectErr: nil,
		},
		{
			name:      "file too large",
			size:      600 * 1024, // 600KB
			mimeType:  "image/png",
			expectErr: ErrStickerTooLarge,
		},
		{
			name:      "invalid format",
			size:      100 * 1024,
			mimeType:  "image/jpeg",
			expectErr: ErrStickerFormat,
		},
		{
			name:      "invalid format - webp",
			size:      100 * 1024,
			mimeType:  "image/webp",
			expectErr: ErrStickerFormat,
		},
		{
			name:      "max size boundary (512KB)",
			size:      512 * 1024,
			mimeType:  "image/png",
			expectErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileHeader := &multipart.FileHeader{
				Filename: "test.png",
				Header:   map[string][]string{"Content-Type": {tc.mimeType}},
				Size:     tc.size,
			}

			err := service.ValidateStickerUpload(fileHeader)
			if tc.expectErr != nil {
				assert.ErrorIs(t, err, tc.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetStickerFormatFromContentType(t *testing.T) {
	testCases := []struct {
		contentType string
		expected    models.StickerFormat
	}{
		{"image/png", models.StickerFormatPNG},
		{"image/apng", models.StickerFormatAPNG},
		{"image/gif", models.StickerFormatGIF},
		{"image/jpeg", models.StickerFormatPNG},
		{"image/webp", models.StickerFormatPNG},
		{"application/octet-stream", models.StickerFormatPNG},
	}

	for _, tc := range testCases {
		t.Run(tc.contentType, func(t *testing.T) {
			result := GetStickerFormatFromContentType(tc.contentType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStickerService_Create(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   map[string][]string{"Content-Type": {"image/png"}},
		Size:     100 * 1024,
	}

	sticker, err := service.Create(ctx, &serverID, "TestSticker", []string{"test"}, fileHeader, userID)

	assert.NoError(t, err)
	assert.NotNil(t, sticker)
	assert.Equal(t, "TestSticker", sticker.Name)
	assert.Equal(t, models.StickerFormatPNG, sticker.Format)
	assert.Equal(t, serverID, *sticker.ServerID)
	assert.Equal(t, userID, sticker.CreatedBy)
}

func TestStickerService_Create_NoName(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   map[string][]string{"Content-Type": {"image/png"}},
		Size:     100 * 1024,
	}

	sticker, err := service.Create(ctx, &serverID, "", []string{"test"}, fileHeader, userID)

	assert.ErrorIs(t, err, ErrStickerNameRequired)
	assert.Nil(t, sticker)
}

func TestStickerService_Create_NameTooLong(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	serverID := uuid.New()
	userID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   map[string][]string{"Content-Type": {"image/png"}},
		Size:     100 * 1024,
	}

	longName := ""
	for i := 0; i < 35; i++ {
		longName += "a"
	}

	sticker, err := service.Create(ctx, &serverID, longName, []string{"test"}, fileHeader, userID)

	assert.ErrorIs(t, err, ErrStickerNameTooLong)
	assert.Nil(t, sticker)
}

func TestStickerService_Get(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	sticker := &models.Sticker{
		ID:     uuid.New(),
		Name:   "TestSticker",
		Tags:   []string{"test"},
		URL:    "/stickers/test.png",
		Format: models.StickerFormatPNG,
	}
	mockRepo.Create(ctx, sticker)

	retrieved, err := service.Get(ctx, sticker.ID)

	assert.NoError(t, err)
	assert.Equal(t, sticker.ID, retrieved.ID)
	assert.Equal(t, sticker.Name, retrieved.Name)
}

func TestStickerService_Get_NotFound(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()

	retrieved, err := service.Get(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrStickerNotFound)
	assert.Nil(t, retrieved)
}

func TestStickerService_GetByServer(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	serverID := uuid.New()

	sticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      "TestSticker",
		Tags:      []string{"test"},
		URL:       "/stickers/test.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, sticker)

	stickers, err := service.GetByServer(ctx, serverID)

	assert.NoError(t, err)
	assert.Len(t, stickers, 1)
	assert.Equal(t, sticker.ID, stickers[0].ID)
}

func TestStickerService_GetGlobal(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()

	// Add global sticker
	globalSticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  nil,
		Name:      "GlobalSticker",
		Tags:      []string{"global"},
		URL:       "/stickers/global.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, globalSticker)

	// Add server-specific sticker
	serverID := uuid.New()
	serverSticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      "ServerSticker",
		Tags:      []string{"server"},
		URL:       "/stickers/server.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, serverSticker)

	stickers, err := service.GetGlobal(ctx)

	assert.NoError(t, err)
	assert.Len(t, stickers, 1)
	assert.Equal(t, globalSticker.ID, stickers[0].ID)
}

func TestStickerService_GetAvailable(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	serverID := uuid.New()

	// Add global sticker
	globalSticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  nil,
		Name:      "GlobalSticker",
		Tags:      []string{"global"},
		URL:       "/stickers/global.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, globalSticker)

	// Add server-specific sticker
	serverSticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      "ServerSticker",
		Tags:      []string{"server"},
		URL:       "/stickers/server.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, serverSticker)

	stickers, err := service.GetAvailable(ctx, &serverID)

	assert.NoError(t, err)
	assert.Len(t, stickers, 2)
}

func TestStickerService_Update(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	sticker := &models.Sticker{
		ID:        uuid.New(),
		Name:      "OldName",
		Tags:      []string{"old"},
		URL:       "/stickers/test.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, sticker)

	updated, err := service.Update(ctx, sticker.ID, "NewName", []string{"new"})

	assert.NoError(t, err)
	assert.Equal(t, "NewName", updated.Name)
	assert.Equal(t, []string{"new"}, updated.Tags)
}

func TestStickerService_Update_NotFound(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()

	updated, err := service.Update(ctx, uuid.New(), "NewName", []string{"new"})

	assert.ErrorIs(t, err, ErrStickerNotFound)
	assert.Nil(t, updated)
}

func TestStickerService_Update_NameTooLong(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	sticker := &models.Sticker{
		ID:        uuid.New(),
		Name:      "OldName",
		Tags:      []string{"old"},
		URL:       "/stickers/test.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, sticker)

	longName := ""
	for i := 0; i < 35; i++ {
		longName += "a"
	}

	updated, err := service.Update(ctx, sticker.ID, longName, nil)

	assert.ErrorIs(t, err, ErrStickerNameTooLong)
	assert.Nil(t, updated)
}

func TestStickerService_Delete(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	sticker := &models.Sticker{
		ID:        uuid.New(),
		Name:      "TestSticker",
		Tags:      []string{"test"},
		URL:       "/stickers/test.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, sticker)

	err := service.Delete(ctx, sticker.ID)

	assert.NoError(t, err)

	_, err = service.Get(ctx, sticker.ID)
	assert.ErrorIs(t, err, ErrStickerNotFound)
}

func TestStickerService_Delete_NotFound(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()

	err := service.Delete(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrStickerNotFound)
}

func TestStickerService_Search(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	serverID := uuid.New()

	sticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      "TestSticker",
		Tags:      []string{"test"},
		URL:       "/stickers/test.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: uuid.New(),
	}
	mockRepo.Create(ctx, sticker)

	results, err := service.Search(ctx, "TestSticker", &serverID)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, sticker.ID, results[0].ID)
}
