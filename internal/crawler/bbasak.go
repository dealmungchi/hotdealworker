package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func NewBbasak(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {

	titleCleanerHandler := func(s *goquery.Selection) string {
		// Find the element
		titleSel := s.Find("td.tit p.ffe0002 a")
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

	postedAtCleanerHandler := func(s *goquery.Selection) string {
		element := s.Find("td p.etc2.fthm").Eq(1)
		if element.Length() == 0 {
			return ""
		}
		return strings.TrimSpace(element.Text())
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.BbasakURL + "/bbs/board.php?bo_table=bbasak1",
		CacheKey:     "bbasak_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.BbasakURL,
		Provider:     ProviderBbasak,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:         "table.t1 tbody tr",
			Title:            "td.tit p.ffe0002 a",
			Link:             "td.tit p.ffe0002 a",
			Thumbnail:        "td a.bigSizeLink img",
			PostedAt:         "td p.etc2.fthm",
			PriceRegex:       `([0-9,]+원)`,
			TitleHandlers:    []ElementHandler{titleCleanerHandler},
			PostedAtHandlers: []ElementHandler{postedAtCleanerHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "&wr_id=", 1)
		},
	}, cacheSvc)
}
