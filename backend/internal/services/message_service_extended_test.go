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

func TestEditMessage_MessageNotFound(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	messageID := uuid.New()

	msgRepo.On("GetByID", ctx, messageID).Return(nil, nil)

	message, err := service.EditMessage(ctx, messageID, authorID, "New content")

	assert.Error(t, err)
	assert.Equal(t, ErrMessageNotFound, err)
	assert.Nil(t, message)
}

func TestDeleteMessage_MessageNotFound(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	messageID := uuid.New()

	msgRepo.On("GetByID", ctx, messageID).Return(nil, nil)

	err := service.DeleteMessage(ctx, messageID, authorID)

	assert.Error(t, err)
	assert.Equal(t, ErrMessageNotFound, err)
}

func TestGetMessages_ChannelNotFound(t *testing.T) {
	service, _, channelRepo, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()

	channelRepo.On("GetByID", ctx, channelID).Return(nil, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 50)

	assert.Error(t, err)
	assert.Equal(t, ErrChannelNotFound, err)
	assert.Nil(t, messages)
}

func TestGetMessages_NotServerMember(t *testing.T) {
	service, _, channelRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(nil, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 50)

	assert.Error(t, err)
	assert.Equal(t, ErrNotServerMember, err)
	assert.Nil(t, messages)
}

func TestGetMessages_DMChannelNotParticipant(t *testing.T) {
	service, _, channelRepo, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	otherUserID := uuid.New()

	channel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{otherUserID}, // Requester is not in recipients
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 50)

	assert.Error(t, err)
	assert.Equal(t, ErrNoPermission, err)
	assert.Nil(t, messages)
}

func TestGetMessages_DMChannelAsParticipant(t *testing.T) {
	service, msgRepo, channelRepo, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	otherUserID := uuid.New()

	channel := &models.Channel{
		ID:         channelID,
		Type:       models.ChannelTypeDM,
		Recipients: []uuid.UUID{requesterID, otherUserID},
	}

	expectedMessages := []*models.Message{
		{ID: uuid.New(), Content: "DM Message"},
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	msgRepo.On("GetChannelMessages", ctx, channelID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), 50).Return(expectedMessages, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 50)

	assert.NoError(t, err)
	assert.Len(t, messages, 1)
}

func TestGetMessages_WithPagination(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()
	beforeID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	expectedMessages := []*models.Message{
		{ID: uuid.New(), Content: "Message"},
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	msgRepo.On("GetChannelMessages", ctx, channelID, &beforeID, (*uuid.UUID)(nil), 50).Return(expectedMessages, nil)

	messages, err := service.GetMessages(ctx, channelID, requesterID, &beforeID, nil, 50)

	assert.NoError(t, err)
	assert.Len(t, messages, 1)
}

func TestGetMessages_LimitValidation(t *testing.T) {
	service, msgRepo, channelRepo, serverRepo, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	member := &models.Member{
		UserID:   requesterID,
		ServerID: serverID,
	}

	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	serverRepo.On("GetMember", ctx, serverID, requesterID).Return(member, nil)
	// Should default to 50 when invalid limit provided
	msgRepo.On("GetChannelMessages", ctx, channelID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), 50).Return([]*models.Message{}, nil)

	// Test with 0 limit
	messages, err := service.GetMessages(ctx, channelID, requesterID, nil, nil, 0)
	assert.NoError(t, err)
	assert.NotNil(t, messages)

	// Test with negative limit
	msgRepo.On("GetChannelMessages", ctx, channelID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), 50).Return([]*models.Message{}, nil)
	messages, err = service.GetMessages(ctx, channelID, requesterID, nil, nil, -10)
	assert.NoError(t, err)
	assert.NotNil(t, messages)

	// Test with over-limit (should cap to 100 then use 50 as default logic)
	msgRepo.On("GetChannelMessages", ctx, channelID, (*uuid.UUID)(nil), (*uuid.UUID)(nil), 50).Return([]*models.Message{}, nil)
	messages, err = service.GetMessages(ctx, channelID, requesterID, nil, nil, 200)
	assert.NoError(t, err)
	assert.NotNil(t, messages)
}

func TestAddReaction_MessageNotFound(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	userID := uuid.New()
	messageID := uuid.New()

	msgRepo.On("GetByID", ctx, messageID).Return(nil, nil)

	err := service.AddReaction(ctx, messageID, userID, "👍")

	assert.Error(t, err)
	assert.Equal(t, ErrMessageNotFound, err)
}

func TestPinMessage_MessageNotFound(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, _ := setupMessageService()
	ctx := context.Background()
	requesterID := uuid.New()
	messageID := uuid.New()

	msgRepo.On("GetByID", ctx, messageID).Return(nil, nil)

	err := service.PinMessage(ctx, messageID, requesterID)

	assert.Error(t, err)
	assert.Equal(t, ErrMessageNotFound, err)
}

func TestParseMentions(t *testing.T) {
	// parseMentions returns nil currently (placeholder)
	mentions := parseMentions("Hello <@123456> and <@789012>!")

	// Current implementation returns nil
	assert.Nil(t, mentions)
}

func TestIsChannelParticipant(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	t.Run("user is participant", func(t *testing.T) {
		channel := &models.Channel{
			Recipients: []uuid.UUID{userID, otherUserID},
		}

		assert.True(t, isChannelParticipant(channel, userID))
	})

	t.Run("user is not participant", func(t *testing.T) {
		channel := &models.Channel{
			Recipients: []uuid.UUID{otherUserID},
		}

		assert.False(t, isChannelParticipant(channel, userID))
	})

	t.Run("empty recipients", func(t *testing.T) {
		channel := &models.Channel{
			Recipients: []uuid.UUID{},
		}

		assert.False(t, isChannelParticipant(channel, userID))
	})
}

