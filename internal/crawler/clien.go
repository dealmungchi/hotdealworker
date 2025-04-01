package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewClienCrawler creates a Clien crawler
func NewClienCrawler(cfg *config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	return NewConfigurableCrawler(CrawlerConfig{
		// Clien crawler configuration
		URL:       cfg.ClienURL,
		CacheKey:  "clien_rate_limited",
		BlockTime: 500,
		BaseURL:   "https://www.clien.net",
		Provider:  "Clien",
		Selectors: Selectors{
			DealList:    "div.list_item.symph_row.jirum",
			Title:       "span.list_subject",
			Link:        "a[data-role='list-title-text']",
			Thumbnail:   "div.list_img a.list_thumbnail img",
			PostedAt:    "div.list_time span.time.popover span.timestamp",
			PriceRegex:  `\(([0-9,]+원)\)$`,
			ClassFilter: "blocked",
		},
		CustomHandlers: CustomHandlers{
			ElementHandlers: map[string]CustomElementHandlerFunc{},
		},
		ElementTransformers: ElementTransformers{
			RemoveElements: []ElementRemoval{},
		},
		IDExtractor: func(link string) (string, error) {
			baseLink := strings.Split(link, "?")[0]
			return helpers.GetSplitPart(baseLink, "/", 6)
		},
	}, cacheSvc)
}