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

// MockScreenShareRepository is a mock implementation of ScreenShareRepository
type MockScreenShareRepository struct {
	mock.Mock
}

func (m *MockScreenShareRepository) CreateSession(ctx context.Context, session *models.StreamSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockScreenShareRepository) GetSession(ctx context.Context, id uuid.UUID) (*models.StreamSession, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StreamSession), args.Error(1)
}

func (m *MockScreenShareRepository) GetActiveSessionByChannel(ctx context.Context, channelID uuid.UUID) (*models.StreamSession, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StreamSession), args.Error(1)
}

func (m *MockScreenShareRepository) UpdateSession(ctx context.Context, session *models.StreamSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockScreenShareRepository) EndSession(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockScreenShareRepository) ListActiveSessions(ctx context.Context) ([]*models.StreamSession, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.StreamSession), args.Error(1)
}

func (m *MockScreenShareRepository) ListActiveSessionsByServer(ctx context.Context, serverID uuid.UUID) ([]*models.StreamSession, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.StreamSession), args.Error(1)
}

func (m *MockScreenShareRepository) AddViewer(ctx context.Context, sessionID, userID uuid.UUID) error {
	args := m.Called(ctx, sessionID, userID)
	return args.Error(0)
}

func (m *MockScreenShareRepository) RemoveViewer(ctx context.Context, sessionID, userID uuid.UUID) error {
	args := m.Called(ctx, sessionID, userID)
	return args.Error(0)
}

func (m *MockScreenShareRepository) GetViewerCount(ctx context.Context, sessionID uuid.UUID) (int, error) {
	args := m.Called(ctx, sessionID)
	return args.Int(0), args.Error(1)
}

func (m *MockScreenShareRepository) GetViewers(ctx context.Context, sessionID uuid.UUID) ([]models.StreamViewer, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.StreamViewer), args.Error(1)
}

func (m *MockScreenShareRepository) IsViewing(ctx context.Context, sessionID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, sessionID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockScreenShareRepository) GetActiveStreamForUser(ctx context.Context, userID uuid.UUID) (*models.StreamSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StreamSession), args.Error(1)
}

// MockChannelRepositoryForScreenShare is a mock implementation of ChannelRepository
type MockChannelRepositoryForScreenShare struct {
	mock.Mock
}

func (m *MockChannelRepositoryForScreenShare) Create(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) GetByID(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForScreenShare) Update(ctx context.Context, channel *models.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForScreenShare) GetDMChannel(ctx context.Context, user1ID, user2ID uuid.UUID) (*models.Channel, error) {
	args := m.Called(ctx, user1ID, user2ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForScreenShare) GetUserDMs(ctx context.Context, userID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockChannelRepositoryForScreenShare) UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, at time.Time) error {
	args := m.Called(ctx, channelID, messageID, at)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) GetPermissionOverrides(ctx context.Context, channelID uuid.UUID) ([]models.PermissionOverride, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.PermissionOverride), args.Error(1)
}

func (m *MockChannelRepositoryForScreenShare) UpsertPermissionOverride(ctx context.Context, override *models.PermissionOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) DeletePermissionOverride(ctx context.Context, channelID, targetID uuid.UUID, targetType string) error {
	args := m.Called(ctx, channelID, targetID, targetType)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) AddRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) RemoveRecipient(ctx context.Context, channelID, userID uuid.UUID) error {
	args := m.Called(ctx, channelID, userID)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) CountRecipients(ctx context.Context, channelID uuid.UUID) (int, error) {
	args := m.Called(ctx, channelID)
	return args.Int(0), args.Error(1)
}

func (m *MockChannelRepositoryForScreenShare) BulkUpdatePositions(ctx context.Context, entries []models.ReorderChannelEntry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *MockChannelRepositoryForScreenShare) UpdateForumConfig(ctx context.Context, channelID uuid.UUID, configJSON []byte) error {
	args := m.Called(ctx, channelID, configJSON)
	return args.Error(0)
}

// MockServerRepositoryForScreenShare is a mock implementation of ServerRepository
type MockServerRepositoryForScreenShare struct {
	mock.Mock
}

