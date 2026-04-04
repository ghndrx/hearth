package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// Mock repository for testing
type mockContentSafetyRepo struct {
	filters   map[uuid.UUID]*models.ContentFilter
	ageVerify map[uuid.UUID]*models.AgeVerificationSetting
	prefs     map[uuid.UUID]*models.UserContentPreference
}

func newMockContentSafetyRepo() *mockContentSafetyRepo {
	return &mockContentSafetyRepo{
		filters:   make(map[uuid.UUID]*models.ContentFilter),
		ageVerify: make(map[uuid.UUID]*models.AgeVerificationSetting),
		prefs:     make(map[uuid.UUID]*models.UserContentPreference),
	}
}

func (m *mockContentSafetyRepo) CreateContentFilter(ctx context.Context, filter *models.ContentFilter) error {
	filter.ID = uuid.New()
	m.filters[filter.ID] = filter
	return nil
}

func (m *mockContentSafetyRepo) GetContentFilterByID(ctx context.Context, id uuid.UUID) (*models.ContentFilter, error) {
	return m.filters[id], nil
}

func (m *mockContentSafetyRepo) GetContentFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error) {
	var result []*models.ContentFilter
	for _, f := range m.filters {
		if f.ServerID == serverID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockContentSafetyRepo) GetContentFiltersByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.ContentFilter, error) {
	var result []*models.ContentFilter
	for _, f := range m.filters {
		if f.ChannelID != nil && *f.ChannelID == channelID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockContentSafetyRepo) GetContentFiltersForMessage(ctx context.Context, serverID, channelID uuid.UUID) ([]*models.ContentFilter, error) {
	var result []*models.ContentFilter
	for _, f := range m.filters {
		if f.ServerID == serverID && f.Enabled {
			if f.ChannelID == nil || *f.ChannelID == channelID {
				result = append(result, f)
			}
		}
	}
	return result, nil
}

func (m *mockContentSafetyRepo) UpdateContentFilter(ctx context.Context, filter *models.ContentFilter) error {
	m.filters[filter.ID] = filter
	return nil
}

func (m *mockContentSafetyRepo) DeleteContentFilter(ctx context.Context, id uuid.UUID) error {
	delete(m.filters, id)
	return nil
}

func (m *mockContentSafetyRepo) GetEnabledFiltersByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.ContentFilter, error) {
	var result []*models.ContentFilter
	for _, f := range m.filters {
		if f.ServerID == serverID && f.Enabled {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *mockContentSafetyRepo) CreateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error {
	settings.ID = uuid.New()
	m.ageVerify[settings.ID] = settings
	return nil
}

func (m *mockContentSafetyRepo) GetAgeVerificationByServerID(ctx context.Context, serverID uuid.UUID) (*models.AgeVerificationSetting, error) {
	for _, s := range m.ageVerify {
		if s.ServerID == serverID && s.ChannelID == nil {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockContentSafetyRepo) GetAgeVerificationForChannel(ctx context.Context, serverID, channelID uuid.UUID) (*models.AgeVerificationSetting, error) {
	for _, s := range m.ageVerify {
		if s.ServerID == serverID && s.ChannelID != nil && *s.ChannelID == channelID {
			return s, nil
		}
	}
	// Fall back to server-wide
	return m.GetAgeVerificationByServerID(ctx, serverID)
}

func (m *mockContentSafetyRepo) UpdateAgeVerification(ctx context.Context, settings *models.AgeVerificationSetting) error {
	m.ageVerify[settings.ID] = settings
	return nil
}

func (m *mockContentSafetyRepo) DeleteAgeVerification(ctx context.Context, id uuid.UUID) error {
	delete(m.ageVerify, id)
	return nil
}

func (m *mockContentSafetyRepo) CreateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error {
	prefs.ID = uuid.New()
	m.prefs[prefs.ID] = prefs
	return nil
}

func (m *mockContentSafetyRepo) GetUserContentPreference(ctx context.Context, userID uuid.UUID) (*models.UserContentPreference, error) {
	for _, p := range m.prefs {
		if p.UserID == userID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockContentSafetyRepo) UpdateUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error {
	m.prefs[prefs.ID] = prefs
	return nil
}

func (m *mockContentSafetyRepo) UpsertUserContentPreference(ctx context.Context, prefs *models.UserContentPreference) error {
	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}
	m.prefs[prefs.ID] = prefs
	return nil
}

func TestContentSafetyService_CreateContentFilter(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	req := &models.CreateContentFilterRequest{
		Name:   "Test Filter",
		Type:   models.FilterTypeNSFW,
		Action: models.FilterActionBlock,
	}

	filter, err := svc.CreateContentFilter(ctx, serverID, userID, req)
	assert.NoError(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, "Test Filter", filter.Name)
	assert.Equal(t, serverID, filter.ServerID)
	assert.True(t, filter.Enabled)
}

func TestContentSafetyService_GetContentFilter(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	filter := &models.ContentFilter{
		ID:        uuid.New(),
		ServerID:  uuid.New(),
		Name:      "Test Filter",
		Type:      models.FilterTypeNSFW,
		Enabled:   true,
		Threshold: models.NSFWThresholdMedium,
		Action:    models.FilterActionBlock,
	}
	repo.filters[filter.ID] = filter

	found, err := svc.GetContentFilter(ctx, filter.ID)
	assert.NoError(t, err)
	assert.Equal(t, filter.ID, found.ID)
	assert.Equal(t, "Test Filter", found.Name)
}

func TestContentSafetyService_GetServerContentFilters(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()

	// Create filters for server
	for i := 0; i < 3; i++ {
		filter := &models.ContentFilter{
			ID:        uuid.New(),
			ServerID:  serverID,
			Name:      "Filter " + string(rune('A'+i)),
			Type:      models.FilterTypeNSFW,
			Enabled:   true,
			Threshold: models.NSFWThresholdMedium,
			Action:    models.FilterActionBlock,
		}
		repo.filters[filter.ID] = filter
	}

	// Create filter for different server
	otherServerFilter := &models.ContentFilter{
		ID:        uuid.New(),
		ServerID:  uuid.New(),
		Name:      "Other Server Filter",
		Type:      models.FilterTypeNSFW,
		Enabled:   true,
		Threshold: models.NSFWThresholdMedium,
		Action:    models.FilterActionBlock,
	}
	repo.filters[otherServerFilter.ID] = otherServerFilter

	filters, err := svc.GetServerContentFilters(ctx, serverID)
	assert.NoError(t, err)
	assert.Len(t, filters, 3)
}

func TestContentSafetyService_UpdateContentFilter(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	filter := &models.ContentFilter{
		ID:        uuid.New(),
		ServerID:  uuid.New(),
		Name:      "Original Name",
		Type:      models.FilterTypeNSFW,
		Enabled:   true,
		Threshold: models.NSFWThresholdLow,
		Action:    models.FilterActionWarn,
	}
	repo.filters[filter.ID] = filter

	newName := "Updated Name"
	newThreshold := models.NSFWThresholdHigh
	req := &models.UpdateContentFilterRequest{
		Name:      &newName,
		Threshold: &newThreshold,
	}

	updated, err := svc.UpdateContentFilter(ctx, filter.ID, req)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, models.NSFWThresholdHigh, updated.Threshold)
	assert.Equal(t, models.FilterActionWarn, updated.Action) // Unchanged
}

func TestContentSafetyService_DeleteContentFilter(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	filter := &models.ContentFilter{
		ID:       uuid.New(),
		ServerID: uuid.New(),
		Name:     "Test Filter",
		Type:     models.FilterTypeNSFW,
		Enabled:  true,
		Action:   models.FilterActionBlock,
	}
	repo.filters[filter.ID] = filter

	err := svc.DeleteContentFilter(ctx, filter.ID)
	assert.NoError(t, err)

	// Verify deleted
	found, err := svc.GetContentFilter(ctx, filter.ID)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestContentSafetyService_CreateAgeVerification(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()
	userID := uuid.New()

	req := &models.CreateAgeVerificationRequest{
		Enabled:          true,
		RequiredAge:      18,
		VerificationType: "automatic",
	}

	settings, err := svc.CreateAgeVerification(ctx, serverID, userID, req)
	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, serverID, settings.ServerID)
	assert.True(t, settings.Enabled)
	assert.Equal(t, 18, settings.RequiredAge)
	assert.Equal(t, "automatic", settings.VerificationType)
}

func TestContentSafetyService_GetAgeVerification(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()

	settings := &models.AgeVerificationSetting{
		ID:               uuid.New(),
		ServerID:         serverID,
		ChannelID:        nil,
		Enabled:          true,
		RequiredAge:      18,
		VerificationType: "automatic",
	}
	repo.ageVerify[settings.ID] = settings

	found, err := svc.GetAgeVerification(ctx, serverID)
	assert.NoError(t, err)
	assert.Equal(t, settings.ID, found.ID)
	assert.Equal(t, 18, found.RequiredAge)
}

func TestContentSafetyService_GetUserContentPreference(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	userID := uuid.New()

	// Initially should return default preferences
	prefs, err := svc.GetUserContentPreference(ctx, userID)
	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Equal(t, userID, prefs.UserID)
	assert.Equal(t, models.NSFWThresholdMedium, prefs.NSFWFilterLevel)
	assert.True(t, prefs.HideNSFWContent)

	// Create actual preferences
	actualPrefs := &models.UserContentPreference{
		ID:                 uuid.New(),
		UserID:             userID,
		NSFWFilterLevel:    models.NSFWThresholdHigh,
		HideNSFWContent:    false,
		HideExplicitContent: true,
		AutoCollapseNSFW:   true,
	}
	repo.prefs[actualPrefs.ID] = actualPrefs

	found, err := svc.GetUserContentPreference(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, actualPrefs.ID, found.ID)
	assert.Equal(t, models.NSFWThresholdHigh, found.NSFWFilterLevel)
	assert.False(t, found.HideNSFWContent)
	assert.True(t, found.AutoCollapseNSFW)
}

func TestContentSafetyService_UpdateUserContentPreference(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	userID := uuid.New()

	req := &models.UpdateUserContentPreferenceRequest{
		NSFWFilterLevel: func() *models.NSFWDetectionThreshold { v := models.NSFWThresholdHigh; return &v }(),
		HideNSFWContent: func() *bool { v := false; return &v }(),
	}

	prefs, err := svc.UpdateUserContentPreference(ctx, userID, req)
	assert.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Equal(t, models.NSFWThresholdHigh, prefs.NSFWFilterLevel)
	assert.False(t, prefs.HideNSFWContent)
}

func TestContentSafetyService_ScanContent_NoFilters(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	result, err := svc.ScanContent(ctx, serverID, channelID, userID, nil, "Hello world")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Empty(t, result.Flags)
}

func TestContentSafetyService_ScanContent_WithKeywordFilter(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	// Create a custom keyword filter
	filter := &models.ContentFilter{
		ID:        uuid.New(),
		ServerID:  serverID,
		ChannelID: nil,
		Name:      "Bad Words Filter",
		Type:      models.FilterTypeCustomKeyword,
		Enabled:   true,
		Threshold: models.NSFWThresholdLow,
		Action:    models.FilterActionBlock,
		FilterData: models.ContentFilterData{
			Keywords: []string{"badword", "offensive"},
		},
	}
	repo.filters[filter.ID] = filter

	// Test with matching content
	result, err := svc.ScanContent(ctx, serverID, channelID, userID, nil, "This contains badword in it")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
	assert.Len(t, result.Flags, 1)
	assert.Equal(t, models.FilterTypeCustomKeyword, result.Flags[0].Type)

	// Test with non-matching content
	result, err = svc.ScanContent(ctx, serverID, channelID, userID, nil, "This is clean content")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
}

func TestContentSafetyService_ScanContent_WithWhitelist(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	// Create a custom keyword filter with whitelist
	filter := &models.ContentFilter{
		ID:        uuid.New(),
		ServerID:  serverID,
		ChannelID: nil,
		Name:      "Bad Words Filter",
		Type:      models.FilterTypeCustomKeyword,
		Enabled:   true,
		Threshold: models.NSFWThresholdLow,
		Action:    models.FilterActionBlock,
		FilterData: models.ContentFilterData{
			Keywords:  []string{"badword"},
			Whitelist: []string{"notreallybadword"},
		},
	}
	repo.filters[filter.ID] = filter

	// Test with whitelisted content
	result, err := svc.ScanContent(ctx, serverID, channelID, userID, nil, "This contains notreallybadword in it")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed) // Should pass due to whitelist
}

func TestContentSafetyService_ScanContent_UserExempt(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	// Create a filter with exempt role
	filter := &models.ContentFilter{
		ID:           uuid.New(),
		ServerID:     serverID,
		ChannelID:    nil,
		Name:         "Bad Words Filter",
		Type:         models.FilterTypeCustomKeyword,
		Enabled:      true,
		Threshold:    models.NSFWThresholdLow,
		Action:       models.FilterActionBlock,
		ExemptRoles:  []uuid.UUID{roleID},
		FilterData: models.ContentFilterData{
			Keywords: []string{"badword"},
		},
	}
	repo.filters[filter.ID] = filter

	// Test with user having exempt role
	result, err := svc.ScanContent(ctx, serverID, channelID, userID, []uuid.UUID{roleID}, "This contains badword")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed) // Should pass due to exempt role

	// Test with user without exempt role
	result, err = svc.ScanContent(ctx, serverID, channelID, userID, []uuid.UUID{uuid.New()}, "This contains badword")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Passed) // Should fail - no exempt role
}

