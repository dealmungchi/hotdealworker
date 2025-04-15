package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

func NewZod(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.ZodURL + "/deal",
		CacheKey:     "zod_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.ZodURL,
		Provider:     "Zod",
		UseChrome:    false,
		ChromeDBAddr: cfg.ZodURL,
		Selectors: Selectors{
			DealList:    "ul.app-board-template-list.zod-board-list--deal li",
			Title:       "a.tw-flex-1 div.tw-flex-1 div.app-list-title.tw-flex-wrap span.tw-mr-1.app-list-title-item",
			Link:        "a.tw-flex-1",
			Thumbnail:   "a.tw-flex-1 div.app-thumbnail img",
			ClassFilter: "notice zod-board-list-deal-ended",
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 4)
		},
	}, cacheSvc)
}
