package crawler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dealmungchi/dealcrawler/services/cache"
	"github.com/dealmungchi/dealcrawler/services/proxy"
	"github.com/dealmungchi/dealcrawler/services/publisher"
	"github.com/rs/zerolog/log"
)

// Crawler represents a crawler with its configuration
type Crawler struct {
	CrawlerConfig
}

// CrawlerWithDeps represents a crawler with injected dependencies
type CrawlerWithDeps struct {
	Crawler
	cache     cache.CacheService
	publisher publisher.Publisher
	proxy     proxy.ProxyManager
}

// NewCrawler creates a new crawler
func NewCrawler(config CrawlerConfig) *Crawler {
	return &Crawler{config}
}

// WithDependencies creates a crawler with injected dependencies
func WithDependencies(c Crawler, cache cache.CacheService, publisher publisher.Publisher, proxy proxy.ProxyManager) *CrawlerWithDeps {
	return &CrawlerWithDeps{
		Crawler:   c,
		cache:     cache,
		publisher: publisher,
		proxy:     proxy,
	}
}

// Crawl executes the crawling process with injected dependencies
func (c *CrawlerWithDeps) Crawl() ([]HotDeal, error) {
	log.Info().Str("provider", c.Provider).Msg("Starting crawl")
	
	// Example: Check cache for recent crawl results
	cacheKey := fmt.Sprintf("crawl:%s:last_run", c.Provider)
	
	if cachedData, err := c.cache.Get(cacheKey); err == nil {
		log.Debug().Str("provider", c.Provider).Msg("Found cached crawl data")
		_ = cachedData // Handle cached data if needed
	}
	
	// Example: Try to get a proxy (optional)
	if fastestProxy, err := c.proxy.GetFastestProxy(); err == nil {
		log.Debug().
			Str("provider", c.Provider).
			Str("proxy", fmt.Sprintf("%s:%d", fastestProxy.Host, fastestProxy.Port)).
			Msg("Using proxy for crawling")
		_ = fastestProxy // Use proxy for HTTP requests
	} else {
		log.Debug().
			Str("provider", c.Provider).
			Err(err).
			Msg("No proxy available, using direct connection")
	}
	
	// TODO: Implement actual crawling logic here
	// This is where you would:
	// 1. Fetch the webpage content (with or without proxy)
	// 2. Parse HTML using the selectors
	// 3. Extract deal information
	// 4. Create HotDeal structs
	
	// For now, return empty results
	deals := []HotDeal{}
	
	// Cache the crawl timestamp
	timestamp := time.Now().Format(time.RFC3339)
	if err := c.cache.Set(cacheKey, []byte(timestamp), 1*time.Hour); err != nil {
		log.Warn().Err(err).Str("provider", c.Provider).Msg("Failed to cache crawl timestamp")
	}
	
	// Publish deals if any found
	if len(deals) > 0 {
		for _, deal := range deals {
			dealJSON, err := json.Marshal(deal)
			if err != nil {
				log.Error().Err(err).Str("deal_id", deal.Id).Msg("Failed to marshal deal")
				continue
			}
			
			if err := c.publisher.Publish(deal.Id, dealJSON); err != nil {
				log.Error().Err(err).Str("deal_id", deal.Id).Msg("Failed to publish deal")
			}
		}
	}
	
	log.Info().Str("provider", c.Provider).Int("deals_count", len(deals)).Msg("Crawl completed")
	return deals, nil
}
