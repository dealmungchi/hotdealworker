package crawler

import (
	"regexp"
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// NewCoolandjoyCrawler creates a Coolandjoy crawler
func NewCoolandjoyCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	// 특수한 형태의 썸네일 이미지 추출을 위한 핸들러
	thumbnailHandler := func(s *goquery.Selection) string {
		thumbSel := s.Find(".thumb-img")
		if thumbSel.Length() == 0 {
			return ""
		}

		// Get background-image from style
		style, exists := thumbSel.Attr("style")
		if !exists {
			return ""
		}

		// Extract URL from background-image style
		re := regexp.MustCompile(`url\((?:['"]?)(.*?)(?:['"]?)\)`)
		if matches := re.FindStringSubmatch(style); len(matches) > 1 {
			thumbURL := matches[1]
			// Create a new unified crawler just for this operation
			tempCrawler := UnifiedCrawler{
				BaseCrawler: BaseCrawler{
					BaseURL: cfg.CoolandjoyURL,
				},
			}
			thumbnail, thumbnailLink, _ := tempCrawler.ProcessImage(thumbURL)
			return thumbnail + "|" + thumbnailLink
		}

		return ""
	}

	// 게시 시간에서 불필요한 요소 제거
	postedAtHandler := func(s *goquery.Selection) string {
		postedAtSel := s.Find("div.float-left.float-md-none.d-md-table-cell.nw-6.nw-md-auto.f-sm.font-weight-normal.py-md-2.pr-md-1")
		if postedAtSel.Length() == 0 {
			return ""
		}

		// Clone to avoid modifying the original selection
		postedAtSelClone := postedAtSel.Clone()
		// Remove unwanted elements
		postedAtSelClone.Find("i").Remove()
		postedAtSelClone.Find("span").Remove()
		return strings.TrimSpace(postedAtSelClone.Text())
	}

	priceHandler := func(s *goquery.Selection) string {
		priceSel := s.Find("div.float-right.float-md-none.d-md-table-cell.nw-7.nw-md-auto.text-right.f-sm.font-weight-normal.pl-2.py-md-2.pr-md-1 font")
		if priceSel.Length() == 0 {
			return ""
		}
		return strings.TrimSpace(priceSel.Text())
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.CoolandjoyURL + "/bbs/jirum",
		CacheKey:     "coolandjoy_rate_limited",
		BlockTime:    300,
		BaseURL:      cfg.CoolandjoyURL,
		Provider:     ProviderCoolandjoy,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:          "ul.na-table li",
			Title:             "a.na-subject",
			Link:              "a.na-subject",
			Thumbnail:         "", // No thumbnail
			PostedAt:          "div.float-left.float-md-none.d-md-table-cell.nw-6.nw-md-auto.f-sm.font-weight-normal.py-md-2.pr-md-1",
			ThumbnailHandlers: []ElementHandler{thumbnailHandler},
			PostedAtHandlers:  []ElementHandler{postedAtHandler},
			PriceHandlers:     []ElementHandler{priceHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 5)
		},
	}, cacheSvc)
}
