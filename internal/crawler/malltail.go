package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

func NewMalltail(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.MalltailURL + "/hotdeals/index",
		CacheKey:     "malltail_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.MalltailURL,
		Provider:     "Malltail",
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:    "div#container div.hotdeal-wrap.event_area table.list tbody tr",
			Title:       "td.title a",
			Link:        "td.title a",
			ClassFilter: "notice",
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 5)
		},
	}, cacheSvc)
}
