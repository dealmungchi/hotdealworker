package main

import (
	"context"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/internal/crawler"
	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/services/cache"
	"sjsage522/hotdealworker/services/publisher"
	"sjsage522/hotdealworker/services/worker"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	// Load configuration
	cfg := config.LoadConfig()

	// Set up context
	ctx := context.Background()

	// Initialize logger
	logger.Init()

	// 로그 출력 테스트
	logger.Info("Starting application in %s environment", cfg.Environment)

	// 디버그 로그 예시
	logger.Debug("This is a debug message that will only appear if log level is debug or lower")

	// Initialize cache service
	cacheService := cache.NewMemcacheService(cfg.MemcacheAddr)

	// Initialize publisher
	redisPublisher := publisher.NewRedisPublisher(
		ctx,
		cfg.RedisAddr,
		cfg.RedisDB,
		cfg.RedisStream,
		cfg.RedisStreamCount,
		cfg.RedisStreamMaxLength,
	)
	defer redisPublisher.Close()

	// Create crawlers
	crawlers := crawler.CreateCrawlers(cfg, cacheService)

	// Create and start worker
	w := worker.NewWorker(
		ctx,
		crawlers,
		redisPublisher,
		cfg.CrawlInterval,
	)

	logger.Info("Starting hot deal worker in %s environment", cfg.Environment)
	w.Start()
}
