package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewPpomEnCrawler creates a PpomEn crawler
func NewPpomEnCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.PpomEnURL + "/zboard/zboard.php?id=ppomppu4",
		CacheKey:     "ppom_en_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.PpomEnURL + "/zboard/",
		Provider:     ProviderPpomEn,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:   "tr.baseList.bbs_new1",
			Title:      "div.baseList-cover a.baseList-title",
			Link:       "div.baseList-cover a.baseList-title",
			Thumbnail:  "a.baseList-thumb img",
			PostedAt:   "time.baseList-time",
			PriceRegex: `\$([\\d,.]+)`,
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "no=", 1)
		},
	}, cacheSvc)
}
