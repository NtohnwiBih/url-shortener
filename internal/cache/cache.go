package cache

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations
// This abstraction allows swapping cache implementations (Redis, Memcached, in-memory)
type Cache interface {
	// Set stores a key-value pair with expiration
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	
	// Get retrieves a value by key
	Get(ctx context.Context, key string) (string, error)
	
	// Delete removes a key from cache
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)
	
	// Close closes the cache connection
	Close() error
}