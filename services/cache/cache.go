package cache

import "time"

// CacheService represents a generic cache service
type CacheService interface {
	// Get retrieves a value from the cache
	Get(key string) ([]byte, error)
	
	// Set stores a value in the cache with an expiration time
	Set(key string, value []byte, expiration time.Duration) error
	
	// Delete removes a value from the cache
	Delete(key string) error
}