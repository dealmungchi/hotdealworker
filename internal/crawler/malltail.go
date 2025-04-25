package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func NewMalltail(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	categoryCleanerHandler := func(s *goquery.Selection) string {
		element := s.Find("td")
		if element.Length() == 0 {
			return ""
		}

		firstTd := element.Eq(0)
		if firstTd.Length() == 0 {
			return ""
		}

		return strings.TrimSpace(firstTd.Text())
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.MalltailURL + "/hotdeals/index",
		CacheKey:     "malltail_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.MalltailURL,
		Provider:     ProviderMalltail,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:         "div#container div.hotdeal-wrap.event_area table.list tbody tr",
			Title:            "td.title a",
			Link:             "td.title a",
			ClassFilter:      "notice",
			CategoryHandlers: []ElementHandler{categoryCleanerHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 5)
		},
	}, cacheSvc)
}
