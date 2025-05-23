package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// NewDamoangCrawler creates a Damoang crawler
func NewDamoangCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	// 게시 시간 추출을 위한 커스텀 핸들러
	postedAtHandler := func(s *goquery.Selection) string {
		// First try the first selector: span.orangered.da-list-date
		postedAt := s.Find("span.orangered.da-list-date").Text()
		if postedAt == "" {
			// If that fails, try the second approach with removal
			postedAtSel := s.Find("div.wr-date.text-nowrap")
			// Clone to avoid modifying the original selection
			postedAtSelClone := postedAtSel.Clone()
			// Remove unwanted elements
			postedAtSelClone.Find("i").Remove()
			postedAtSelClone.Find("span").Remove()
			postedAt = strings.TrimSpace(postedAtSelClone.Text())
		} else {
			postedAt = strings.TrimSpace(postedAt)
		}
		return postedAt
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.DamoangURL + "/economy",
		CacheKey:     "damoang_rate_limited",
		BlockTime:    300,
		BaseURL:      cfg.DamoangURL,
		Provider:     ProviderDamoang,
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:         "section#bo_list ul.list-group.list-group-flush.border-bottom li:not(.hd-wrap):not(.da-atricle-row--notice)",
			Title:            "a.da-link-block.da-article-link.subject-ellipsis",
			Link:             "a.da-link-block.da-article-link.subject-ellipsis",
			PriceRegex:       `\(([0-9,]+원)\)$`,
			PostedAtHandlers: []ElementHandler{postedAtHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 4)
		},
	}, cacheSvc)
}
