package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewCoolandjoyCrawler creates a Coolandjoy crawler
func NewCoolandjoyCrawler(cfg *config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	// Create transformers for element cleanup
	elementTransformers := ElementTransformers{
		RemoveElements: []ElementRemoval{
			{Selector: "i", ApplyToPath: "postedAt"},
			{Selector: "span", ApplyToPath: "postedAt"},
		},
	}

	return NewConfigurableCrawler(CrawlerConfig{
		// Coolandjoy crawler configuration
		URL:       cfg.CoolandjoyURL,
		CacheKey:  "coolandjoy_rate_limited",
		BlockTime: 500,
		BaseURL:   "https://coolenjoy.net",
		Provider:  "Coolandjoy",
		Selectors: Selectors{
			DealList:   "ul.na-table li",
			Title:      "a.na-subject",
			Link:       "a.na-subject",
			Thumbnail:  "", // No thumbnail
			PostedAt:   "div.float-left.float-md-none.d-md-table-cell.nw-6.nw-md-auto.f-sm.font-weight-normal.py-md-2.pr-md-1",
			PriceRegex: `\(([0-9,]+원)\)$`,
		},
		ElementTransformers: elementTransformers,
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 5)
		},
	}, cacheSvc)
}