func TestContentSafetyService_ShouldBlockContent(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	// Create a block action filter
	filter := &models.ContentFilter{
		ID:        uuid.New(),
		ServerID:  serverID,
		ChannelID: nil,
		Name:      "Block Filter",
		Type:      models.FilterTypeCustomKeyword,
		Enabled:   true,
		Threshold: models.NSFWThresholdLow,
		Action:    models.FilterActionBlock,
		FilterData: models.ContentFilterData{
			Keywords: []string{"badword"},
		},
	}
	repo.filters[filter.ID] = filter

	// Test blocking
	shouldBlock, result, err := svc.ShouldBlockContent(ctx, serverID, channelID, userID, nil, "This contains badword")
	assert.NoError(t, err)
	assert.True(t, shouldBlock)
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
	assert.Equal(t, models.FilterActionBlock, result.ActionTaken)

	// Test non-blocking content
	shouldBlock, result, err = svc.ShouldBlockContent(ctx, serverID, channelID, userID, nil, "This is clean")
	assert.NoError(t, err)
	assert.False(t, shouldBlock)
	assert.True(t, result.Passed)
}

func TestContentSafetyService_GetServerSafetySettings(t *testing.T) {
	repo := newMockContentSafetyRepo()
	svc := NewContentSafetyService(repo)
	ctx := context.Background()

	serverID := uuid.New()

	// Create filters
	for i := 0; i < 2; i++ {
		filter := &models.ContentFilter{
			ID:        uuid.New(),
			ServerID:  serverID,
			Name:      "Filter " + string(rune('A'+i)),
			Type:      models.FilterTypeNSFW,
			Enabled:   true,
			Threshold: models.NSFWThresholdMedium,
			Action:    models.FilterActionBlock,
		}
		repo.filters[filter.ID] = filter
	}

	// Create age verification
	ageVerify := &models.AgeVerificationSetting{
		ID:               uuid.New(),
		ServerID:         serverID,
		Enabled:          true,
		RequiredAge:      18,
		VerificationType: "automatic",
	}
	repo.ageVerify[ageVerify.ID] = ageVerify

	settings, err := svc.GetServerSafetySettings(ctx, serverID)
	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Equal(t, serverID, settings.ServerID)
	assert.Len(t, settings.Filters, 2)
	assert.NotNil(t, settings.AgeVerification)
	assert.Equal(t, 18, settings.AgeVerification.RequiredAge)
	assert.Equal(t, models.NSFWThresholdMedium, settings.ServerDefaultThreshold)
}
