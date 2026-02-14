package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Constants defining the service limits
const (
	MaxGuildMembers = 50_000
	UserCacheTTL    = 5 * time.Minute
)

var (
	// ErrMemberNotFound is an error used when a user is not found in a database.
	ErrMemberNotFound = errors.New("member not found")
)

// --- Interfaces ---

// MemberStore defines the contract for persisting and fetching guild members.
// In a real app, this would be an interface for your Database client.
type MemberStore interface {
	// FetchByID retrieves a member by ID. 
	// Returns ErrMemberNotFound if the member does not exist.
	FetchByID(ctx context.Context, guildID, memberID string) (*GuildMember, error)
}

// EventBus defines how asynchronous events are emitted.
type EventBus interface {
	Publish(topic string, payload interface{}) error
}

// --- Domain Models ---

// GuildMember represents a user in the context of a Discord-like guild.
type GuildMember struct {
	ID       string
	Name     string
	RoleIDs  []string
	JoinedAt time.Time
}

// --- Service ---

// MemberService handles business logic for guild memberships and provides caching.
type MemberService struct {
	memberStore MemberStore
	eventBus    EventBus
	
	// Cache tracking
	membersMap sync.Map // map[string]*GuildMember (key is "guildID:memberID")
}

// NewMemberService creates a new instance of the MemberService.
func NewMemberService(store MemberStore, bus EventBus) *MemberService {
	return &MemberService{
		memberStore: store,
		eventBus:    bus,
	}
}

// GetMember retrieves a member. It checks the in-memory cache first, 
// then falls back to the database.
func (s *MemberService) GetMember(ctx context.Context, guildID, memberID string) (*GuildMember, error) {
	cacheKey := cacheKey(guildID, memberID)

	// 1. Try to load from cache
	val, found := s.membersMap.Load(cacheKey)
	if found {
		return val.(*GuildMember), nil
	}

	// 2. If not found in cache, load from database
	member, err := s.memberStore.FetchByID(ctx, guildID, memberID)
	if err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			// Optional: Load a "placeholder" into cache so we don't hammer the DB
			// for missing users. (Remember to set a short TTL for this)
			s.membersMap.Store(cacheKey, nil)
			return nil, ErrMemberNotFound
		}
		return nil, err
	}

	// 3. Cache the result
	s.membersMap.Store(cacheKey, member)

	// 4. Return the member
	return member, nil
}

// UpdateMember processes a logic change and emits an event.
func (s *MemberService) UpdateMember(ctx context.Context, guildID, memberID string, newName string) error {
	// In a real app, you might call memberStore.Update here...
	
	// For this example, we simulate a DB lookup to ensure consistency
	member, err := s.memberStore.FetchByID(ctx, guildID, memberID)
	if err != nil {
		return err
	}

	// Business logic: Update Role or Name
	member.Name = newName
	
	// Publish async event ("MemberUpdated")
	if s.eventBus != nil {
		go s.eventBus.Publish("member.updated", guildID)
	}

	return nil
}

// cacheKey constructs a composite key for the sync.Map.
func cacheKey(guildID, memberID string) string {
	return fmt.Sprintf("%s:%s", guildID, memberID)
}

// CleanupCache is a method to periodically clean up stale cache entries.
func (s *MemberService) CleanupCache() {
	ticker := time.NewTicker(UserCacheTTL / 2)
	defer ticker.Stop()

	for range ticker.C {
		// Implementation simplified for example:
		// In production, you'd use a linked hash map structure 
		// to precisely remove items by AccessTime.
	}
}