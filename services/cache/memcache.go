package cache

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// MemcacheService implements CacheService using memcache
type MemcacheService struct {
	client *memcache.Client
}

// NewMemcacheService creates a new memcache service
func NewMemcacheService(serverAddr string) *MemcacheService {
	return &MemcacheService{
		client: memcache.New(serverAddr),
	}
}

// Get retrieves a value from memcache
func (m *MemcacheService) Get(key string) ([]byte, error) {
	item, err := m.client.Get(key)
	if err != nil {
		return nil, err
	}
	return item.Value, nil
}

// Set stores a value in memcache with an expiration time
func (m *MemcacheService) Set(key string, value []byte, expiration time.Duration) error {
	return m.client.Set(&memcache.Item{
		Key:        key,
		Value:      value,
		Expiration: int32(expiration.Seconds()),
	})
}

// Delete removes a value from memcache
func (m *MemcacheService) Delete(key string) error {
	return m.client.Delete(key)
}