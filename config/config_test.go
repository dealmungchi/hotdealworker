package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Test with default values
	config := LoadConfig()
	assert.Equal(t, "localhost:6379", config.RedisAddr)
	assert.Equal(t, 0, config.RedisDB)
	assert.Equal(t, 1, config.RedisStreamCount)
	assert.Equal(t, "localhost:11211", config.MemcacheAddr)
	assert.Equal(t, 60*time.Second, config.CrawlInterval)

	// Test with environment variables
	os.Setenv("REDIS_ADDR", "redis.example.com:6379")
	os.Setenv("REDIS_DB", "1")
	os.Setenv("REDIS_STREAM_COUNT", "1")
	os.Setenv("MEMCACHE_ADDR", "memcache.example.com:11211")
	os.Setenv("CRAWL_INTERVAL_SECONDS", "30")
	os.Setenv("FMKOREA_URL", "https://example.com/fmkorea")

	config = LoadConfig()
	assert.Equal(t, "redis.example.com:6379", config.RedisAddr)
	assert.Equal(t, 1, config.RedisDB)
	assert.Equal(t, 1, config.RedisStreamCount)
	assert.Equal(t, "memcache.example.com:11211", config.MemcacheAddr)
	assert.Equal(t, 30*time.Second, config.CrawlInterval)
	assert.Equal(t, "https://example.com/fmkorea", config.FMKoreaURL)

	// Clean up
	os.Unsetenv("REDIS_ADDR")
	os.Unsetenv("REDIS_DB")
	os.Unsetenv("REDIS_STREAM_COUNT")
	os.Unsetenv("MEMCACHE_ADDR")
	os.Unsetenv("CRAWL_INTERVAL_SECONDS")
	os.Unsetenv("FMKOREA_URL")
}
