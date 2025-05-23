package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"sjsage522/hotdealworker/internal/crawler"
	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/pkg/errors"
	"sjsage522/hotdealworker/services/publisher"
)

// Worker handles the crawling and publishing process
type Worker struct {
	ctx           context.Context
	crawlers      []crawler.Crawler
	publisher     publisher.Publisher
	crawlInterval time.Duration
	logger        *logger.Logger
}

// NewWorker creates a new worker
func NewWorker(
	ctx context.Context,
	crawlers []crawler.Crawler,
	pub publisher.Publisher,
	crawlInterval time.Duration,
) *Worker {
	return &Worker{
		ctx:           ctx,
		crawlers:      crawlers,
		publisher:     pub,
		crawlInterval: crawlInterval,
		logger:        logger.ForWorker(),
	}
}

// Start starts the worker process
func (w *Worker) Start() error {
	w.logger.Info().
		Int("crawler_count", len(w.crawlers)).
		Dur("interval", w.crawlInterval).
		Msg("Worker started")

	ticker := time.NewTicker(w.crawlInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.runCrawlCycle()

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info().Msg("Worker shutting down")
			return nil
		case <-ticker.C:
			w.runCrawlCycle()
		}
	}
}

// runCrawlCycle runs a single crawl cycle
func (w *Worker) runCrawlCycle() {
	start := time.Now()

	w.logger.Debug().Msg("Starting crawl cycle")

	results := w.runCrawlers()

	elapsed := time.Since(start)

	// Log cycle summary
	w.logger.Info().
		Dur("duration", elapsed).
		Int("total_deals", results.TotalDeals).
		Int("successful_crawlers", results.SuccessfulCrawlers).
		Int("failed_crawlers", results.FailedCrawlers).
		Msg("Crawl cycle completed")
}

// CrawlResults holds the results of a crawl cycle
type CrawlResults struct {
	TotalDeals         int
	SuccessfulCrawlers int
	FailedCrawlers     int
}

// runCrawlers runs all the crawlers in parallel and then trims the streams
func (w *Worker) runCrawlers() CrawlResults {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results CrawlResults
	)

	// Create a channel for crawler results
	resultChan := make(chan crawlerResult, len(w.crawlers))

	// Run crawlers in parallel
	for _, c := range w.crawlers {
		wg.Add(1)
		go func(crawler crawler.Crawler) {
			defer wg.Done()

			result := w.crawlAndPublish(crawler)
			resultChan <- result
		}(c)
	}

	// Wait for all crawlers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		mu.Lock()
		if result.Success {
			results.SuccessfulCrawlers++
			results.TotalDeals += result.DealCount
		} else {
			results.FailedCrawlers++
		}
		mu.Unlock()
	}

	// Trim all streams after crawling
	if err := w.publisher.TrimStreams(); err != nil {
		w.logger.Error().
			Err(err).
			Msg("Failed to trim streams")
	}

	return results
}

// crawlerResult holds the result of a single crawler run
type crawlerResult struct {
	CrawlerName string
	Success     bool
	DealCount   int
	Error       error
}

// crawlAndPublish crawls deals from a crawler and publishes them
func (w *Worker) crawlAndPublish(c crawler.Crawler) crawlerResult {
	crawlerName := c.GetName()
	provider := c.GetProvider()

	log := w.logger.WithField("crawler", crawlerName)
	result := crawlerResult{CrawlerName: crawlerName}

	// Check context before starting
	select {
	case <-w.ctx.Done():
		result.Error = fmt.Errorf("context cancelled")
		return result
	default:
	}

	// Fetch deals
	log.Debug().Msg("Fetching deals")
	deals, err := c.FetchDeals()
	if err != nil {
		// Check if it's a custom error
		var crawlerErr *errors.CrawlerError
		if cErr, ok := err.(*errors.CrawlerError); ok {
			crawlerErr = cErr
		} else {
			crawlerErr = errors.New(errors.ErrorTypeNetwork, provider, "Failed to fetch deals", err)
		}

		log.Error().
			Err(crawlerErr).
			Bool("retryable", crawlerErr.IsRetryable()).
			Msg("Failed to fetch deals")

		result.Error = crawlerErr
		return result
	}

	// Publish deals
	publishedCount := 0
	for _, deal := range deals {
		// Check context for each deal
		select {
		case <-w.ctx.Done():
			result.Error = fmt.Errorf("context cancelled during publishing")
			result.Success = false
			result.DealCount = publishedCount
			return result
		default:
		}

		dealData, err := json.Marshal(deal)
		if err != nil {
			log.Error().
				Err(err).
				Str("deal_id", deal.Id).
				Msg("Failed to marshal deal")
			continue
		}

		if err := w.publisher.Publish(provider, dealData); err != nil {
			log.Error().
				Err(err).
				Str("deal_id", deal.Id).
				Msg("Failed to publish deal")
			continue
		}

		publishedCount++
	}

	// Log summary
	if publishedCount > 0 {
		log.Info().
			Int("fetched", len(deals)).
			Int("published", publishedCount).
			Msg("Deals processed")
	} else if len(deals) == 0 {
		log.Debug().Msg("No deals found")
	}

	// Log deal details if debug enabled
	if logger.IsDebugEnabled() && len(deals) > 0 {
		w.logDeals(deals, log)
	}

	result.Success = true
	result.DealCount = publishedCount
	return result
}

// logDeals logs deal details for debugging
func (w *Worker) logDeals(deals []crawler.HotDeal, log *logger.Logger) {
	for i, deal := range deals {
		// Create a copy without thumbnail for logging
		loggableDeal := map[string]interface{}{
			"id":             deal.Id,
			"title":          deal.Title,
			"link":           deal.Link,
			"price":          deal.Price,
			"thumbnail_link": deal.ThumbnailLink,
			"posted_at":      deal.PostedAt,
			"category":       deal.Category,
			"provider":       deal.Provider,
		}

		if deal.Thumbnail != "" {
			loggableDeal["thumbnail"] = "OK"
		}

		log.Debug().
			Int("index", i+1).
			Interface("deal", loggableDeal).
			Msg("Deal details")
	}
}
