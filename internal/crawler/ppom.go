package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewPpomCrawler creates a Ppom crawler
func NewPpomCrawler(cfg config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	return NewConfigurableCrawler(CrawlerConfig{
		// Ppom crawler configuration
		URL:       cfg.PpomURL + "/zboard/zboard.php?id=ppomppu",
		CacheKey:  "ppom_rate_limited",
		BlockTime: 500,
		BaseURL:   cfg.PpomURL,
		Provider:  "Ppom",
		Selectors: Selectors{
			DealList:   "tr.baseList.bbs_new1",
			Title:      "div.baseList-cover a.baseList-title",
			Link:       "div.baseList-cover a.baseList-title",
			Thumbnail:  "a.baseList-thumb img",
			PostedAt:   "time.baseList-time",
			PriceRegex: `\(([0-9,]+원)\)$`,
		},
		CustomHandlers: CustomHandlers{
			ElementHandlers: map[string]CustomElementHandlerFunc{},
		},
		ElementTransformers: ElementTransformers{
			RemoveElements: []ElementRemoval{},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "no=", 1)
		},
	}, cacheSvc)
}
