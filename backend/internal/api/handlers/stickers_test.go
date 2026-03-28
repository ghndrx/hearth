package handlers

import (
	"context"
	"io"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hearth/internal/models"
	"hearth/internal/services"
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
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(query)) {
			result = append(result, s)
			continue
		}
		// Tag search
		for _, tag := range s.Tags {
			if strings.Contains(strings.ToLower(tag), strings.ToLower(query)) {
				result = append(result, s)
				break
			}
		}
	}
	return result, nil
}

func TestStickerService_Create(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	// Create a mock file header
	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	sticker, err := service.Create(ctx, &serverID, "TestSticker", []string{"tag1", "tag2"}, fileHeader, userID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, sticker.ID)
	assert.Equal(t, "TestSticker", sticker.Name)
	assert.Equal(t, []string{"tag1", "tag2"}, sticker.Tags)
	assert.Equal(t, models.StickerFormatPNG, sticker.Format)
	assert.Equal(t, userID, sticker.CreatedBy)
	assert.Equal(t, &serverID, sticker.ServerID)
}

func TestStickerService_Create_NoName(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	_, err := service.Create(ctx, &serverID, "", nil, fileHeader, userID)
	assert.ErrorIs(t, err, services.ErrStickerNameRequired)
}

func TestStickerService_Create_NameTooLong(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	_, err := service.Create(ctx, &serverID, strings.Repeat("a", 31), nil, fileHeader, userID)
	assert.ErrorIs(t, err, services.ErrStickerNameTooLong)
}

func TestStickerService_Create_InvalidFormat(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.jpg",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/jpeg")

	_, err := service.Create(ctx, &serverID, "TestSticker", nil, fileHeader, userID)
	assert.ErrorIs(t, err, services.ErrStickerFormat)
}

func TestStickerService_Create_FileTooLarge(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     600 * 1024, // 600KB - over 512KB limit
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	_, err := service.Create(ctx, &serverID, "TestSticker", nil, fileHeader, userID)
	assert.ErrorIs(t, err, services.ErrStickerTooLarge)
}

func TestStickerService_Get(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	created, err := service.Create(ctx, &serverID, "TestSticker", nil, fileHeader, userID)
	require.NoError(t, err)

	sticker, err := service.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, sticker.ID)
	assert.Equal(t, "TestSticker", sticker.Name)
}

func TestStickerService_Get_NotFound(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()

	_, err := service.Get(ctx, uuid.New())
	assert.ErrorIs(t, err, services.ErrStickerNotFound)
}

func TestStickerService_GetByServer(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()
	otherServerID := uuid.New()

	// Create stickers for server
	for i := 0; i < 3; i++ {
		fileHeader := &multipart.FileHeader{
			Filename: "test.png",
			Header:   make(map[string][]string),
			Size:     1024,
		}
		fileHeader.Header.Set("Content-Type", "image/png")
		_, err := service.Create(ctx, &serverID, "ServerSticker", nil, fileHeader, userID)
		require.NoError(t, err)
	}

	// Create sticker for other server
	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")
	_, err := service.Create(ctx, &otherServerID, "OtherServerSticker", nil, fileHeader, userID)
	require.NoError(t, err)

	stickers, err := service.GetByServer(ctx, serverID)
	require.NoError(t, err)
	assert.Len(t, stickers, 3)
}

func TestStickerService_GetGlobal(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	// Create global sticker
	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")
	_, err := service.Create(ctx, nil, "GlobalSticker", nil, fileHeader, userID)
	require.NoError(t, err)

	// Create server sticker
	fileHeader2 := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader2.Header.Set("Content-Type", "image/png")
	_, err = service.Create(ctx, &serverID, "ServerSticker", nil, fileHeader2, userID)
	require.NoError(t, err)

	stickers, err := service.GetGlobal(ctx)
	require.NoError(t, err)
	assert.Len(t, stickers, 1)
	assert.Equal(t, "GlobalSticker", stickers[0].Name)
}

func TestStickerService_GetAvailable(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	// Create global sticker
	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")
	_, err := service.Create(ctx, nil, "GlobalSticker", nil, fileHeader, userID)
	require.NoError(t, err)

	// Create server sticker
	fileHeader2 := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader2.Header.Set("Content-Type", "image/png")
	_, err = service.Create(ctx, &serverID, "ServerSticker", nil, fileHeader2, userID)
	require.NoError(t, err)

	// Without server context - only global
	stickers, err := service.GetAvailable(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, stickers, 1)
	assert.Equal(t, "GlobalSticker", stickers[0].Name)

	// With server context - global + server
	stickers, err = service.GetAvailable(ctx, &serverID)
	require.NoError(t, err)
	assert.Len(t, stickers, 2)
}

func TestStickerService_Update(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	created, err := service.Create(ctx, &serverID, "OriginalName", nil, fileHeader, userID)
	require.NoError(t, err)

	// Update name
	updated, err := service.Update(ctx, created.ID, "NewName", nil)
	require.NoError(t, err)
	assert.Equal(t, "NewName", updated.Name)

	// Update tags
	updated, err = service.Update(ctx, created.ID, "", []string{"new", "tags"})
	require.NoError(t, err)
	assert.Equal(t, []string{"new", "tags"}, updated.Tags)

	// Update both
	updated, err = service.Update(ctx, created.ID, "AnotherName", []string{"other", "tags"})
	require.NoError(t, err)
	assert.Equal(t, "AnotherName", updated.Name)
	assert.Equal(t, []string{"other", "tags"}, updated.Tags)
}

