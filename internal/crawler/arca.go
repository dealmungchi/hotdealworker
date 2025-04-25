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

	priceHandler := func(s *goquery.Selection) string {
		priceSel := s.Find("span.deal-price")
		if priceSel.Length() == 0 {
			return ""
		}

		return strings.TrimSpace(priceSel.Text())
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.ArcaURL + "/b/hotdeal",
		CacheKey:     "arca_rate_limited",
		BlockTime:    300,
		BaseURL:      cfg.ArcaURL,
		Provider:     ProviderArca,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:      "div.list-table.hybrid div.vrow.hybrid",
			Title:         "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
			Link:          "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
			Thumbnail:     "a.title.preview-image div.vrow-preview img",
			PostedAt:      "span.col-time time",
			Category:      "span.badges a.badge",
			PriceRegex:    `\(([0-9,]+원)\)$`,
			TitleHandlers: []ElementHandler{titleHandler},
			PriceHandlers: []ElementHandler{priceHandler},
		},
		IDExtractor: func(link string) (string, error) {
			baseLink := strings.Split(link, "?")[0]
			return helpers.GetSplitPart(baseLink, "/", 5)
		},
	}, cacheSvc)
}
