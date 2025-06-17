package worker

import (
	"context"
	"time"

	"github.com/dealmungchi/dealcrawler/crawler"
	"github.com/dealmungchi/dealcrawler/internal"
	"github.com/rs/zerolog/log"
)

// CrawlResults holds the results of a crawl cycle
type CrawlResults struct {
	TotalDeals         int
	SuccessfulCrawlers int
	FailedCrawlers     int
}

// Worker manages the crawling process
type Worker struct {
	deps            internal.Dependencies
	crawlers        []crawler.Crawler
	intervalSeconds int
}

// NewWorker creates a new worker with the given dependencies
func NewWorker(deps internal.Dependencies, crawlers []crawler.Crawler, intervalSeconds int) *Worker {
	if intervalSeconds == 0 {
		intervalSeconds = 60 // Default to 60 seconds
	}

	return &Worker{
		deps:            deps,
		crawlers:        crawlers,
		intervalSeconds: intervalSeconds,
	}
}

// Start starts the worker and runs crawling cycles
func (w *Worker) Start(ctx context.Context) error {
	interval := time.Duration(w.intervalSeconds) * time.Second

	log.Info().
		Int("crawler_count", len(w.crawlers)).
		Dur("interval", interval).
		Msg("Worker started")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Worker shutting down")
			return ctx.Err()
		case <-ticker.C:
			w.runCrawlCycle()
		}
	}
}

// runCrawlCycle executes one crawl cycle
func (w *Worker) runCrawlCycle() {
	start := time.Now()

	log.Debug().Msg("Starting crawl cycle")

	results := w.runCrawlers()

	elapsed := time.Since(start)

	log.Info().
		Dur("duration", elapsed).
		Int("total_deals", results.TotalDeals).
		Int("successful_crawlers", results.SuccessfulCrawlers).
		Int("failed_crawlers", results.FailedCrawlers).
		Msg("Crawl cycle completed")
}

// runCrawlers executes all crawlers and returns results
func (w *Worker) runCrawlers() CrawlResults {
	log.Debug().Int("crawler_count", len(w.crawlers)).Msg("Running crawlers")

	results := CrawlResults{}

	for i, c := range w.crawlers {
		log.Debug().
			Int("crawler_index", i).
			Str("provider", c.Provider).
			Msg("Executing crawler")

		// Create crawler with dependencies
		crawlerWithDeps := crawler.WithDependencies(c, w.deps.Cache, w.deps.Publisher, w.deps.Proxy)

		deals, err := crawlerWithDeps.Crawl()
		if err != nil {
			log.Error().
				Err(err).
				Str("provider", c.Provider).
				Msg("Crawler failed")
			results.FailedCrawlers++
			continue
		}

		results.SuccessfulCrawlers++
		results.TotalDeals += len(deals)

		log.Info().
			Str("provider", c.Provider).
			Int("deals_found", len(deals)).
			Msg("Crawler completed successfully")
	}

	return results
}
