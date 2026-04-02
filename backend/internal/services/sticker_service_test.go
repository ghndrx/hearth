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
	stickers       map[uuid.UUID]*models.Sticker
	packs          map[uuid.UUID]*models.StickerPack
	packStickers   map[uuid.UUID]map[uuid.UUID]*models.PackSticker
}

func newMockStickerRepo() *mockStickerRepo {
	return &mockStickerRepo{
		stickers:     make(map[uuid.UUID]*models.Sticker),
		packs:        make(map[uuid.UUID]*models.StickerPack),
		packStickers: make(map[uuid.UUID]map[uuid.UUID]*models.PackSticker),
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

// Sticker Pack methods
func (m *mockStickerRepo) CreatePack(ctx context.Context, pack *models.StickerPack) error {
	m.packs[pack.ID] = pack
	m.packStickers[pack.ID] = make(map[uuid.UUID]*models.PackSticker)
	return nil
}

func (m *mockStickerRepo) GetPackByID(ctx context.Context, id uuid.UUID) (*models.StickerPack, error) {
	pack, ok := m.packs[id]
	if !ok {
		return nil, nil
	}
	return pack, nil
}

func (m *mockStickerRepo) UpdatePack(ctx context.Context, pack *models.StickerPack) error {
	m.packs[pack.ID] = pack
	return nil
}

func (m *mockStickerRepo) DeletePack(ctx context.Context, id uuid.UUID) error {
	delete(m.packs, id)
	delete(m.packStickers, id)
	return nil
}

func (m *mockStickerRepo) GetPacksByServer(ctx context.Context, serverID uuid.UUID) ([]*models.StickerPack, error) {
	var result []*models.StickerPack
	for _, p := range m.packs {
		if p.ServerID != nil && *p.ServerID == serverID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockStickerRepo) GetGlobalPacks(ctx context.Context) ([]*models.StickerPack, error) {
	var result []*models.StickerPack
	for _, p := range m.packs {
		if p.IsGlobal {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockStickerRepo) GetPacksByTier(ctx context.Context, tier models.StickerPackTier) ([]*models.StickerPack, error) {
	var result []*models.StickerPack
	for _, p := range m.packs {
		if p.Tier == tier {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockStickerRepo) GetAvailablePacks(ctx context.Context, serverID *uuid.UUID, userTier models.StickerPackTier) ([]*models.StickerPack, error) {
	var result []*models.StickerPack
	tierOrder := map[models.StickerPackTier]int{
		models.StickerPackTierFree:    0,
		models.StickerPackTierBasic:   1,
		models.StickerPackTierPremium: 2,
	}
	userLevel := tierOrder[userTier]

	for _, p := range m.packs {
		if !p.IsActive {
			continue
		}
		if p.IsGlobal || (serverID != nil && p.ServerID != nil && *p.ServerID == *serverID) {
			if tierOrder[p.Tier] <= userLevel {
				result = append(result, p)
			}
		}
	}
	return result, nil
}

func (m *mockStickerRepo) AddStickerToPack(ctx context.Context, packID, stickerID uuid.UUID, position int, isDefault bool) error {
	if _, ok := m.packStickers[packID]; !ok {
		m.packStickers[packID] = make(map[uuid.UUID]*models.PackSticker)
	}
	m.packStickers[packID][stickerID] = &models.PackSticker{
		ID:        uuid.New(),
		PackID:    packID,
		StickerID: stickerID,
		Position:  position,
		IsDefault: isDefault,
	}
	return nil
}

func (m *mockStickerRepo) RemoveStickerFromPack(ctx context.Context, packID, stickerID uuid.UUID) error {
	if pack, ok := m.packStickers[packID]; ok {
		delete(pack, stickerID)
	}
	return nil
}

func (m *mockStickerRepo) GetStickersInPack(ctx context.Context, packID uuid.UUID) ([]*models.Sticker, error) {
	stickerIDs, ok := m.packStickers[packID]
	if !ok {
		return nil, nil
	}
	var result []*models.Sticker
	for stickerID := range stickerIDs {
		if sticker, ok := m.stickers[stickerID]; ok {
			result = append(result, sticker)
		}
	}
	return result, nil
}

func (m *mockStickerRepo) GetPacksContainingSticker(ctx context.Context, stickerID uuid.UUID) ([]*models.StickerPack, error) {
	var result []*models.StickerPack
	for packID, stickers := range m.packStickers {
		if _, ok := stickers[stickerID]; ok {
			if pack, ok := m.packs[packID]; ok {
				result = append(result, pack)
			}
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
		ID:           uuid.New(),
		Name:         "TestSticker",
		Tags:         []string{"test"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
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
		ID:           uuid.New(),
		ServerID:     &serverID,
		Name:         "TestSticker",
		Tags:         []string{"test"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
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
		ID:           uuid.New(),
		ServerID:     nil,
		Name:         "GlobalSticker",
		Tags:         []string{"global"},
		URL:          "/stickers/global.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
	}
	mockRepo.Create(ctx, globalSticker)

	// Add server-specific sticker
	serverID := uuid.New()
	serverSticker := &models.Sticker{
		ID:           uuid.New(),
		ServerID:     &serverID,
		Name:         "ServerSticker",
		Tags:         []string{"server"},
		URL:          "/stickers/server.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
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
		ID:           uuid.New(),
		ServerID:     nil,
		Name:         "GlobalSticker",
		Tags:         []string{"global"},
		URL:          "/stickers/global.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
	}
	mockRepo.Create(ctx, globalSticker)

	// Add server-specific sticker
	serverSticker := &models.Sticker{
		ID:           uuid.New(),
		ServerID:     &serverID,
		Name:         "ServerSticker",
		Tags:         []string{"server"},
		URL:          "/stickers/server.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
	}
	mockRepo.Create(ctx, serverSticker)

	stickers, err := service.GetAvailable(ctx, &serverID, models.StickerPackTierFree)

	assert.NoError(t, err)
	assert.Len(t, stickers, 2)
}

func TestStickerService_Update(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	sticker := &models.Sticker{
		ID:           uuid.New(),
		Name:         "OldName",
		Tags:         []string{"old"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
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
		ID:           uuid.New(),
		Name:         "OldName",
		Tags:         []string{"old"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
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
		ID:           uuid.New(),
		Name:         "TestSticker",
		Tags:         []string{"test"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
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
		ID:           uuid.New(),
		ServerID:     &serverID,
		Name:         "TestSticker",
		Tags:         []string{"test"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    uuid.New(),
	}
	mockRepo.Create(ctx, sticker)

	results, err := service.Search(ctx, "TestSticker", &serverID)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, sticker.ID, results[0].ID)
}

// Sticker Pack Tests

func TestStickerService_CreatePack(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	desc := "Test pack description"

	pack, err := service.CreatePack(ctx, "Test Pack", &desc, nil, models.StickerPackTierFree, false, nil, userID)

	assert.NoError(t, err)
	assert.NotNil(t, pack)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Equal(t, models.StickerPackTierFree, pack.Tier)
	assert.True(t, pack.IsActive)
}

func TestStickerService_CreatePack_NoName(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	pack, err := service.CreatePack(ctx, "", nil, nil, models.StickerPackTierFree, false, nil, userID)

	assert.ErrorIs(t, err, ErrPackNameRequired)
	assert.Nil(t, pack)
}

func TestStickerService_CreatePack_NameTooLong(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	longName := ""
	for i := 0; i < 105; i++ {
		longName += "a"
	}

	pack, err := service.CreatePack(ctx, longName, nil, nil, models.StickerPackTierFree, false, nil, userID)

	assert.ErrorIs(t, err, ErrPackNameTooLong)
	assert.Nil(t, pack)
}

func TestStickerService_GetPack(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	// Create a pack first
	createdPack, _ := service.CreatePack(ctx, "Test Pack", nil, nil, models.StickerPackTierFree, false, nil, userID)

	// Retrieve it
	retrieved, err := service.GetPack(ctx, createdPack.ID)

	assert.NoError(t, err)
	assert.Equal(t, createdPack.ID, retrieved.ID)
	assert.Equal(t, "Test Pack", retrieved.Name)
}

func TestStickerService_GetPack_NotFound(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()

	retrieved, err := service.GetPack(ctx, uuid.New())

	assert.ErrorIs(t, err, ErrPackNotFound)
	assert.Nil(t, retrieved)
}

func TestStickerService_GetPackWithStickers(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	// Create a sticker
	sticker := &models.Sticker{
		ID:           uuid.New(),
		ServerID:     &serverID,
		Name:         "TestSticker",
		Tags:         []string{"test"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    userID,
	}
	mockRepo.Create(ctx, sticker)

	// Create a pack
	pack, _ := service.CreatePack(ctx, "Test Pack", nil, nil, models.StickerPackTierFree, false, &serverID, userID)

	// Add sticker to pack
	err := service.AddStickerToPack(ctx, pack.ID, sticker.ID, 0, false)
	assert.NoError(t, err)

	// Get pack with stickers
	retrieved, err := service.GetPackWithStickers(ctx, pack.ID, models.StickerPackTierFree)

	assert.NoError(t, err)
	assert.Equal(t, pack.ID, retrieved.ID)
	assert.Len(t, retrieved.Stickers, 1)
	assert.Equal(t, sticker.ID, retrieved.Stickers[0].ID)
}

func TestStickerService_DeletePack(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()

	// Create a pack
	pack, _ := service.CreatePack(ctx, "Test Pack", nil, nil, models.StickerPackTierFree, false, nil, userID)

	// Delete it
	err := service.DeletePack(ctx, pack.ID)
	assert.NoError(t, err)

	// Try to retrieve it
	retrieved, err := service.GetPack(ctx, pack.ID)
	assert.ErrorIs(t, err, ErrPackNotFound)
	assert.Nil(t, retrieved)
}

func TestStickerService_AddStickerToPack(t *testing.T) {
	mockRepo := newMockStickerRepo()
	service := NewStickerService(mockRepo, nil)

	ctx := context.Background()
	userID := uuid.New()
	serverID := uuid.New()

	// Create a sticker
	sticker := &models.Sticker{
		ID:           uuid.New(),
		ServerID:     &serverID,
		Name:         "TestSticker",
		Tags:         []string{"test"},
		URL:          "/stickers/test.png",
		Format:       models.StickerFormatPNG,
		RequiredTier: models.StickerPackTierFree,
		CreatedBy:    userID,
	}
	mockRepo.Create(ctx, sticker)

	// Create a pack
	pack, _ := service.CreatePack(ctx, "Test Pack", nil, nil, models.StickerPackTierFree, false, &serverID, userID)

	// Add sticker to pack
	err := service.AddStickerToPack(ctx, pack.ID, sticker.ID, 0, true)
	assert.NoError(t, err)
}

func TestTierMeetsRequirement(t *testing.T) {
	testCases := []struct {
		name         string
		userTier     models.StickerPackTier
		requiredTier models.StickerPackTier
		expected     bool
	}{
		{"free user can access free", models.StickerPackTierFree, models.StickerPackTierFree, true},
		{"free user cannot access basic", models.StickerPackTierFree, models.StickerPackTierBasic, false},
		{"free user cannot access premium", models.StickerPackTierFree, models.StickerPackTierPremium, false},
		{"basic user can access free", models.StickerPackTierBasic, models.StickerPackTierFree, true},
		{"basic user can access basic", models.StickerPackTierBasic, models.StickerPackTierBasic, true},
		{"basic user cannot access premium", models.StickerPackTierBasic, models.StickerPackTierPremium, false},
		{"premium user can access free", models.StickerPackTierPremium, models.StickerPackTierFree, true},
		{"premium user can access basic", models.StickerPackTierPremium, models.StickerPackTierBasic, true},
		{"premium user can access premium", models.StickerPackTierPremium, models.StickerPackTierPremium, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := models.TierMeetsRequirement(tc.userTier, tc.requiredTier)
			assert.Equal(t, tc.expected, result)
		})
	}
}
