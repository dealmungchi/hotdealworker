package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewArcaCrawler creates an Arca crawler
func NewArcaCrawler(cfg config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	// Create transformers for element cleanup
	elementTransformers := ElementTransformers{
		RemoveElements: []ElementRemoval{
			{Selector: "span", ApplyToPath: "title"},
		},
	}

	return NewConfigurableCrawler(CrawlerConfig{
		// Arca crawler configuration
		URL:       cfg.ArcaURL + "/b/hotdeal",
		CacheKey:  "arca_rate_limited",
		BlockTime: 500,
		BaseURL:   cfg.ArcaURL,
		Provider:  "Arca",
		Selectors: Selectors{
			DealList:   "div.list-table.hybrid div.vrow.hybrid",
			Title:      "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
			Link:       "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
			Thumbnail:  "a.title.preview-image div.vrow-preview img",
			PostedAt:   "span.col-time time",
			PriceRegex: `\(([0-9,]+원)\)$`,
		},
		ElementTransformers: elementTransformers,
		IDExtractor: func(link string) (string, error) {
			baseLink := strings.Split(link, "?")[0]
			return helpers.GetSplitPart(baseLink, "/", 5)
		},
	}, cacheSvc)
}
