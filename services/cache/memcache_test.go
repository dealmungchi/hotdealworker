package cache

import (
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/stretchr/testify/assert"
)

// This test requires a running memcached instance
// If memcached is not available, the test will be skipped
func TestMemcacheService(t *testing.T) {
	// Create a memcache client
	mc := NewMemcacheService("localhost:11211")

	// Test if memcached is available
	_, err := mc.client.Get("test")
	if err != nil && err != memcache.ErrCacheMiss {
		t.Skip("Memcached is not available, skipping test")
	}

	// Set a value
	err = mc.Set("test_key", []byte("test_value"), 1*time.Second)
	assert.NoError(t, err)

	// Get the value
	value, err := mc.Get("test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", string(value))

	// Delete the value
	err = mc.Delete("test_key")
	assert.NoError(t, err)

	// Try to get the deleted value
	_, err = mc.Get("test_key")
	assert.Error(t, err)
}