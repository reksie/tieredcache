package interfaces

import "time"

// CacheStore defines the interface for a cache store.
type CacheStore interface {
	// Name returns a name for metrics or identification purposes.
	Name() string

	// Set stores a value in the cache with an optional TTL (time to live) in milliseconds.
	Set(key string, value any, ttl time.Duration) error

	// Get retrieves a value from the cache by its key.
	Get(key string) (any, error)

	// Delete removes a value from the cache by its key.
	Delete(key string) error

	// Clear removes all values from the cache (optional).
	Clear() error

	// Close releases any resources or connections when the cache is no longer in use (optional).
	Close() error
}
