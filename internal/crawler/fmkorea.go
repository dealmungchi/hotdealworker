package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// NewFMKoreaCrawler creates an FMKorea crawler
func NewFMKoreaCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	// 제목에서 span을 제거하는 핸들러
	titleCleanerHandler := func(s *goquery.Selection) string {
		// Find the element
		titleSel := s.Find("h3.title a")
		if titleSel.Length() == 0 {
			return ""
		}

		// Clone the selection to avoid modifying the original
		cleanTitle := titleSel.Clone()

		// Remove spans
		cleanTitle.Find("span").Remove()

		// Return the cleaned title
		return strings.TrimSpace(cleanTitle.Text())
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.FMKoreaURL + "/hotdeal",
		CacheKey:     "fmkorea_rate_limited",
		BlockTime:    300,
		BaseURL:      cfg.FMKoreaURL,
		Provider:     "FMKorea",
		UseChrome:    true, // FMKorea는 항상 Chrome 사용
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:      "ul li.li",
			Title:         "h3.title a",
			Link:          "h3.title a",
			Thumbnail:     "a img.thumb",
			PostedAt:      "div span.regdate",
			PriceRegex:    `\(([0-9,]+원)\)$`,
			TitleHandlers: []ElementHandler{titleCleanerHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 3)
		},
	}, cacheSvc)
}