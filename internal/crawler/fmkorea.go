package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewFMKoreaCrawler creates an FMKorea crawler
func NewFMKoreaCrawler(cfg config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	// Create transformers for element cleanup
	elementTransformers := ElementTransformers{
		RemoveElements: []ElementRemoval{
			{Selector: "span", ApplyToPath: "title"},
		},
	}

	return NewConfigurableCrawler(CrawlerConfig{
		// FMKorea crawler configuration
		URL:       cfg.FMKoreaURL + "/hotdeal",
		CacheKey:  "fmkorea_rate_limited",
		BlockTime: 300,
		BaseURL:   cfg.FMKoreaURL,
		Provider:  "FMKorea",
		Selectors: Selectors{
			DealList:   "ul li.li",
			Title:      "h3.title a",
			Link:       "h3.title a",
			Thumbnail:  "a img.thumb",
			PostedAt:   "div span.regdate",
			PriceRegex: `\(([0-9,]+원)\)$`,
		},
		ElementTransformers: elementTransformers,
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 3)
		},
	}, cacheSvc)
}
