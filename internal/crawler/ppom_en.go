package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewPpomEnCrawler creates a PpomEn crawler
func NewPpomEnCrawler(cfg *config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	return NewConfigurableCrawler(CrawlerConfig{
		// PpomEn crawler configuration
		URL:       cfg.PpomEnURL,
		CacheKey:  "ppom_en_rate_limited",
		BlockTime: 500,
		BaseURL:   "https://www.ppomppu.co.kr",
		Provider:  "PpomEn",
		Selectors: Selectors{
			DealList:   "tr.baseList.bbs_new1",
			Title:      "div.baseList-cover a.baseList-title",
			Link:       "div.baseList-cover a.baseList-title",
			Thumbnail:  "a.baseList-thumb img",
			PostedAt:   "time.baseList-time",
			PriceRegex: `\$([\d,.]+)`,
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