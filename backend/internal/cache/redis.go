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

// Presence

func (c *RedisCache) SetPresence(ctx context.Context, userID uuid.UUID, status string, ttl time.Duration) error {
	return c.Set(ctx, "presence:"+userID.String(), []byte(status), ttl)
}

func (c *RedisCache) GetPresence(ctx context.Context, userID uuid.UUID) (string, error) {
	data, err := c.Get(ctx, "presence:"+userID.String())
	if err != nil {
		return "offline", nil
	}
	return string(data), nil
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
