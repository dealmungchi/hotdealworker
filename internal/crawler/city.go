package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

func NewCity(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.CityURL + "/ln",
		CacheKey:     "city_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.CityURL,
		Provider:     ProviderCity,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:    "table.bd_lst.bd_tb_lst.bd_tb tbody tr",
			Title:       "td.title a.hx",
			Link:        "td.title a.hx",
			Thumbnail:   "td a img.thumb_border",
			PostedAt:    "td.time",
			PriceRegex:  `([0-9,]+원)`,
			ClassFilter: "notice",
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 4)
		},
	}, cacheSvc)
}
