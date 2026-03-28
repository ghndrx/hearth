package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"hearth/internal/models"
)

// MockMessageServiceForAnnouncement is a mock implementation of MessageServiceForAnnouncement
type MockMessageServiceForAnnouncement struct {
	mock.Mock
}

func (m *MockMessageServiceForAnnouncement) SendMessageForWebhook(ctx context.Context, req SendWebhookMessageRequest) (*models.Message, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func TestFollowChannel_Success(t *testing.T) {
	ctx := context.Background()
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	messageService := new(MockMessageServiceForAnnouncement)
	eventBus := new(MockEventBus)

	service := NewAnnouncementService(webhookRepo, channelRepo, serverRepo, messageService, nil, eventBus)

	// Create source announcement channel
	sourceServerID := uuid.New()
	targetServerID := uuid.New()
	sourceChannelID := uuid.New()
	targetChannelID := uuid.New()
	userID := uuid.New()

	sourceChannel := &models.Channel{
		ID:       sourceChannelID,
		ServerID: &sourceServerID,
		Type:     models.ChannelTypeAnnouncement,
		Name:     "announcements",
	}
	channelRepo.On("GetByID", mock.Anything, sourceChannelID).Return(sourceChannel, nil)

	targetChannel := &models.Channel{
		ID:       targetChannelID,
		ServerID: &targetServerID,
		Type:     models.ChannelTypeText,
		Name:     "general",
	}
	channelRepo.On("GetByID", mock.Anything, targetChannelID).Return(targetChannel, nil)

	sourceServer := &models.Server{ID: sourceServerID, Name: "Source Server"}
	serverRepo.On("GetByID", ctx, sourceServerID).Return(sourceServer, nil)

	// Add user as member of target server
	serverRepo.On("GetMember", mock.Anything, targetServerID, userID).Return(&models.Member{ServerID: targetServerID, UserID: userID}, nil)

	// Expect webhook creation
	webhookRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	webhookRepo.On("GetByChannelID", mock.Anything, sourceChannelID).Return([]*models.Webhook{}, nil)

	// Expect event bus publish
	eventBus.On("Publish", "channel.followed", mock.Anything).Return()

	// Expect channel.followed event
	eventBus.On("Publish", "channel.followed", mock.AnythingOfType("*services.ChannelFollowedEvent")).Return()

	// Follow channel
	webhook, err := service.FollowChannel(ctx, sourceChannelID, targetChannelID, userID)

	assert.NoError(t, err)
	assert.NotNil(t, webhook)
	assert.Equal(t, models.WebhookTypeChannelFollower, webhook.Type)
	assert.NotNil(t, webhook.SourceChannelID)
	assert.Equal(t, sourceChannelID, *webhook.SourceChannelID)
	assert.Equal(t, targetChannelID, webhook.ChannelID)

	webhookRepo.AssertExpectations(t)
	channelRepo.AssertExpectations(t)
	serverRepo.AssertExpectations(t)
}

func TestFollowChannel_NotAnnouncementChannel(t *testing.T) {
	ctx := context.Background()
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	messageService := new(MockMessageServiceForAnnouncement)
	eventBus := new(MockEventBus)

	service := NewAnnouncementService(webhookRepo, channelRepo, serverRepo, messageService, nil, eventBus)

	// Create a non-announcement channel
	channelID := uuid.New()
	serverID := uuid.New()
	userID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
		Name:     "general",
	}
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	// Try to follow a non-announcement channel
	_, err := service.FollowChannel(ctx, channelID, uuid.New(), userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotAnnouncementChannel, err)
}

func TestFollowChannel_CannotFollowOwnChannel(t *testing.T) {
	ctx := context.Background()
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	messageService := new(MockMessageServiceForAnnouncement)
	eventBus := new(MockEventBus)

	service := NewAnnouncementService(webhookRepo, channelRepo, serverRepo, messageService, nil, eventBus)

	// Create source and target channels in same server
	serverID := uuid.New()
	sourceChannelID := uuid.New()
	targetChannelID := uuid.New()
	userID := uuid.New()

	sourceChannel := &models.Channel{
		ID:       sourceChannelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeAnnouncement,
		Name:     "announcements",
	}
	channelRepo.On("GetByID", ctx, sourceChannelID).Return(sourceChannel, nil)

	targetChannel := &models.Channel{
		ID:       targetChannelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
		Name:     "general",
	}
	channelRepo.On("GetByID", ctx, targetChannelID).Return(targetChannel, nil)

	server := &models.Server{ID: serverID, Name: "Test Server"}
	serverRepo.On("GetByID", ctx, serverID).Return(server, nil)

	// Add user as member
	serverRepo.On("GetMember", ctx, serverID, userID).Return(&models.Member{ServerID: serverID, UserID: userID}, nil)

	// Try to follow channel in same server
	_, err := service.FollowChannel(ctx, sourceChannelID, targetChannelID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrCannotFollowOwnChannel, err)
}

