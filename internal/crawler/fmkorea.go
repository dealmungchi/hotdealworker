package crawler

import (
	"regexp"
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

	priceHandler := func(s *goquery.Selection) string {
		priceText := s.Find("div.hotdeal_info span a").Text()
		re := regexp.MustCompile(`[\d,]+원`)
		match := re.FindString(priceText)
		return strings.TrimSpace(match)
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.FMKoreaURL + "/hotdeal",
		CacheKey:     "fmkorea_rate_limited",
		BlockTime:    300,
		BaseURL:      cfg.FMKoreaURL,
		Provider:     ProviderFMKorea,
		UseChrome:    true,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:      "ul li.li",
			Title:         "h3.title a",
			Link:          "h3.title a",
			Thumbnail:     "a img.thumb",
			PostedAt:      "div span.regdate",
			Category:      "div span.category a",
			PriceRegex:    `\(([0-9,]+원)\)$`,
			TitleHandlers: []ElementHandler{titleCleanerHandler},
			PriceHandlers: []ElementHandler{priceHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 3)
		},
	}, cacheSvc)
}