func TestStickerService_Update_NameTooLong(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	created, err := service.Create(ctx, &serverID, "TestSticker", nil, fileHeader, userID)
	require.NoError(t, err)

	_, err = service.Update(ctx, created.ID, strings.Repeat("a", 31), nil)
	assert.ErrorIs(t, err, services.ErrStickerNameTooLong)
}

func TestStickerService_Update_NotFound(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()

	_, err := service.Update(ctx, uuid.New(), "NewName", nil)
	assert.ErrorIs(t, err, services.ErrStickerNotFound)
}

func TestStickerService_Delete(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	fileHeader := &multipart.FileHeader{
		Filename: "test.png",
		Header:   make(map[string][]string),
		Size:     1024,
	}
	fileHeader.Header.Set("Content-Type", "image/png")

	created, err := service.Create(ctx, &serverID, "TestSticker", nil, fileHeader, userID)
	require.NoError(t, err)

	err = service.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = service.Get(ctx, created.ID)
	assert.ErrorIs(t, err, services.ErrStickerNotFound)
}

func TestStickerService_Delete_NotFound(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()

	err := service.Delete(ctx, uuid.New())
	assert.ErrorIs(t, err, services.ErrStickerNotFound)
}

func TestStickerService_Search(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := services.NewStickerService(mockRepo, nil)
	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	// Create stickers
	testCases := []struct {
		name string
		tags []string
	}{
		{"Happy Cat", []string{"cat", "happy"}},
		{"Sad Dog", []string{"dog", "sad"}},
		{"Happy Dog", []string{"dog", "happy"}},
		{"Cat", []string{"cat"}},
	}

	for _, tc := range testCases {
		fileHeader := &multipart.FileHeader{
			Filename: "test.png",
			Header:   make(map[string][]string),
			Size:     1024,
		}
		fileHeader.Header.Set("Content-Type", "image/png")
		_, err := service.Create(ctx, &serverID, tc.name, tc.tags, fileHeader, userID)
		require.NoError(t, err)
	}

	// Search by name
	results, err := service.Search(ctx, "cat", &serverID)
	require.NoError(t, err)
	assert.Len(t, results, 2) // "Happy Cat" and "Cat"

	// Search by tag
	results, err = service.Search(ctx, "happy", &serverID)
	require.NoError(t, err)
	assert.Len(t, results, 2) // "Happy Cat" and "Happy Dog"

	// Search by tag (sad)
	results, err = service.Search(ctx, "sad", &serverID)
	require.NoError(t, err)
	assert.Len(t, results, 1) // "Sad Dog"

	// Search without server filter
	results, err = service.Search(ctx, "cat", nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestStickerService_GetStickerFormatFromContentType(t *testing.T) {
	testCases := []struct {
		contentType string
		expected    models.StickerFormat
	}{
		{"image/png", models.StickerFormatPNG},
		{"image/apng", models.StickerFormatAPNG},
		{"image/gif", models.StickerFormatGIF},
		{"image/jpeg", models.StickerFormatPNG}, // Default fallback
	}

	for _, tc := range testCases {
		result := services.GetStickerFormatFromContentType(tc.contentType)
		assert.Equal(t, tc.expected, result)
	}
}

// MockMultipartFileHeader creates a mock multipart.FileHeader for testing
func createMockFileHeader(filename, contentType string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: filename,
		Header:   map[string][]string{"Content-Type": {contentType}},
		Size:     size,
	}
}

// MockFileReader wraps a string as a file reader for testing
type MockFileReader struct {
	content string
}

func (m *MockFileReader) Read(p []byte) (n int, err error) {
	if len(m.content) == 0 {
		return 0, io.EOF
	}
	n = copy(p, m.content)
	m.content = m.content[n:]
	return n, nil
}

func (m *MockFileReader) Close() error {
	return nil
}

// TestStickerResponse tests the ToResponse conversion
func TestStickerResponse(t *testing.T) {
	now := time.Now()
	serverID := uuid.New()
	userID := uuid.New()

	sticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  &serverID,
		Name:      "TestSticker",
		Tags:      []string{"tag1", "tag2"},
		URL:       "/stickers/test.png",
		Format:    models.StickerFormatPNG,
		CreatedBy: userID,
		CreatedAt: now,
	}

	resp := sticker.ToResponse()

	assert.Equal(t, sticker.ID.String(), resp.ID)
	assert.Equal(t, sticker.Name, resp.Name)
	assert.Equal(t, sticker.Tags, resp.Tags)
	assert.Equal(t, sticker.URL, resp.URL)
	assert.Equal(t, string(sticker.Format), resp.Format)
	assert.Equal(t, sticker.CreatedBy.String(), resp.CreatedBy)
	assert.NotNil(t, resp.ServerID)
	assert.Equal(t, serverID.String(), *resp.ServerID)
}

// TestStickerResponse_Global tests ToResponse for global stickers
func TestStickerResponse_Global(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	sticker := &models.Sticker{
		ID:        uuid.New(),
		ServerID:  nil, // Global sticker
		Name:      "GlobalSticker",
		Tags:      []string{"global"},
		URL:       "/stickers/global.png",
		Format:    models.StickerFormatGIF,
		CreatedBy: userID,
		CreatedAt: now,
	}

	resp := sticker.ToResponse()

	assert.Equal(t, sticker.ID.String(), resp.ID)
	assert.Nil(t, resp.ServerID) // Global sticker has nil server ID
	assert.Equal(t, "GlobalSticker", resp.Name)
}
