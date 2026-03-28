package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"hearth/internal/models"
)

// RedisCache implements CacheService using Redis
type RedisCache struct {
	client *redis.Client
	prefix string
}

// Client returns the underlying Redis client for advanced operations
func (c *RedisCache) Client() *redis.Client {
	return c.client
}

// URL returns a Redis URL that can be used to create new connections
// Useful for creating separate pub/sub connections
func (c *RedisCache) URL() string {
	opts := c.client.Options()
	return "redis://" + opts.Addr + "/" + "0"
}

// NewRedisCache creates a new Redis cache client
func NewRedisCache(redisURL string) (*RedisCache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
		prefix: "hearth:",
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Generic operations

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, c.prefix+key).Bytes()
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, c.prefix+key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.prefix+key).Err()
}

// User caching

func (c *RedisCache) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	data, err := c.Get(ctx, "user:"+id.String())
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *RedisCache) SetUser(ctx context.Context, user *models.User, ttl time.Duration) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return c.Set(ctx, "user:"+user.ID.String(), data, ttl)
}

func (c *RedisCache) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return c.Delete(ctx, "user:"+id.String())
}

// Server caching

func (c *RedisCache) GetServer(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	data, err := c.Get(ctx, "server:"+id.String())
	if err != nil {
		return nil, err
	}

	var server models.Server
	if err := json.Unmarshal(data, &server); err != nil {
		return nil, err
	}

	return &server, nil
}

func (c *RedisCache) SetServer(ctx context.Context, server *models.Server, ttl time.Duration) error {
	data, err := json.Marshal(server)
	if err != nil {
		return err
	}

	return c.Set(ctx, "server:"+server.ID.String(), data, ttl)
}

func (c *RedisCache) DeleteServer(ctx context.Context, id uuid.UUID) error {
	return c.Delete(ctx, "server:"+id.String())
}

// Channel caching

func (c *RedisCache) GetChannel(ctx context.Context, id uuid.UUID) (*models.Channel, error) {
	data, err := c.Get(ctx, "channel:"+id.String())
	if err != nil {
		return nil, err
	}

	var channel models.Channel
	if err := json.Unmarshal(data, &channel); err != nil {
		return nil, err
	}

	return &channel, nil
}

func (c *RedisCache) SetChannel(ctx context.Context, channel *models.Channel, ttl time.Duration) error {
	data, err := json.Marshal(channel)
	if err != nil {
		return err
	}

	return c.Set(ctx, "channel:"+channel.ID.String(), data, ttl)
}

func (c *RedisCache) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	return c.Delete(ctx, "channel:"+id.String())
}

// Rate limiting

func (c *RedisCache) IncrementWithExpiry(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, c.prefix+key)
	pipe.Expire(ctx, c.prefix+key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// Presence caching - stores full Presence struct as JSON

// GetPresence retrieves a user's presence from cache.
// Returns an offline Presence on cache miss.
func (c *RedisCache) GetPresence(ctx context.Context, userID uuid.UUID) (*models.Presence, error) {
	data, err := c.Get(ctx, "presence:"+userID.String())
	if err != nil {
		// Cache miss - return offline presence
		return &models.Presence{
			UserID: userID,
			Status: models.StatusOffline,
		}, nil
	}

	var presence models.Presence
	if err := json.Unmarshal(data, &presence); err != nil {
		// Invalid data - return offline presence
		return &models.Presence{
			UserID: userID,
			Status: models.StatusOffline,
		}, nil
	}

	return &presence, nil
}

// SetPresence stores a user's full presence struct in cache
func (c *RedisCache) SetPresence(ctx context.Context, userID uuid.UUID, presence *models.Presence, ttl time.Duration) error {
	data, err := json.Marshal(presence)
	if err != nil {
		return err
	}

	return c.Set(ctx, "presence:"+userID.String(), data, ttl)
}

// DeletePresence removes a user's presence from cache
func (c *RedisCache) DeletePresence(ctx context.Context, userID uuid.UUID) error {
	return c.Delete(ctx, "presence:"+userID.String())
}

// GetPresenceBulk retrieves presence for multiple users using MGET for efficiency.
// Returns a map of userID to Presence. Users not in cache will have offline presence.
func (c *RedisCache) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Presence, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*models.Presence), nil
	}

	// Build keys
	keys := make([]string, len(userIDs))
	keyToUserID := make(map[string]uuid.UUID)
	for i, userID := range userIDs {
		key := c.prefix + "presence:" + userID.String()
		keys[i] = key
		keyToUserID[key] = userID
	}

	// Use MGET for efficiency
	results, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// Build response map
	presences := make(map[uuid.UUID]*models.Presence)
	for i, result := range results {
		userID := userIDs[i]
		if result == nil {
			// Cache miss
			presences[userID] = &models.Presence{
				UserID: userID,
				Status: models.StatusOffline,
			}
			continue
		}

		str, ok := result.(string)
		if !ok {
			presences[userID] = &models.Presence{
				UserID: userID,
				Status: models.StatusOffline,
			}
			continue
		}

		var presence models.Presence
		if err := json.Unmarshal([]byte(str), &presence); err != nil {
			presences[userID] = &models.Presence{
				UserID: userID,
				Status: models.StatusOffline,
			}
			continue
		}

		presences[userID] = &presence
	}

	return presences, nil
}

// Session caching

// GetSession retrieves a session from cache
func (c *RedisCache) GetSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	data, err := c.Get(ctx, "session:"+sessionID.String())
	if err != nil {
		return nil, err
	}

	var session models.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// SetSession stores a session in cache
func (c *RedisCache) SetSession(ctx context.Context, session *models.Session, ttl time.Duration) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return c.Set(ctx, "session:"+session.ID.String(), data, ttl)
}

// DeleteSession removes a session from cache
func (c *RedisCache) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	return c.Delete(ctx, "session:"+sessionID.String())
}

// Pub/Sub for real-time events

func (c *RedisCache) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.client.Publish(ctx, c.prefix+channel, data).Err()
}

func (c *RedisCache) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.client.Subscribe(ctx, c.prefix+channel)
}
