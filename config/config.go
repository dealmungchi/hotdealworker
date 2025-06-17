package config

import (
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
)

// Config holds application configuration
type Config struct {
	// Cache configuration
	MemcacheAddr string

	// Redis configuration
	RedisAddr            string
	RedisDB              int
	RedisStream          string
	RedisStreamCount     int
	RedisStreamMaxLength int

	// Crawler configuration
	CrawlIntervalSeconds int

	// Site URLs
	ArcaURL string

	// Logging
	LogLevel string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	config := &Config{
		// Cache defaults
		MemcacheAddr: getEnvWithDefault("MEMCACHE_ADDR", "localhost:11211"),

		// Redis defaults
		RedisAddr:            getEnvWithDefault("REDIS_ADDR", "localhost:6379"),
		RedisDB:              getEnvIntWithDefault("REDIS_DB", 0),
		RedisStream:          getEnvWithDefault("REDIS_STREAM", "streamHotdeals"),
		RedisStreamCount:     getEnvIntWithDefault("REDIS_STREAM_COUNT", 1),
		RedisStreamMaxLength: getEnvIntWithDefault("REDIS_STREAM_MAX_LENGTH", 500),

		// Crawler defaults
		CrawlIntervalSeconds: getEnvIntWithDefault("CRAWL_INTERVAL_SECONDS", 60),

		// Site URLs
		ArcaURL: getEnvWithDefault("ARCA_URL", "https://arca.live"),

		// Logging
		LogLevel: getEnvWithDefault("LOG_LEVEL", "INFO"),
	}

	logConfigValues(config)
	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.CrawlIntervalSeconds <= 0 {
		c.CrawlIntervalSeconds = 60
		log.Warn().Int("value", c.CrawlIntervalSeconds).Msg("Invalid crawl interval, using default")
	}

	if c.RedisStreamCount <= 0 {
		c.RedisStreamCount = 1
		log.Warn().Int("value", c.RedisStreamCount).Msg("Invalid redis stream count, using default")
	}

	if c.RedisStreamMaxLength <= 0 {
		c.RedisStreamMaxLength = 500
		log.Warn().Int("value", c.RedisStreamMaxLength).Msg("Invalid redis stream max length, using default")
	}

	return nil
}

// getEnvWithDefault returns environment variable value or default
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntWithDefault returns environment variable as int or default
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Warn().Str("key", key).Str("value", value).Msg("Invalid integer in environment variable, using default")
	}
	return defaultValue
}

// logConfigValues logs the loaded configuration values (without sensitive data)
func logConfigValues(config *Config) {
	log.Info().
		Str("memcache_addr", config.MemcacheAddr).
		Str("redis_addr", config.RedisAddr).
		Int("redis_db", config.RedisDB).
		Str("redis_stream", config.RedisStream).
		Int("crawl_interval_seconds", config.CrawlIntervalSeconds).
		Str("arca_url", config.ArcaURL).
		Str("log_level", config.LogLevel).
		Msg("Configuration loaded")
}
