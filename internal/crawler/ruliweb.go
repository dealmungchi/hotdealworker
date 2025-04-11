package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// NewRuliwebCrawler creates a Ruliweb crawler
func NewRuliwebCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	// 게시 시간 추출 핸들러
	postedAtHandler := func(s *goquery.Selection) string {
		postedAt := strings.TrimSpace(s.Find("div.article_info span.time").Text())
		postedAt = strings.TrimSpace(strings.TrimPrefix(postedAt, "날짜"))
		return postedAt
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.RuliwebURL + "/market/board/1020?view=thumbnail&page=1",
		CacheKey:     "ruliweb_rate_limited",
		BlockTime:    300,
		BaseURL:      cfg.RuliwebURL,
		Provider:     "Ruliweb",
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:         "tr.table_body.normal",
			Title:            "td.subject a.subject_link, div.title_wrapper a.subject_link",
			Link:             "td.subject a.subject_link, div.title_wrapper a.subject_link",
			Thumbnail:        "a.baseList-thumb img, a.thumbnail",
			ThumbRegex:       `url\((?:['"]?)(.*?)(?:['"]?)\)`,
			PriceRegex:       `\(([0-9,]+원)\)$`,
			PostedAtHandlers: []ElementHandler{postedAtHandler},
		},
		IDExtractor: func(link string) (string, error) {
			baseLink := strings.Split(link, "?")[0]
			return helpers.GetSplitPart(baseLink, "/", 7)
		},
	}, cacheSvc)
}
