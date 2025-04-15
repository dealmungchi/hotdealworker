package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

func NewEomisae(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.EomisaeURL + "/index.php?mid=fs&sort_index=regdate&order_type=desc",
		CacheKey:     "eomisae_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.EomisaeURL,
		Provider:     "Eomisae",
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:  "div.card_wrap div.bd_card.cf div.card_el.n_ntc.clear",
			Title:     "div.rt_area.is_tmb div.card_content h3 a.pjax",
			Link:      "div.rt_area.is_tmb div.card_content h3 a.pjax",
			Thumbnail: "div.rt_area.is_tmb div.tmb_wrp img.tmb",
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "document_srl=", 1)
		},
	}, cacheSvc)
}
