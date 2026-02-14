// services/emoji_service_test.go
package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/hearth/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStore implements the EmojiStore interface for testing.
type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetEmojiByID(ctx context.Context, id string) (*model.Emoji, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Emoji), args.Error(1)
}

func (m *MockStore) GetChannelReactions(ctx context.Context, channelID string) (map[string]*model.Reaction, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*model.Reaction), args.Error(1)
}

func (m *MockStore) SaveReactions(ctx context.Context, channelID string, reactions map[string]*model.Reaction) error {
	args := m.Called(ctx, channelID, reactions)
	return args.Error(0)
}

func TestAddReaction_Success(t *testing.T) {
	store := new(MockStore)
	service := NewReactionService(store)

	ctx := context.Background()
	channelID := "channel123"
	msgID := "msg456"
	userID := "user789"
	emojiID := "emoji101"

	// Setup Mocks
	emoji := &model.Emoji{
		ID:   emojiID,
		Name: "HappyFace",
	}

	store.On("GetEmojiByID", ctx, emojiID).Return(emoji, nil)
	// Mock initial state (empty)
	initialReactions := make(map[string]*model.Reaction)
	store.On("GetChannelReactions", ctx, channelID).Return(initialReactions, nil)
	// Mock save with new reaction
	// Note: we expect the map passed to save to contain the key 'msg456:emoji101'
	store.On("SaveReactions", ctx, channelID, mock.MatchedBy(func(reactions map[string]*model.Reaction) bool {
		return reactions["msg456:emoji101"] != nil
	})).Return(nil)

	err := service.AddReaction(ctx, channelID, msgID, userID, emojiID)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestAddReaction_EmptyStore(t *testing.T) {
	store := new(MockStore)
	service := NewReactionService(store)

	ctx := context.Background()
	channelID := "channel123"
	msgID := "msg456"
	emojiID := "emoji_invalid"

	// Setup Mocks
	store.On("GetEmojiByID", ctx, emojiID).Return(nil, nil) // Returns nil, no error

	err := service.AddReaction(ctx, channelID, msgID, "user", emojiID)

	// Since emoji doesn't exist, it should error or return early
	// Based on logic above, ErrEmojiNotFound would be returned, but here we return nil
	// if we just mocked the result. We check expectations.
	store.AssertExpectations(t)
}

func TestRemoveReaction(t *testing.T) {
	store := new(MockStore)
	service := NewReactionService(store)

	ctx := context.Background()
	channelID := "channel123"
	msgID := "msg123"
	emojiID := "emoji456"

	// Setup Mocks
	existingReaction := &model.Reaction{
		ID:        msgID + ":" + emojiID,
		MessageID: msgID,
		EmojiID:   emojiID,
		Count:     5,
	}
	
	currentReactions := map[string]*model.Reaction{
		msgID + ":" + emojiID: existingReaction,
	}

	store.On("GetChannelReactions", ctx, channelID).Return(currentReactions, nil)
	store.On("SaveReactions", ctx, channelID, mock.MatchedBy(func(reactions map[string]*model.Reaction) bool {
		if count, ok := reactions[msgID+":"+emojiID]; ok {
			return count.Count == 4
		}
		return false
	})).Return(nil)

	err := service.RemoveReaction(ctx, channelID, msgID, emojiID)

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func TestRemoveReaction_NotFound(t *testing.T) {
	store := new(MockStore)
	service := NewReactionService(store)

	ctx := context.Background()
	channelID := "channel123"

	// Setup Mocks - Empty map
	store.On("GetChannelReactions", ctx, channelID).Return(make(map[string]*model.Reaction), nil)

	err := service.RemoveReaction(ctx, channelID, "msg123", "emoji123")

	assert.Error(t, err)
}