package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// NewQuasarCrawler creates a Quasar crawler
func NewQuasarCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	priceHandler := func(s *goquery.Selection) string {
		priceSel := s.Find("div.market-info-sub p span span")
		if priceSel.Length() == 0 {
			return ""
		}
		return strings.TrimSpace(priceSel.Text())
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.QuasarURL + "/bbs/qb_saleinfo",
		CacheKey:     "quasar_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.QuasarURL,
		Provider:     ProviderQuasar,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:      "div.market-type-list.market-info-type-list.relative table tbody tr",
			Title:         "div.market-info-list-cont p.tit a.subject-link span.ellipsis-with-reply-cnt",
			Link:          "div.market-info-list-cont p.tit a.subject-link",
			Thumbnail:     "div.market-info-list div.thumb-wrap a.thumb img.maxImg",
			PostedAt:      "span.date",
			PriceHandlers: []ElementHandler{priceHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 6)
		},
	}, cacheSvc)
}
