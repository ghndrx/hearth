package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockSearchRepository is a mock implementation of SearchRepository
type MockSearchRepository struct {
	mock.Mock
}

func (m *MockSearchRepository) SearchMessages(ctx context.Context, opts SearchMessageOptions) (*SearchResult, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SearchResult), args.Error(1)
}

func (m *MockSearchRepository) SearchUsers(ctx context.Context, query string, serverID *uuid.UUID, limit int) ([]*models.PublicUser, error) {
	args := m.Called(ctx, query, serverID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PublicUser), args.Error(1)
}

func (m *MockSearchRepository) SearchChannels(ctx context.Context, query string, serverID *uuid.UUID, limit int) ([]*models.Channel, error) {
	args := m.Called(ctx, query, serverID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

// MockUserRepositoryForSearch is a mock implementation of UserRepository
type MockUserRepositoryForSearch struct {
	mock.Mock
}

func (m *MockUserRepositoryForSearch) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForSearch) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForSearch) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepositoryForSearch) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) GetFriends(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepositoryForSearch) AddFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) RemoveFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	args := m.Called(ctx, userID, friendID)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) GetBlockedUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepositoryForSearch) BlockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) UnblockUser(ctx context.Context, userID, blockedID uuid.UUID) error {
	args := m.Called(ctx, userID, blockedID)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) UpdatePresence(ctx context.Context, userID uuid.UUID, status models.PresenceStatus) error {
	args := m.Called(ctx, userID, status)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Presence), args.Error(1)
}

func (m *MockUserRepositoryForSearch) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	args := m.Called(ctx, userIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*models.Presence), args.Error(1)
}

func (m *MockUserRepositoryForSearch) GetRelationship(ctx context.Context, userID, targetID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID, targetID)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepositoryForSearch) SendFriendRequest(ctx context.Context, senderID, receiverID uuid.UUID) error {
	args := m.Called(ctx, senderID, receiverID)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) GetIncomingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepositoryForSearch) GetOutgoingFriendRequests(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepositoryForSearch) AcceptFriendRequest(ctx context.Context, receiverID, senderID uuid.UUID) error {
	args := m.Called(ctx, receiverID, senderID)
	return args.Error(0)
}

func (m *MockUserRepositoryForSearch) DeclineFriendRequest(ctx context.Context, userID, otherID uuid.UUID) error {
	args := m.Called(ctx, userID, otherID)
	return args.Error(0)
}

func setupSearchService() (*SearchService, *MockSearchRepository, *MockMessageRepository, *MockChannelRepositoryForMessages, *MockServerRepository, *MockUserRepositoryForSearch, *MockCacheService) {
	searchRepo := new(MockSearchRepository)
	msgRepo := new(MockMessageRepository)
	channelRepo := new(MockChannelRepositoryForMessages)
	serverRepo := new(MockServerRepository)
	userRepo := new(MockUserRepositoryForSearch)
	cache := new(MockCacheService)

	service := NewSearchService(
		searchRepo,
		msgRepo,
		channelRepo,
		serverRepo,
		userRepo,
		cache,
	)

	return service, searchRepo, msgRepo, channelRepo, serverRepo, userRepo, cache
}

func TestSearchMessages_Basic(t *testing.T) {
	service, searchRepo, _, _, _, userRepo, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	authorID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeText,
		ServerID:   &serverID,
		Recipients: []uuid.UUID{},
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedMessages := []*models.Message{
		{
			ID:        uuid.New(),
			ChannelID: channelID,
			AuthorID:  authorID,
			Content:   "Hello world",
		},
	}

	expectedResult := &SearchResult{
		Messages: expectedMessages,
		Total:    1,
		HasMore:  false,
	}

	searchRepo.On("SearchMessages", ctx, mock.AnythingOfType("SearchMessageOptions")).Return(expectedResult, nil)
	channelRepo := new(MockChannelRepositoryForMessages)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	service.channelRepo = channelRepo

	serverRepo := new(MockServerRepository)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	service.serverRepo = serverRepo

	user := &models.User{
		ID:       authorID,
		Username: "testuser",
	}
	userRepo.On("GetByID", ctx, authorID).Return(user, nil)

	opts := SearchMessageOptions{
		Query:       "hello",
		ChannelID:   &channelID,
		RequesterID: requesterID,
		Limit:       25,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Messages, 1)
	assert.Equal(t, "Hello world", result.Messages[0].Content)
}

