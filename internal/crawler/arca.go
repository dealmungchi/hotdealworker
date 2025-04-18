package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// NewArcaCrawler creates an Arca crawler
func NewArcaCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	// 제목에서 span 태그를 제거하는 핸들러
	titleHandler := func(s *goquery.Selection) string {
		titleSel := s.Find("div.vrow-inner div.vrow-top.deal a.title.hybrid-title")
		if titleSel.Length() == 0 {
			return ""
		}

		// Clone to avoid modifying the original selection
		cleanTitle := titleSel.Clone()
		// Remove spans
		cleanTitle.Find("span").Remove()

		return strings.TrimSpace(cleanTitle.Text())
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.ArcaURL + "/b/hotdeal",
		CacheKey:     "arca_rate_limited",
		BlockTime:    300,
		BaseURL:      cfg.ArcaURL,
		Provider:     "Arca",
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:      "div.list-table.hybrid div.vrow.hybrid",
			Title:         "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
			Link:          "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
			Thumbnail:     "a.title.preview-image div.vrow-preview img",
			PostedAt:      "span.col-time time",
			PriceRegex:    `\(([0-9,]+원)\)$`,
			TitleHandlers: []ElementHandler{titleHandler},
		},
		IDExtractor: func(link string) (string, error) {
			baseLink := strings.Split(link, "?")[0]
			return helpers.GetSplitPart(baseLink, "/", 5)
		},
	}, cacheSvc)
}
