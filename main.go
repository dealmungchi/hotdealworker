package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dealmungchi/dealcrawler/config"
	"github.com/dealmungchi/dealcrawler/crawler"
	"github.com/dealmungchi/dealcrawler/internal"
	"github.com/dealmungchi/dealcrawler/logger"
	"github.com/dealmungchi/dealcrawler/services/cache"
	"github.com/dealmungchi/dealcrawler/services/proxy"
	"github.com/dealmungchi/dealcrawler/services/publisher"
	"github.com/dealmungchi/dealcrawler/services/worker"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// App represents the main application
type App struct {
	deps     internal.Dependencies
	config   *config.Config
	crawlers []crawler.Crawler
	worker   *worker.Worker
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) (*App, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	// Initialize services
	deps, err := initializeDependencies(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %v", err)
	}

	// Create crawlers
	crawlers := crawler.CreateCrawlers()
	if len(crawlers) == 0 {
		log.Warn().Msg("No crawlers were created")
	}

	// Create worker with dependencies
	w := worker.NewWorker(deps, crawlers, cfg.CrawlIntervalSeconds)

	return &App{
		deps:     deps,
		config:   cfg,
		crawlers: crawlers,
		worker:   w,
	}, nil
}

// Start starts the application
func (app *App) Start(ctx context.Context) error {
	log.Info().
		Int("crawler_count", len(app.crawlers)).
		Int("interval_seconds", app.config.CrawlIntervalSeconds).
		Msg("Starting application")

	return app.worker.Start(ctx)
}

// Cleanup cleans up application resources
func (app *App) Cleanup() {
	log.Info().Msg("Cleaning up application resources")

	if app.deps.Publisher != nil {
		app.deps.Publisher.Close()
	}
}

// initializeDependencies creates and initializes all service dependencies
func initializeDependencies(cfg *config.Config) (internal.Dependencies, error) {
	var deps internal.Dependencies

	// Initialize cache service
	log.Info().Str("addr", cfg.MemcacheAddr).Msg("Initializing cache service")
	cacheService := cache.NewMemcacheService(cfg.MemcacheAddr)
	if cacheService == nil {
		return deps, fmt.Errorf("failed to create cache service")
	}
	deps.Cache = cacheService

	// Initialize publisher service
	log.Info().
		Str("addr", cfg.RedisAddr).
		Int("db", cfg.RedisDB).
		Str("stream", cfg.RedisStream).
		Msg("Initializing Redis publisher")

	redisPublisher := publisher.NewRedisPublisher(
		context.Background(),
		cfg.RedisAddr,
		cfg.RedisDB,
		cfg.RedisStream,
		cfg.RedisStreamCount,
		cfg.RedisStreamMaxLength,
	)
	if redisPublisher == nil {
		return deps, fmt.Errorf("failed to create redis publisher")
	}
	deps.Publisher = redisPublisher

	// Initialize proxy manager
	log.Info().Msg("Initializing proxy manager")
	proxyManager := proxy.NewProxyManager()
	if err := proxyManager.UpdateProxies(); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize proxy list - continuing without proxies")
	}
	deps.Proxy = proxyManager

	return deps, nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Info().Err(err).Msg("No .env file found, using environment variables and defaults")
	}

	// Initialize logger
	logger.Init()
	log.Info().Msg("Starting deal crawler application")

	// Load configuration
	cfg := config.Load()

	// Create application
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create application")
	}
	defer app.Cleanup()

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start application in goroutine
	appDone := make(chan error, 1)
	go func() {
		appDone <- app.Start(ctx)
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or app error
	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		gracefulShutdown(cancel)
	case err := <-appDone:
		if err != nil {
			log.Error().Err(err).Msg("Application exited with error")
		} else {
			log.Info().Msg("Application exited normally")
		}
		gracefulShutdown(cancel)
	}

	log.Info().Msg("Application shutdown completed")
}

// gracefulShutdown handles graceful shutdown with timeout
func gracefulShutdown(cancel context.CancelFunc) {
	timeout := 30 * time.Second
	log.Info().Dur("timeout", timeout).Msg("Starting graceful shutdown")

	// Cancel context to signal all goroutines to stop
	cancel()

	// Wait for shutdown or timeout
	done := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		close(done)
	}()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), timeout)
	defer shutdownCancel()

	select {
	case <-done:
		log.Info().Msg("Graceful shutdown completed")
	case <-shutdownCtx.Done():
		log.Warn().Msg("Graceful shutdown timed out")
	}
}