func TestUnfollowChannel_Success(t *testing.T) {
	ctx := context.Background()
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	messageService := new(MockMessageServiceForAnnouncement)
	eventBus := new(MockEventBus)

	service := NewAnnouncementService(webhookRepo, channelRepo, serverRepo, messageService, nil, eventBus)

	// Setup
	sourceServerID := uuid.New()
	targetServerID := uuid.New()
	sourceChannelID := uuid.New()
	targetChannelID := uuid.New()
	webhookID := uuid.New()
	userID := uuid.New()

	sourceChannel := &models.Channel{
		ID:       sourceChannelID,
		ServerID: &sourceServerID,
		Type:     models.ChannelTypeAnnouncement,
		Name:     "announcements",
	}
	channelRepo.On("GetByID", ctx, sourceChannelID).Return(sourceChannel, nil)

	webhook := &models.Webhook{
		ID:              webhookID,
		Type:            models.WebhookTypeChannelFollower,
		ServerID:        &targetServerID,
		ChannelID:       targetChannelID,
		SourceChannelID: &sourceChannelID,
		SourceServerID:  &sourceServerID,
	}
	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	webhookRepo.On("Delete", ctx, webhookID).Return(nil)

	// Unfollow
	err := service.UnfollowChannel(ctx, sourceChannelID, webhookID, userID)

	assert.NoError(t, err)
	webhookRepo.AssertExpectations(t)
}

func TestUnfollowChannel_NotFollower(t *testing.T) {
	ctx := context.Background()
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	messageService := new(MockMessageServiceForAnnouncement)
	eventBus := new(MockEventBus)

	service := NewAnnouncementService(webhookRepo, channelRepo, serverRepo, messageService, nil, eventBus)

	// Setup
	sourceServerID := uuid.New()
	sourceChannelID := uuid.New()
	webhookID := uuid.New()
	userID := uuid.New()

	sourceChannel := &models.Channel{
		ID:       sourceChannelID,
		ServerID: &sourceServerID,
		Type:     models.ChannelTypeAnnouncement,
		Name:     "announcements",
	}
	channelRepo.On("GetByID", ctx, sourceChannelID).Return(sourceChannel, nil)

	// Create webhook that's NOT a follower of this channel
	otherChannelID := uuid.New()
	webhook := &models.Webhook{
		ID:        webhookID,
		Type:      models.WebhookTypeIncoming,
		ChannelID: otherChannelID,
	}
	webhookRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)

	// Try to unfollow
	err := service.UnfollowChannel(ctx, sourceChannelID, webhookID, userID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotFollower, err)
}

func TestGetFollowers_Success(t *testing.T) {
	ctx := context.Background()
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	messageService := new(MockMessageServiceForAnnouncement)
	eventBus := new(MockEventBus)

	service := NewAnnouncementService(webhookRepo, channelRepo, serverRepo, messageService, nil, eventBus)

	// Setup announcement channel
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeAnnouncement,
		Name:     "announcements",
	}
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	// Add some follower webhooks
	followers := []*models.Webhook{
		{ID: uuid.New(), Type: models.WebhookTypeChannelFollower, ChannelID: uuid.New(), SourceChannelID: &channelID},
		{ID: uuid.New(), Type: models.WebhookTypeChannelFollower, ChannelID: uuid.New(), SourceChannelID: &channelID},
		{ID: uuid.New(), Type: models.WebhookTypeChannelFollower, ChannelID: uuid.New(), SourceChannelID: &channelID},
	}
	webhookRepo.On("GetByChannelID", ctx, channelID).Return(followers, nil)

	// Get followers
	result, err := service.GetFollowers(ctx, channelID)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	webhookRepo.AssertExpectations(t)
}

func TestGetFollowers_NotAnnouncementChannel(t *testing.T) {
	ctx := context.Background()
	webhookRepo := new(MockWebhookRepository)
	channelRepo := new(MockChannelRepository)
	serverRepo := new(MockServerRepository)
	messageService := new(MockMessageServiceForAnnouncement)
	eventBus := new(MockEventBus)

	service := NewAnnouncementService(webhookRepo, channelRepo, serverRepo, messageService, nil, eventBus)

	// Setup regular channel
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
		Name:     "general",
	}
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	// Try to get followers
	_, err := service.GetFollowers(ctx, channelID)

	assert.Error(t, err)
	assert.Equal(t, ErrNotAnnouncementChannel, err)
}
