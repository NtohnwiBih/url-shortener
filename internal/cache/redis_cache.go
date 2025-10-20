package cache

import (
	"context"
	"fmt"
	"time"
	
	"github.com/redis/go-redis/v9"
)

// redisCache implements the Cache interface using Redis
type redisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache client
// Returns error if connection fails
func NewRedisCache(addr, password string, db int) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10, // Connection pool size
		MinIdleConns: 5,  // Minimum idle connections
	})
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	return &redisCache{client: client}, nil
}

// Set stores a key-value pair in Redis with TTL
// Uses SET command with EX option for atomic operation
func (c *redisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	// Add prefix to avoid key collisions with other applications
	prefixedKey := c.prefixKey(key)
	
	err := c.client.Set(ctx, prefixedKey, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}
	
	return nil
}

// Get retrieves a value from Redis by key
// Returns empty string if key doesn't exist (not an error)
func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	prefixedKey := c.prefixKey(key)
	
	val, err := c.client.Get(ctx, prefixedKey).Result()
	if err == redis.Nil {
		// Key doesn't exist - return empty string, not an error
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	
	return val, nil
}

// Delete removes a key from Redis
func (c *redisCache) Delete(ctx context.Context, key string) error {
	prefixedKey := c.prefixKey(key)
	
	err := c.client.Del(ctx, prefixedKey).Err()
	if err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	
	return nil
}

// Exists checks if a key exists in Redis
func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	prefixedKey := c.prefixKey(key)
	
	count, err := c.client.Exists(ctx, prefixedKey).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists check failed: %w", err)
	}
	
	return count > 0, nil
}

// Close closes the Redis connection
func (c *redisCache) Close() error {
	return c.client.Close()
}

// prefixKey adds a namespace prefix to avoid key collisions
func (c *redisCache) prefixKey(key string) string {
	return fmt.Sprintf("urlshortener:%s", key)
}

// Batch operations for performance optimization

// SetMultiple stores multiple key-value pairs in a single pipeline
// More efficient than multiple Set calls
func (c *redisCache) SetMultiple(ctx context.Context, items map[string]string, ttl time.Duration) error {
	pipe := c.client.Pipeline()
	
	for key, value := range items {
		prefixedKey := c.prefixKey(key)
		pipe.Set(ctx, prefixedKey, value, ttl)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline failed: %w", err)
	}
	
	return nil
}

// GetMultiple retrieves multiple values in a single pipeline
func (c *redisCache) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}
	
	pipe := c.client.Pipeline()
	
	// Create commands for each key
	cmds := make(map[string]*redis.StringCmd, len(keys))
	for _, key := range keys {
		prefixedKey := c.prefixKey(key)
		cmds[key] = pipe.Get(ctx, prefixedKey)
	}
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("redis pipeline failed: %w", err)
	}
	
	// Collect results
	results := make(map[string]string, len(keys))
	for key, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			continue // Skip missing keys
		}
		if err != nil {
			continue // Skip errors for individual keys
		}
		results[key] = val
	}
	
	return results, nil
}

// IncrementCounter atomically increments a counter in Redis
// Useful for rate limiting or statistics
func (c *redisCache) IncrementCounter(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	prefixedKey := c.prefixKey(key)
	
	// Use INCR for atomic increment
	count, err := c.client.Incr(ctx, prefixedKey).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr failed: %w", err)
	}
	
	// Set expiration if this is the first increment
	if count == 1 && ttl > 0 {
		c.client.Expire(ctx, prefixedKey, ttl)
	}
	
	return count, nil
}