func (m *MockServerRepositoryForScreenShare) Create(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) GetByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) Update(ctx context.Context, server *models.Server) error {
	args := m.Called(ctx, server)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) GetMember(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) AddMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) RemoveMember(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) GetMembers(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) UpdateMember(ctx context.Context, member *models.Member) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) GetInvites(ctx context.Context, serverID uuid.UUID) ([]*models.Invite, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) CreateInvite(ctx context.Context, invite *models.Invite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) GetChannels(ctx context.Context, serverID uuid.UUID) ([]*models.Channel, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Channel), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetBans(ctx context.Context, serverID uuid.UUID) ([]*models.Ban, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) CreateBan(ctx context.Context, serverID, userID, creatorID uuid.UUID, reason string) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID, creatorID, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) RemoveBan(ctx context.Context, serverID, userID uuid.UUID) error {
	args := m.Called(ctx, serverID, userID)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) GetRoles(ctx context.Context, serverID uuid.UUID) ([]*models.Role, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Role), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetDefaultRole(ctx context.Context, serverID uuid.UUID) (*models.Role, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetMemberCount(ctx context.Context, serverID uuid.UUID) (int, error) {
	args := m.Called(ctx, serverID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetUserServers(ctx context.Context, userID uuid.UUID) ([]*models.Server, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Server), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetAllMembers(ctx context.Context, serverID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetMemberWithRoles(ctx context.Context, serverID, userID uuid.UUID) (*models.Member, []*models.Role, int64, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, nil, 0, args.Error(1)
	}
	return args.Get(0).(*models.Member), args.Get(1).([]*models.Role), args.Get(2).(int64), args.Error(3)
}

func (m *MockServerRepositoryForScreenShare) AddBan(ctx context.Context, ban *models.Ban) error {
	args := m.Called(ctx, ban)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) TransferOwnership(ctx context.Context, serverID, newOwnerID uuid.UUID) error {
	args := m.Called(ctx, serverID, newOwnerID)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) GetMembersPaginated(ctx context.Context, serverID uuid.UUID, cursor *models.MemberCursor, limit int) (*models.PaginatedMembers, error) {
	args := m.Called(ctx, serverID, cursor, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaginatedMembers), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetMembersWithRole(ctx context.Context, serverID, roleID uuid.UUID) ([]*models.Member, error) {
	args := m.Called(ctx, serverID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Member), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetOwnedServersCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetBan(ctx context.Context, serverID, userID uuid.UUID) (*models.Ban, error) {
	args := m.Called(ctx, serverID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Ban), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) GetInvite(ctx context.Context, code string) (*models.Invite, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invite), args.Error(1)
}

func (m *MockServerRepositoryForScreenShare) DeleteInvite(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}

func (m *MockServerRepositoryForScreenShare) IncrementInviteUses(ctx context.Context, code string) error {
	args := m.Called(ctx, code)
	return args.Error(0)
}
func (m *MockServerRepositoryForScreenShare) GetInviteByVanityCode(ctx context.Context, vanityCode string) (*models.Invite, error) {
	return nil, nil
}
func (m *MockServerRepositoryForScreenShare) LogInviteUse(ctx context.Context, log *models.InviteUseLog) error {
	return nil
}
func (m *MockServerRepositoryForScreenShare) GetInviteUseLogs(ctx context.Context, inviteCode string) ([]models.InviteUseLog, error) {
	return nil, nil
}
func (m *MockServerRepositoryForScreenShare) GetServerInviteUseLogs(ctx context.Context, serverID uuid.UUID) ([]models.InviteUseLog, error) {
	return nil, nil
}

// MockEventBusForScreenShare is a mock implementation of EventBus
type MockEventBusForScreenShare struct {
	mock.Mock
}

func (m *MockEventBusForScreenShare) Publish(event string, data interface{}) {
	m.Called(event, data)
}

func (m *MockEventBusForScreenShare) Subscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

func (m *MockEventBusForScreenShare) Unsubscribe(event string, handler func(data interface{})) {
	m.Called(event, handler)
}

func setupScreenShareService() (*ScreenShareService, *MockScreenShareRepository, *MockChannelRepositoryForScreenShare, *MockServerRepositoryForScreenShare, *MockEventBusForScreenShare) {
	repo := new(MockScreenShareRepository)
	channelRepo := new(MockChannelRepositoryForScreenShare)
	serverRepo := new(MockServerRepositoryForScreenShare)
	eventBus := new(MockEventBusForScreenShare)
	service := NewScreenShareService(repo, channelRepo, serverRepo, nil, eventBus)
	return service, repo, channelRepo, serverRepo, eventBus
}

func TestScreenShareService_StartStream(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeVoice,
		Name:     "General",
	}

	member := &models.Member{
		UserID:   userID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	t.Run("successful screen share start", func(t *testing.T) {
		service, repo, channelRepo, serverRepo, eventBus := setupScreenShareService()
		channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
		serverRepo.On("GetMember", ctx, serverID, userID).Return(member, nil)
		repo.On("GetActiveSessionByChannel", ctx, channelID).Return(nil, nil)
		repo.On("GetActiveStreamForUser", ctx, userID).Return(nil, nil)
		repo.On("CreateSession", ctx, mock.AnythingOfType("*models.StreamSession")).Return(nil)
		repo.On("GetViewerCount", ctx, mock.AnythingOfType("uuid.UUID")).Return(0, nil)
		eventBus.On("Publish", "stream.started", mock.AnythingOfType("*services.StreamStartedEvent")).Return()

		info, err := service.StartStream(ctx, channelID, userID, models.StreamTypeScreen, "1080p", 30)

		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, channelID, info.ChannelID)
		assert.Equal(t, userID, info.UserID)
		assert.Equal(t, models.StreamTypeScreen, info.StreamType)
		assert.Equal(t, "1080p", info.Resolution)
		assert.Equal(t, 30, info.FrameRate)
		assert.Equal(t, 0, info.ViewerCount)

		channelRepo.AssertExpectations(t)
		serverRepo.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("channel not found", func(t *testing.T) {
		service, _, channelRepo, _, _ := setupScreenShareService()
		channelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

		info, err := service.StartStream(ctx, channelID, userID, models.StreamTypeScreen, "1080p", 30)

		assert.Error(t, err)
		assert.Equal(t, ErrChannelNotFound, err)
		assert.Nil(t, info)

		channelRepo.AssertExpectations(t)
	})

	t.Run("not a voice channel", func(t *testing.T) {
		service, _, channelRepo, _, _ := setupScreenShareService()
		textChannel := &models.Channel{
			ID:   channelID,
			Type: models.ChannelTypeText,
		}
		channelRepo.On("GetByID", ctx, channelID).Return(textChannel, nil)

		info, err := service.StartStream(ctx, channelID, userID, models.StreamTypeScreen, "1080p", 30)

		assert.Error(t, err)
		assert.Equal(t, ErrChannelNotVoice, err)
		assert.Nil(t, info)

		channelRepo.AssertExpectations(t)
	})

	t.Run("already streaming", func(t *testing.T) {
		service, repo, channelRepo, serverRepo, _ := setupScreenShareService()
		channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
		serverRepo.On("GetMember", ctx, serverID, userID).Return(member, nil)
		existingStream := &models.StreamSession{
			ID:        uuid.New(),
			ChannelID: channelID,
			UserID:    uuid.New(),
			Status:    models.StreamStatusActive,
		}
		repo.On("GetActiveSessionByChannel", ctx, channelID).Return(existingStream, nil)

		info, err := service.StartStream(ctx, channelID, userID, models.StreamTypeScreen, "1080p", 30)

		assert.Error(t, err)
		assert.Equal(t, ErrAlreadyStreaming, err)
		assert.Nil(t, info)

		channelRepo.AssertExpectations(t)
		serverRepo.AssertExpectations(t)
		repo.AssertExpectations(t)
	})
}

func TestScreenShareService_EndStream(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()
	streamID := uuid.New()

	session := &models.StreamSession{
		ID:        streamID,
		ServerID:  serverID,
		ChannelID: channelID,
		UserID:    userID,
		Status:    models.StreamStatusActive,
	}

	t.Run("successful stream end by streamer", func(t *testing.T) {
		service, repo, _, _, eventBus := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(session, nil)
		repo.On("EndSession", ctx, streamID).Return(nil)
		eventBus.On("Publish", "stream.ended", mock.AnythingOfType("*services.StreamEndedEvent")).Return()

		err := service.EndStream(ctx, streamID, userID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("stream not found", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(nil, nil)

		err := service.EndStream(ctx, streamID, userID)

		assert.Error(t, err)
		assert.Equal(t, ErrStreamNotFound, err)

		repo.AssertExpectations(t)
	})

	t.Run("not the streamer", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		otherUserID := uuid.New()
		repo.On("GetSession", ctx, streamID).Return(session, nil)

		err := service.EndStream(ctx, streamID, otherUserID)

		assert.Error(t, err)
		assert.Equal(t, ErrNotStreamer, err)

		repo.AssertExpectations(t)
	})
}

func TestScreenShareService_JoinStream(t *testing.T) {
	ctx := context.Background()
	serverID := uuid.New()
	channelID := uuid.New()
	streamerID := uuid.New()
	viewerID := uuid.New()
	streamID := uuid.New()

	session := &models.StreamSession{
		ID:        streamID,
		ServerID:  serverID,
		ChannelID: channelID,
		UserID:    streamerID,
		Status:    models.StreamStatusActive,
	}

	member := &models.Member{
		UserID:   viewerID,
		ServerID: serverID,
		Roles:    []uuid.UUID{},
	}

	t.Run("successful join", func(t *testing.T) {
		service, repo, _, serverRepo, eventBus := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(session, nil)
		repo.On("IsViewing", ctx, streamID, viewerID).Return(false, nil)
		serverRepo.On("GetMember", ctx, serverID, viewerID).Return(member, nil)
		repo.On("AddViewer", ctx, streamID, viewerID).Return(nil)
		repo.On("GetViewerCount", ctx, streamID).Return(1, nil)
		eventBus.On("Publish", "stream.viewer_joined", mock.AnythingOfType("*services.StreamViewerJoinedEvent")).Return()

		err := service.JoinStream(ctx, streamID, viewerID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
		serverRepo.AssertExpectations(t)
	})

	t.Run("stream not found", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(nil, nil)

		err := service.JoinStream(ctx, streamID, viewerID)

		assert.Error(t, err)
		assert.Equal(t, ErrStreamNotFound, err)

		repo.AssertExpectations(t)
	})

	t.Run("cannot join own stream", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(session, nil)

		err := service.JoinStream(ctx, streamID, streamerID)

		assert.Error(t, err)
		assert.Equal(t, ErrCannotJoinOwnStream, err)

		repo.AssertExpectations(t)
	})

	t.Run("already viewing", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(session, nil)
		repo.On("IsViewing", ctx, streamID, viewerID).Return(true, nil)

		err := service.JoinStream(ctx, streamID, viewerID)

		assert.Error(t, err)
		assert.Equal(t, ErrAlreadyViewing, err)

		repo.AssertExpectations(t)
	})
}

func TestScreenShareService_LeaveStream(t *testing.T) {
	ctx := context.Background()
	streamID := uuid.New()
	viewerID := uuid.New()

	session := &models.StreamSession{
		ID:     streamID,
		UserID: uuid.New(),
		Status: models.StreamStatusActive,
	}

	t.Run("successful leave", func(t *testing.T) {
		service, repo, _, _, eventBus := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(session, nil)
		repo.On("IsViewing", ctx, streamID, viewerID).Return(true, nil)
		repo.On("RemoveViewer", ctx, streamID, viewerID).Return(nil)
		repo.On("GetViewerCount", ctx, streamID).Return(0, nil)
		eventBus.On("Publish", "stream.viewer_left", mock.AnythingOfType("*services.StreamViewerLeftEvent")).Return()

		err := service.LeaveStream(ctx, streamID, viewerID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("not viewing", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(session, nil)
		repo.On("IsViewing", ctx, streamID, viewerID).Return(false, nil)

		err := service.LeaveStream(ctx, streamID, viewerID)

		assert.Error(t, err)
		assert.Equal(t, ErrNotViewing, err)

		repo.AssertExpectations(t)
	})
}

func TestScreenShareService_GetStreamInfo(t *testing.T) {
	ctx := context.Background()
	streamID := uuid.New()
	viewerCount := 5

	session := &models.StreamSession{
		ID:         streamID,
		ServerID:   uuid.New(),
		ChannelID:  uuid.New(),
		UserID:     uuid.New(),
		StreamType: models.StreamTypeScreen,
		Status:     models.StreamStatusActive,
		Resolution: "1080p",
		FrameRate:  30,
		StartedAt:  time.Now(),
	}

	t.Run("successful get", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(session, nil)
		repo.On("GetViewerCount", ctx, streamID).Return(viewerCount, nil)

		info, err := service.GetStreamInfo(ctx, streamID)

		assert.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, streamID, info.ID)
		assert.Equal(t, viewerCount, info.ViewerCount)
		assert.Equal(t, "1080p", info.Resolution)
		assert.Equal(t, 30, info.FrameRate)

		repo.AssertExpectations(t)
	})

	t.Run("stream not found", func(t *testing.T) {
		service, repo, _, _, _ := setupScreenShareService()
		repo.On("GetSession", ctx, streamID).Return(nil, nil)

		info, err := service.GetStreamInfo(ctx, streamID)

		assert.Error(t, err)
		assert.Equal(t, ErrStreamNotFound, err)
		assert.Nil(t, info)

		repo.AssertExpectations(t)
	})
}
