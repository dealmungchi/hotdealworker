package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"sjsage522/hotdealworker/pkg/errors"
)

// CrawlerConfig represents configuration for individual crawlers
type CrawlerConfig struct {
	Enabled bool
	URL     string
}

// Config represents the application configuration
type Config struct {
	// Redis configuration
	RedisAddr            string
	RedisDB              int
	RedisStream          string
	RedisStreamCount     int
	RedisStreamMaxLength int

	// Memcache configuration
	MemcacheAddr string

	// Crawler configuration
	CrawlInterval time.Duration

	// ChromeDB configuration
	ChromeDBAddr string
	UseChromeDB  bool

	// URLs for different crawlers
	FMKoreaURL      string
	DamoangURL      string
	ArcaURL         string
	QuasarURL       string
	CoolandjoyURL   string
	ClienURL        string
	PpomURL         string
	PpomEnURL       string
	RuliwebURL      string
	DealbadaURL     string
	MissycouponsURL string
	MalltailURL     string
	BbasakURL       string
	CityURL         string
	EomisaeURL      string
	ZodURL          string

	// Environment
	Environment string

	// Crawler configurations
	Crawlers map[string]CrawlerConfig
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.RedisAddr == "" {
		return errors.NewConfiguration("redis address is required", nil)
	}
	if c.MemcacheAddr == "" {
		return errors.NewConfiguration("memcache address is required", nil)
	}
	if c.CrawlInterval < 10*time.Second {
		return errors.NewConfiguration("crawl interval must be at least 10 seconds", nil)
	}
	if c.RedisStreamMaxLength <= 0 {
		return errors.NewConfiguration("redis stream max length must be positive", nil)
	}

	// Validate at least one crawler is configured
	enabledCount := 0
	for name, cfg := range c.Crawlers {
		if cfg.Enabled {
			if cfg.URL == "" {
				return errors.NewConfiguration(fmt.Sprintf("%s crawler URL is required when enabled", name), nil)
			}
			enabledCount++
		}
	}

	if enabledCount == 0 {
		return errors.NewConfiguration("at least one crawler must be enabled", nil)
	}

	return nil
}

// LoadConfig loads the configuration from environment variables with defaults
func LoadConfig() Config {
	// Load .env file if it exists

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	crawlInterval, _ := strconv.Atoi(getEnv("CRAWL_INTERVAL_SECONDS", "60"))
	redisStreamCount, _ := strconv.Atoi(getEnv("REDIS_STREAM_COUNT", "1"))
	redisStreamMaxLength, _ := strconv.Atoi(getEnv("REDIS_STREAM_MAX_LENGTH", "500"))
	environment := getEnv("HOTDEAL_ENVIRONMENT", "development")

	cfg := Config{
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:              redisDB,
		RedisStream:          getEnv("REDIS_STREAM", "streamHotdeals"),
		RedisStreamCount:     redisStreamCount,
		RedisStreamMaxLength: redisStreamMaxLength,
		MemcacheAddr:         getEnv("MEMCACHE_ADDR", "localhost:11211"),
		CrawlInterval:        time.Duration(crawlInterval) * time.Second,
		ChromeDBAddr:         getEnv("CHROME_DB_ADDR", "http://localhost:3000"),
		UseChromeDB:          getEnvBool("USE_CHROME_DB", false),
		FMKoreaURL:           getEnv("FMKOREA_URL", "https://www.fmkorea.com"),
		DamoangURL:           getEnv("DAMOANG_URL", "https://damoang.net"),
		ArcaURL:              getEnv("ARCA_URL", "https://arca.live"),
		QuasarURL:            getEnv("QUASAR_URL", "https://quasarzone.com"),
		CoolandjoyURL:        getEnv("COOLANDJOY_URL", "https://coolenjoy.net"),
		ClienURL:             getEnv("CLIEN_URL", "https://www.clien.net"),
		PpomURL:              getEnv("PPOM_URL", "https://www.ppomppu.co.kr"),
		PpomEnURL:            getEnv("PPOMEN_URL", "https://www.ppomppu.co.kr"),
		RuliwebURL:           getEnv("RULIWEB_URL", "https://bbs.ruliweb.com"),
		DealbadaURL:          getEnv("DEALBADA_URL", "https://www.dealbada.com"),
		MissycouponsURL:      getEnv("MISSYCOUPONS_URL", "https://www.missycoupons.com"),
		MalltailURL:          getEnv("MALLTAIL_URL", "https://post.malltail.com"),
		BbasakURL:            getEnv("BBASAK_URL", "https://bbasak.com"),
		CityURL:              getEnv("CITY_URL", "https://www.city.kr"),
		EomisaeURL:           getEnv("EOMISAE_URL", "https://eomisae.co.kr"),
		ZodURL:               getEnv("ZOD_URL", "https://zod.kr"),
		Environment:          environment,
		Crawlers:             make(map[string]CrawlerConfig),
	}

	// Initialize crawler configurations
	crawlerList := []struct {
		name string
		url  string
	}{
		{"fmkorea", cfg.FMKoreaURL},
		{"damoang", cfg.DamoangURL},
		{"arca", cfg.ArcaURL},
		{"quasar", cfg.QuasarURL},
		{"coolandjoy", cfg.CoolandjoyURL},
		{"clien", cfg.ClienURL},
		{"ppom", cfg.PpomURL},
		{"ppomen", cfg.PpomEnURL},
		{"ruliweb", cfg.RuliwebURL},
		{"dealbada", cfg.DealbadaURL},
		{"missycoupons", cfg.MissycouponsURL},
		{"malltail", cfg.MalltailURL},
		{"bbasak", cfg.BbasakURL},
		{"city", cfg.CityURL},
		{"eomisae", cfg.EomisaeURL},
		{"zod", cfg.ZodURL},
	}

	for _, c := range crawlerList {
		cfg.Crawlers[c.name] = CrawlerConfig{
			Enabled: getEnvBool(fmt.Sprintf("CRAWLER_%s_ENABLED", toUpper(c.name)), true),
			URL:     c.url,
		}
	}

	return cfg
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvBool retrieves an environment variable as bool or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

// toUpper converts string to uppercase for environment variable names
func toUpper(s string) string {
	return strings.ToUpper(s)
}
