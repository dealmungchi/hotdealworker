package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

func NewMissycoupons(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.MissycouponsURL + "/zero/board.php#id=hotdeals",
		CacheKey:     "missycoupons_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.MissycouponsURL + "/zero/",
		Provider:     ProviderMissycoupons,
		UseChrome:    true,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:   "form div.rp-list-table div.rp-list-table-row.normal.post",
			Title:      "div.rp-list-table-cell.board-list.mc-l-subject",
			Link:       "div.rp-list-table-cell.board-list.mc-l-subject a",
			Thumbnail:  "a.mc-l-thumbnail",
			PostedAt:   "div.mc_localtime",
			ThumbRegex: `url\((?:['"]?)(.*?)(?:['"]?)\)`,
			PriceRegex: `\(([0-9,]+원)\)$`,
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "no=", 1)
		},
	}, cacheSvc)
}
