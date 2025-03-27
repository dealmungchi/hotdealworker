package crawler

import (
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// BaseCrawler provides common functionality for all crawlers
type BaseCrawler struct {
	URL       string
	CacheKey  string
	CacheSvc  cache.CacheService
	BlockTime time.Duration
}

// fetchWithCache fetches a URL with caching and rate limiting
func (c *BaseCrawler) fetchWithCache() (io.Reader, error) {
	// Check if the crawler is rate limited
	if c.CacheSvc != nil && c.CacheKey != "" {
		_, err := c.CacheSvc.Get(c.CacheKey)
		if err == nil {
			return nil, fmt.Errorf("%s: %d초 동안 더 이상 요청을 보내지 않음", c.CacheKey, c.BlockTime/time.Second)
		}
	}

	// Fetch the page
	utf8Body, err := helpers.FetchWithRandomHeaders(c.URL)
	if err != nil {
		if c.CacheSvc != nil && c.CacheKey != "" && err.Error() != "" {
			if fmt.Sprintf("%v", err)[:12] == "rate limited" {
				// Set rate limiting cache
				c.CacheSvc.Set(c.CacheKey, []byte(fmt.Sprintf("%d", c.BlockTime/time.Second)), c.BlockTime)
			}
		}
		return nil, err
	}

	return utf8Body, nil
}

// createDocument creates a goquery document from a reader
func (c *BaseCrawler) createDocument(reader io.Reader) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 오류: %v", err)
	}
	return doc, nil
}

// processDeals processes deals in parallel using goroutines
func (c *BaseCrawler) processDeals(selections *goquery.Selection, processor func(*goquery.Selection) *HotDeal) []HotDeal {
	dealChan := make(chan *HotDeal, selections.Length())
	var wg sync.WaitGroup

	selections.Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		go func(s *goquery.Selection) {
			defer wg.Done()
			
			// Process the deal in the goroutine
			deal := processor(s)
			if deal != nil {
				dealChan <- deal
			}
		}(s)
	})

	wg.Wait()
	close(dealChan)

	// Collect the processed deals
	var deals []HotDeal
	for deal := range dealChan {
		if deal != nil {
			deals = append(deals, *deal)
		}
	}

	return deals
}

// GetName returns the crawler's type name for logging
func (c *BaseCrawler) GetName() string {
	// This will be overridden by concrete implementations
	// But fallback to reflect-based name if not
	return reflect.TypeOf(c).Elem().Name()
}