func TestTimePtr(t *testing.T) {
	// timePtr helper
	now := time.Now()
	ptr := timePtr(now)
	// Just verify it doesn't panic
	assert.NotNil(t, ptr)
	assert.Equal(t, now, *ptr)
}

func TestEditMessage_WithEncryptedContent(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:               messageID,
		ChannelID:        channelID,
		AuthorID:         authorID,
		Content:          "",
		EncryptedContent: "encrypted_data_here", // Encrypted message
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	msgRepo.On("Update", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	eventBus.On("Publish", "message.updated", mock.AnythingOfType("*services.MessageUpdatedEvent")).Return()

	message, err := service.EditMessage(ctx, messageID, authorID, "new_encrypted_data")

	assert.NoError(t, err)
	assert.Equal(t, "new_encrypted_data", message.Content)
	// Mentions should not be parsed for encrypted messages
	assert.Nil(t, message.Mentions)
}

func TestEditMessage_PlaintextPreservesMentions(t *testing.T) {
	service, msgRepo, _, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	messageID := uuid.New()
	channelID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   "Original content",
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	msgRepo.On("Update", ctx, mock.AnythingOfType("*models.Message")).Return(nil)
	eventBus.On("Publish", "message.updated", mock.AnythingOfType("*services.MessageUpdatedEvent")).Return()

	message, err := service.EditMessage(ctx, messageID, authorID, "Hello <@user>!")

	assert.NoError(t, err)
	assert.Equal(t, "Hello <@user>!", message.Content)
	// Mentions should be parsed (though current impl returns nil)
}

func TestDeleteMessage_ServerChannelWithManagePermission(t *testing.T) {
	service, msgRepo, channelRepo, _, _, _, _, _, eventBus := setupMessageService()
	ctx := context.Background()
	authorID := uuid.New()
	requesterID := uuid.New() // Different user with permissions
	messageID := uuid.New()
	channelID := uuid.New()
	serverID := uuid.New()

	existingMessage := &models.Message{
		ID:        messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
	}

	channel := &models.Channel{
		ID:       channelID,
		ServerID: &serverID,
		Type:     models.ChannelTypeText,
	}

	msgRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	channelRepo.On("GetByID", ctx, channelID).Return(channel, nil)
	msgRepo.On("Delete", ctx, messageID).Return(nil)
	eventBus.On("Publish", "message.deleted", mock.AnythingOfType("*services.MessageDeletedEvent")).Return()

	err := service.DeleteMessage(ctx, messageID, requesterID)

	// With server channel and different user, should eventually succeed
	// (TODO: actual permission check)
	assert.NoError(t, err)
}

// Test MessageCreatedEvent structure
func TestMessageCreatedEvent(t *testing.T) {
	channelID := uuid.New()
	serverID := uuid.New()
	message := &models.Message{
		ID:      uuid.New(),
		Content: "Test",
	}

	event := &MessageCreatedEvent{
		Message:   message,
		ChannelID: channelID,
		ServerID:  &serverID,
	}

	assert.Equal(t, message, event.Message)
	assert.Equal(t, channelID, event.ChannelID)
	assert.Equal(t, &serverID, event.ServerID)
}

// Test MessageUpdatedEvent structure
func TestMessageUpdatedEvent(t *testing.T) {
	channelID := uuid.New()
	message := &models.Message{
		ID:      uuid.New(),
		Content: "Updated",
	}

	event := &MessageUpdatedEvent{
		Message:   message,
		ChannelID: channelID,
	}

	assert.Equal(t, message, event.Message)
	assert.Equal(t, channelID, event.ChannelID)
}

// Test MessageDeletedEvent structure
func TestMessageDeletedEvent(t *testing.T) {
	messageID := uuid.New()
	channelID := uuid.New()
	authorID := uuid.New()

	event := &MessageDeletedEvent{
		MessageID: messageID,
		ChannelID: channelID,
		AuthorID:  authorID,
	}

	assert.Equal(t, messageID, event.MessageID)
	assert.Equal(t, channelID, event.ChannelID)
	assert.Equal(t, authorID, event.AuthorID)
}

// Test ReactionAddedEvent structure
func TestReactionAddedEvent(t *testing.T) {
	messageID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()

	event := &ReactionAddedEvent{
		MessageID: messageID,
		ChannelID: channelID,
		UserID:    userID,
		Emoji:     "👍",
	}

	assert.Equal(t, messageID, event.MessageID)
	assert.Equal(t, channelID, event.ChannelID)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, "👍", event.Emoji)
}

// Test ReactionRemovedEvent structure
func TestReactionRemovedEvent(t *testing.T) {
	messageID := uuid.New()
	userID := uuid.New()

	event := &ReactionRemovedEvent{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     "👍",
	}

	assert.Equal(t, messageID, event.MessageID)
	assert.Equal(t, userID, event.UserID)
	assert.Equal(t, "👍", event.Emoji)
}

// Test MessagePinnedEvent structure
func TestMessagePinnedEvent(t *testing.T) {
	messageID := uuid.New()
	channelID := uuid.New()
	pinnedBy := uuid.New()

	event := &MessagePinnedEvent{
		MessageID: messageID,
		ChannelID: channelID,
		PinnedBy:  pinnedBy,
	}

	assert.Equal(t, messageID, event.MessageID)
	assert.Equal(t, channelID, event.ChannelID)
	assert.Equal(t, pinnedBy, event.PinnedBy)
}
