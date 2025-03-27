package main

import (
	"context"
	"log"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/internal/crawler"
	"sjsage522/hotdealworker/services/cache"
	"sjsage522/hotdealworker/services/publisher"
	"sjsage522/hotdealworker/services/worker"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Set up context
	ctx := context.Background()

	// Initialize logger
	logger := helpers.NewLogger("./error")

	// Initialize cache service
	cacheService := cache.NewMemcacheService(cfg.MemcacheAddr)

	// Initialize publisher
	redisPublisher := publisher.NewRedisPublisher(ctx, cfg.RedisAddr, cfg.RedisDB)
	defer redisPublisher.Close()

	// Create crawlers
	crawlers := crawler.CreateCrawlers(cfg, cacheService)

	// Create and start worker
	w := worker.NewWorker(
		ctx,
		crawlers,
		redisPublisher,
		logger,
		cfg.CrawlInterval,
		cfg.RedisChannel,
	)

	log.Printf("Starting hot deal worker... in %s environment", cfg.Environment)
	w.Start()
}
