package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

func NewDealbadaCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.DealbadaURL + "/bbs/board.php?bo_table=deal_domestic",
		CacheKey:     "dealbada_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.DealbadaURL,
		Provider:     ProviderDealbada,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:    "div.tbl_head01.tbl_wrap table.hoverTable tbody tr",
			Title:       "td.td_subject a",
			Link:        "td.td_subject a",
			Thumbnail:   "td.td_img a img",
			PostedAt:    "td.td_date",
			PriceRegex:  `([0-9,]+원)`,
			ClassFilter: "bo_notice best_article",
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "&wr_id=", 1)
		},
	}, cacheSvc)
}
