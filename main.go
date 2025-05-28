package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/internal/crawler"
	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/services/cache"
	"sjsage522/hotdealworker/services/publisher"
	"sjsage522/hotdealworker/services/worker"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Initialize logger first
	logger.Init()
	log := logger.Default

	// Load and validate configuration
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	if err := crawler.InitializeProxyManager(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize proxy manager")
	}
	stats := crawler.GetProxyStats()
	log.Info().Interface("proxy_stats", stats).Msg("Proxy stats")

	log.Info().
		Str("environment", cfg.Environment).
		Dur("crawl_interval", cfg.CrawlInterval).
		Msg("Starting application")

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize services
	services, err := initializeServices(ctx, &cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize services")
	}
	defer services.Cleanup()

	// Create crawlers
	crawlers := crawler.CreateCrawlers(&cfg, services.Cache)
	if len(crawlers) == 0 {
		log.Fatal().Msg("No crawlers were created")
	}

	log.Info().
		Int("crawler_count", len(crawlers)).
		Msg("Created crawlers")

	// Create and start worker
	w := worker.NewWorker(
		ctx,
		crawlers,
		services.Publisher,
		cfg.CrawlInterval,
	)

	// Start worker in a goroutine
	workerDone := make(chan error, 1)
	go func() {
		log.Info().Msg("Starting hot deal worker")
		err := w.Start()
		workerDone <- err
	}()

	// Wait for shutdown signal or worker error
	select {
	case sig := <-sigChan:
		log.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal")
		cancel()
	case err := <-workerDone:
		if err != nil {
			log.Error().Err(err).Msg("Worker exited with error")
		} else {
			log.Info().Msg("Worker exited normally")
		}
	}

	// Graceful shutdown
	log.Info().Msg("Shutting down gracefully...")
}

// Services holds all the initialized services
type Services struct {
	Cache     cache.CacheService
	Publisher publisher.Publisher
}

// Cleanup cleans up all services
func (s *Services) Cleanup() {
	if s.Publisher != nil {
		s.Publisher.Close()
	}
}

// initializeServices initializes all required services
func initializeServices(ctx context.Context, cfg *config.Config) (*Services, error) {
	services := &Services{}

	// Initialize cache service
	cacheService := cache.NewMemcacheService(cfg.MemcacheAddr)
	if cacheService == nil {
		return nil, fmt.Errorf("failed to create cache service")
	}
	services.Cache = cacheService

	logger.Info("Connected to Memcache at %s", cfg.MemcacheAddr)

	// Initialize publisher
	redisPublisher := publisher.NewRedisPublisher(
		ctx,
		cfg.RedisAddr,
		cfg.RedisDB,
		cfg.RedisStream,
		cfg.RedisStreamCount,
		cfg.RedisStreamMaxLength,
	)
	if redisPublisher == nil {
		return nil, fmt.Errorf("failed to create redis publisher")
	}
	services.Publisher = redisPublisher

	logger.Info("Connected to Redis at %s (DB: %d, Stream: %s)",
		cfg.RedisAddr, cfg.RedisDB, cfg.RedisStream)

	return services, nil
}
