package config

import (
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	// Redis configuration
	RedisAddr    string
	RedisDB      int
	RedisChannel string

	// Memcache configuration
	MemcacheAddr string

	// Crawler configuration
	CrawlInterval time.Duration

	// URLs for different crawlers
	FMKoreaURL    string
	DamoangURL    string
	ArcaURL       string
	QuasarURL     string
	CoolandjoyURL string
	ClienURL      string
	PpomURL       string
	PpomEnURL     string
	RuliwebURL    string

	// Environment
	Environment string
}

// LoadConfig loads the configuration from environment variables with defaults
func LoadConfig() *Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	crawlInterval, _ := strconv.Atoi(getEnv("CRAWL_INTERVAL_SECONDS", "60"))

	return &Config{
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:       redisDB,
		RedisChannel:  getEnv("REDIS_CHANNEL", "hotdeals"),
		MemcacheAddr:  getEnv("MEMCACHE_ADDR", "localhost:11211"),
		CrawlInterval: time.Duration(crawlInterval) * time.Second,
		FMKoreaURL:    getEnv("FMKOREA_URL", "http://www.fmkorea.com/hotdeal"),
		DamoangURL:    getEnv("DAMOANG_URL", "https://damoang.net/economy"),
		ArcaURL:       getEnv("ARCA_URL", "https://arca.live/b/hotdeal"),
		QuasarURL:     getEnv("QUASAR_URL", "https://quasarzone.com/bbs/qb_saleinfo"),
		CoolandjoyURL: getEnv("COOLANDJOY_URL", "https://coolenjoy.net/bbs/jirum"),
		ClienURL:      getEnv("CLIEN_URL", "https://www.clien.net/service/board/jirum"),
		PpomURL:       getEnv("PPOM_URL", "https://www.ppomppu.co.kr/zboard/zboard.php?id=ppomppu"),
		PpomEnURL:     getEnv("PPOMEN_URL", "https://www.ppomppu.co.kr/zboard/zboard.php?id=ppomppu4"),
		RuliwebURL:    getEnv("RULIWEB_URL", "https://bbs.ruliweb.com/market/board/1020?view=thumbnail&page=1"),
		Environment:   getEnv("HOTDEAL_ENVIRONMENT", "development"),
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
