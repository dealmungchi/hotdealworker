package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// NewRuliwebCrawler creates a Ruliweb crawler
func NewRuliwebCrawler(cfg *config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	// Create custom handler for posted time
	customHandlers := CustomHandlers{
		ElementHandlers: map[string]CustomElementHandlerFunc{
			"postedAt": func(s *goquery.Selection) string {
				postedAt := strings.TrimSpace(s.Find("div.article_info span.time").Text())
				postedAt = strings.TrimSpace(strings.TrimPrefix(postedAt, "날짜"))
				return postedAt
			},
		},
	}

	return NewConfigurableCrawler(CrawlerConfig{
		// Ruliweb crawler configuration
		URL:       cfg.RuliwebURL,
		CacheKey:  "ruliweb_rate_limited",
		BlockTime: 500,
		BaseURL:   "https://bbs.ruliweb.com",
		Provider:  "Ruliweb",
		Selectors: Selectors{
			DealList:   "tr.table_body.normal",
			Title:      "td.subject a.subject_link, div.title_wrapper a.subject_link",
			Link:       "td.subject a.subject_link, div.title_wrapper a.subject_link",
			Thumbnail:  "a.baseList-thumb img, a.thumbnail",
			PriceRegex: `\(([\d,]+)\)$`,
			ThumbRegex: `url\((?:['"]?)(.*?)(?:['"]?)\)`,
		},
		CustomHandlers: customHandlers,
		IDExtractor: func(link string) (string, error) {
			baseLink := strings.Split(link, "?")[0]
			return helpers.GetSplitPart(baseLink, "/", 7)
		},
	}, cacheSvc)
}