func TestSearchMessages_WithServer(t *testing.T) {
	service, searchRepo, _, channelRepo, serverRepo, userRepo, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	serverID := uuid.New()
	channelID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	expectedMessages := []*models.Message{
		{
			ID:        uuid.New(),
			ChannelID: channelID,
			ServerID:  &serverID,
			AuthorID:  requesterID,
			Content:   "Test message",
		},
	}

	expectedResult := &SearchResult{
		Messages: expectedMessages,
		Total:    1,
		HasMore:  false,
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	channelRepo.On("GetByServerID", ctx, serverID).Return([]*models.Channel{channel}, nil)
	searchRepo.On("SearchMessages", ctx, mock.AnythingOfType("SearchMessageOptions")).Return(expectedResult, nil)

	user := &models.User{
		ID:       requesterID,
		Username: "testuser",
	}
	userRepo.On("GetByID", ctx, requesterID).Return(user, nil)

	opts := SearchMessageOptions{
		Query:       "test",
		ServerID:    &serverID,
		RequesterID: requesterID,
		Limit:       25,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Messages, 1)
}

func TestSearchMessages_NotServerMember(t *testing.T) {
	service, _, _, _, serverRepo, _, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	serverID := uuid.New()

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	opts := SearchMessageOptions{
		Query:       "test",
		ServerID:    &serverID,
		RequesterID: requesterID,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, result)
}

func TestSearchMessages_WithFilters(t *testing.T) {
	service, searchRepo, _, channelRepo, _, userRepo, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	authorID := uuid.New()
	serverID := uuid.New()
	before := time.Now()
	after := before.Add(-24 * time.Hour)
	hasAttachments := true
	pinned := true

	channel := &models.Channel{
		ID:       channelID,
		Type:     models.ChannelTypeText,
		ServerID: &serverID,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedMessages := []*models.Message{
		{
			ID:        uuid.New(),
			ChannelID: channelID,
			AuthorID:  authorID,
			Content:   "Filtered message",
			Pinned:    true,
		},
	}

	expectedResult := &SearchResult{
		Messages: expectedMessages,
		Total:    1,
		HasMore:  false,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	searchRepo.On("SearchMessages", ctx, mock.AnythingOfType("SearchMessageOptions")).Return(expectedResult, nil)

	serverRepo := new(MockServerRepository)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	service.serverRepo = serverRepo

	user := &models.User{
		ID:       authorID,
		Username: "testuser",
	}
	userRepo.On("GetByID", ctx, authorID).Return(user, nil)

	opts := SearchMessageOptions{
		Query:          "filtered",
		ChannelID:      &channelID,
		AuthorID:       &authorID,
		Before:         &before,
		After:          &after,
		HasAttachments: &hasAttachments,
		Pinned:         &pinned,
		RequesterID:    requesterID,
		Limit:          25,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Messages, 1)
}

func TestSearchMessages_DefaultLimit(t *testing.T) {
	service, searchRepo, _, channelRepo, _, userRepo, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		Type:     models.ChannelTypeText,
		ServerID: &serverID,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedMessages := []*models.Message{
		{ID: uuid.New(), ChannelID: channelID, Content: "Test"},
	}

	expectedResult := &SearchResult{
		Messages: expectedMessages,
		Total:    1,
		HasMore:  false,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	searchRepo.On("SearchMessages", ctx, mock.Anything).Return(expectedResult, nil)

	serverRepo := new(MockServerRepository)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	service.serverRepo = serverRepo

	user := &models.User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	userRepo.On("GetByID", ctx, mock.Anything).Return(user, nil)

	opts := SearchMessageOptions{
		Query:       "test",
		ChannelID:   &channelID,
		RequesterID: requesterID,
		Limit:       0, // Should default to 25
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSearchMessages_Pagination(t *testing.T) {
	service, searchRepo, _, channelRepo, _, userRepo, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		Type:     models.ChannelTypeText,
		ServerID: &serverID,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	// Create 26 messages to test pagination
	expectedMessages := make([]*models.Message, 25)
	for i := 0; i < 25; i++ {
		expectedMessages[i] = &models.Message{
			ID:        uuid.New(),
			ChannelID: channelID,
			Content:   "Message",
		}
	}

	expectedResult := &SearchResult{
		Messages: expectedMessages,
		Total:    100,
		HasMore:  true,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	searchRepo.On("SearchMessages", ctx, mock.Anything).Return(expectedResult, nil)

	serverRepo := new(MockServerRepository)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	service.serverRepo = serverRepo

	user := &models.User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	userRepo.On("GetByID", ctx, mock.Anything).Return(user, nil)

	opts := SearchMessageOptions{
		Query:       "message",
		ChannelID:   &channelID,
		RequesterID: requesterID,
		Limit:       25,
		Offset:      0,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasMore)
	assert.Equal(t, 100, result.Total)
}

func TestSearchUsers_Success(t *testing.T) {
	service, searchRepo, _, _, serverRepo, _, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	serverID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedUsers := []*models.PublicUser{
		{
			ID:       uuid.New(),
			Username: "testuser",
		},
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	searchRepo.On("SearchUsers", ctx, "test", &serverID, 25).Return(expectedUsers, nil)

	users, err := service.SearchUsers(ctx, "test", &serverID, requesterID, 25)

	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "testuser", users[0].Username)
}

func TestSearchUsers_DefaultLimit(t *testing.T) {
	service, searchRepo, _, _, _, _, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()

	expectedUsers := []*models.PublicUser{
		{ID: uuid.New(), Username: "user1"},
	}

	searchRepo.On("SearchUsers", ctx, "test", (*uuid.UUID)(nil), 25).Return(expectedUsers, nil)

	users, err := service.SearchUsers(ctx, "test", nil, requesterID, 0)

	assert.NoError(t, err)
	assert.Len(t, users, 1)
}

func TestSearchChannels_Success(t *testing.T) {
	service, searchRepo, _, _, serverRepo, _, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	serverID := uuid.New()

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedChannels := []*models.Channel{
		{
			ID:   uuid.New(),
			Name: "general",
		},
	}

	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	searchRepo.On("SearchChannels", ctx, "general", &serverID, 25).Return(expectedChannels, nil)

	channels, err := service.SearchChannels(ctx, "general", &serverID, requesterID, 25)

	assert.NoError(t, err)
	assert.Len(t, channels, 1)
	assert.Equal(t, "general", channels[0].Name)
}

func TestParseSearchQuery(t *testing.T) {
	// Test that ParseSearchQuery returns a SearchMessageOptions with the query
	opts := ParseSearchQuery("hello world")
	assert.Equal(t, "hello world", opts.Query)

	// Test empty query
	opts = ParseSearchQuery("")
	assert.Equal(t, "", opts.Query)
}

func TestSearchMessages_WithMentions(t *testing.T) {
	service, searchRepo, _, channelRepo, _, userRepo, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	mentionedUserID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		Type:     models.ChannelTypeText,
		ServerID: &serverID,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedMessages := []*models.Message{
		{
			ID:        uuid.New(),
			ChannelID: channelID,
			Content:   "@user hello",
		},
	}

	expectedResult := &SearchResult{
		Messages: expectedMessages,
		Total:    1,
		HasMore:  false,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	searchRepo.On("SearchMessages", ctx, mock.AnythingOfType("SearchMessageOptions")).Return(expectedResult, nil)

	serverRepo := new(MockServerRepository)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	service.serverRepo = serverRepo

	user := &models.User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	userRepo.On("GetByID", ctx, mock.Anything).Return(user, nil)

	opts := SearchMessageOptions{
		Query:       "hello",
		ChannelID:   &channelID,
		Mentions:    []uuid.UUID{mentionedUserID},
		RequesterID: requesterID,
		Limit:       25,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSearchMessages_DMChannel(t *testing.T) {
	service, searchRepo, _, channelRepo, _, userRepo, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	otherUserID := uuid.New()
	channelID := uuid.New()

	channel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{requesterID, otherUserID},
	}

	expectedMessages := []*models.Message{
		{
			ID:        uuid.New(),
			ChannelID: channelID,
			Content:   "DM message",
		},
	}

	expectedResult := &SearchResult{
		Messages: expectedMessages,
		Total:    1,
		HasMore:  false,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	searchRepo.On("SearchMessages", ctx, mock.AnythingOfType("SearchMessageOptions")).Return(expectedResult, nil)

	user := &models.User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	userRepo.On("GetByID", ctx, mock.Anything).Return(user, nil)

	opts := SearchMessageOptions{
		Query:       "DM",
		ChannelID:   &channelID,
		RequesterID: requesterID,
		Limit:       25,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestSearchMessages_DMChannelNotParticipant(t *testing.T) {
	service, _, _, channelRepo, _, _, _ := setupSearchService()
	ctx := context.Background()
	requesterID := uuid.New()
	otherUserID := uuid.New()
	differentUserID := uuid.New()
	channelID := uuid.New()

	channel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{otherUserID, differentUserID}, // requesterID is NOT in the list
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	opts := SearchMessageOptions{
		Query:       "test",
		ChannelID:   &channelID,
		RequesterID: requesterID,
	}

	result, err := service.SearchMessages(ctx, opts)

	assert.Error(t, err)
	assert.Equal(t, ErrNoPermission, err)
	assert.Nil(t, result)
}
