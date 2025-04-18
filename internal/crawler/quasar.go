package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewQuasarCrawler creates a Quasar crawler
func NewQuasarCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.QuasarURL + "/bbs/qb_saleinfo",
		CacheKey:     "quasar_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.QuasarURL,
		Provider:     "Quasar",
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:   "div.market-type-list.market-info-type-list.relative table tbody tr",
			Title:      "div.market-info-list-cont p.tit a.subject-link span.ellipsis-with-reply-cnt",
			Link:       "div.market-info-list-cont p.tit a.subject-link",
			Thumbnail:  "div.market-info-list div.thumb-wrap a.thumb img.maxImg",
			PostedAt:   "span.date",
			PriceRegex: `([0-9,]+원)`,
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 6)
		},
	}, cacheSvc)
}
