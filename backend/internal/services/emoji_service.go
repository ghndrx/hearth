// services/emoji_service.go
package services

import (
	"context"
	"sync"

	"github.com/hearth/core/model" // Assuming a shared model package for types
)

// EmojiStore defines the interface for storing and retrieving emoji data.
type EmojiStore interface {
	GetEmojiByID(ctx context.Context, id string) (*model.Emoji, error)
	GetChannelReactions(ctx context.Context, channelID string) (map[string]*model.Reaction, error)
	SaveReactions(ctx context.Context, channelID string, reactions map[string]*model.Reaction) error
}

// ReactionService handles logic related to message reactions.
type ReactionService struct {
	store EmojiStore
	mu    sync.RWMutex // Protects the in-memory usage tracking if necessary, though logic is mostly IO
}

// NewReactionService creates a new ReactionService.
func NewReactionService(store EmojiStore) *ReactionService {
	return &ReactionService{
		store: store,
	}
}

// AddReaction adds a reaction to a message in a channel.
// It handles checking if the emoji exists and incrementing the count.
func (s *ReactionService) AddReaction(ctx context.Context, channelID, messageID, userID, emojiID string) error {
	// 1. Validate Emoji Exists
	emoji, err := s.store.GetEmojiByID(ctx, emojiID)
	if err != nil {
		return err
	}
	if emoji == nil {
		return model.ErrEmojiNotFound
	}

	// 2. Lock for writing to avoid race conditions with GetReactions
	s.mu.Lock()
	defer s.mu.Unlock()

	// 3. Retrieve current reactions
	reactions, err := s.store.GetChannelReactions(ctx, channelID)
	if err != nil {
		return err
	}

	// 4. Update or Add Reaction
	key := messageID + ":" + emojiID
	if existing, ok := reactions[key]; ok {
		// User has already reacted, do nothing (or update tenant/user if implemented)
		return nil
	}

	reactions[key] = &model.Reaction{
		ID:        key,
		MessageID: messageID,
		EmojiID:   emojiID,
		Count:     1,
		// In a real app, we would track users who reacted here
	}

	return s.store.SaveReactions(ctx, channelID, reactions)
}

// RemoveReaction removes a specific reaction from a message.
// Effectively decrements the count or removes the key if it hits zero.
func (s *ReactionService) RemoveReaction(ctx context.Context, channelID, messageID, emojiID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reactions, err := s.store.GetChannelReactions(ctx, channelID)
	if err != nil {
		return err
	}

	key := messageID + ":" + emojiID
	if reaction, ok := reactions[key]; ok {
		reaction.Count--

		// Clean up if count drops to zero to save space (optional)
		if reaction.Count <= 0 {
			delete(reactions, key)
		}

		return s.store.SaveReactions(ctx, channelID, reactions)
	}

	return model.ErrReactionNotFound
}

// GetReactions returns a map of reactions for all messages in a specific channel.
// The map keys are constructed as "messageID:emojiID".
func (s *ReactionService) GetReactions(ctx context.Context, channelID string) (map[string]*model.Reaction, error) {
	// Read lock allows concurrent reads
	s.mu.RLock()
	defer s.mu.RUnlock()

	reactions, err := s.store.GetChannelReactions(ctx, channelID)
	if err != nil {
		return nil, err
	}

	return reactions, nil
}