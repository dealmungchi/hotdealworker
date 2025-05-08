package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewPpomCrawler creates a Ppomppu crawler
func NewPpomCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.PpomURL + "/zboard/zboard.php?id=ppomppu",
		CacheKey:     "ppom_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.PpomURL + "/zboard/",
		Provider:     ProviderPpom,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:   "tr.baseList.bbs_new1",
			Title:      "div.baseList-cover a.baseList-title",
			Link:       "div.baseList-cover a.baseList-title",
			Thumbnail:  "a.baseList-thumb img",
			PostedAt:   "time.baseList-time",
			Category:   "div.baseList-box small.baseList-small",
			PriceRegex: `\(([0-9,]+원)\)$`,
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "no=", 1)
		},
	}, cacheSvc)
}
