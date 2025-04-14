package config

import (
	"os"
	"strconv"
	"time"
)

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

	// Environment
	Environment string
}

// LoadConfig loads the configuration from environment variables with defaults
func LoadConfig() Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	crawlInterval, _ := strconv.Atoi(getEnv("CRAWL_INTERVAL_SECONDS", "40"))
	redisStreamCount, _ := strconv.Atoi(getEnv("REDIS_STREAM_COUNT", "1"))
	redisStreamMaxLength, _ := strconv.Atoi(getEnv("REDIS_STREAM_MAX_LENGTH", "500"))

	return Config{
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:              redisDB,
		RedisStream:          getEnv("REDIS_STREAM", "streamHotdeals"),
		RedisStreamCount:     redisStreamCount,
		RedisStreamMaxLength: redisStreamMaxLength,
		MemcacheAddr:         getEnv("MEMCACHE_ADDR", "localhost:11211"),
		CrawlInterval:        time.Duration(crawlInterval) * time.Second,
		ChromeDBAddr:         getEnv("CHROME_DB_ADDR", "http://localhost:3000"),
		FMKoreaURL:           getEnv("FMKOREA_URL", "http://www.fmkorea.com"),
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
		Environment:          getEnv("HOTDEAL_ENVIRONMENT", "development"),
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
