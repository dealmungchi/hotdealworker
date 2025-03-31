package crawler

import (
	"fmt"
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// CreateCrawlers creates all the crawlers based on the configuration
func CreateCrawlers(cfg *config.Config, cacheSvc cache.CacheService) []Crawler {
	// Use only configurable crawlers
	crawlers := createConfigurableCrawlers(cfg, cacheSvc)

	// Add debug logging
	fmt.Printf("Created %d crawlers\n", len(crawlers))
	for i, c := range crawlers {
		configurableCrawler, ok := c.(*ConfigurableCrawler)
		if ok {
			fmt.Printf("Crawler %d: %s with URL %s\n", i, c.GetName(), configurableCrawler.URL)
		} else {
			fmt.Printf("Crawler %d: %s (not a ConfigurableCrawler)\n", i, c.GetName())
		}
	}

	return crawlers
}

// createConfigurableCrawlers creates crawlers with the configurable approach
func createConfigurableCrawlers(cfg *config.Config, cacheSvc cache.CacheService) []Crawler {
	// Define crawler configurations
	configurations := []CrawlerConfig{
		{
			// Clien crawler configuration
			URL:       cfg.ClienURL,
			CacheKey:  "clien_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://www.clien.net",
			Provider:  "Clien",
			Selectors: Selectors{
				DealList:    "div.list_item.symph_row.jirum",
				Title:       "span.list_subject",
				Link:        "a[data-role='list-title-text']",
				Thumbnail:   "div.list_img a.list_thumbnail img",
				PostedAt:    "div.list_time span.time.popover span.timestamp",
				PriceRegex:  `\(([0-9,]+원)\)$`,
				ClassFilter: "blocked",
			},
			IDExtractor: func(link string) (string, error) {
				baseLink := strings.Split(link, "?")[0]
				return helpers.GetSplitPart(baseLink, "/", 6)
			},
		},
		{
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
				PostedAt:   "div.article_info span.time",
				PriceRegex: `\(([\d,]+)\)$`,
				ThumbRegex: `url\((?:['"]?)(.*?)(?:['"]?)\)`,
			},
			IDExtractor: func(link string) (string, error) {
				baseLink := strings.Split(link, "?")[0]
				return helpers.GetSplitPart(baseLink, "/", 7)
			},
		},
		{
			// FMKorea crawler configuration
			URL:       cfg.FMKoreaURL,
			CacheKey:  "fmkorea_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://www.fmkorea.com",
			Provider:  "FMKorea",
			Selectors: Selectors{
				DealList:   "ul li.li",
				Title:      "h3.title a",
				Link:       "h3.title a",
				Thumbnail:  "a img.thumb",
				PostedAt:   "div span.regdate",
				PriceRegex: `\(([0-9,]+원)\)$`,
			},
			IDExtractor: func(link string) (string, error) {
				return helpers.GetSplitPart(link, "/", 3)
			},
		},
		{
			// Ppom crawler configuration
			URL:       cfg.PpomURL,
			CacheKey:  "ppom_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://www.ppomppu.co.kr",
			Provider:  "Ppom",
			Selectors: Selectors{
				DealList:   "tr.baseList.bbs_new1",
				Title:      "div.baseList-cover a.baseList-title",
				Link:       "div.baseList-cover a.baseList-title",
				Thumbnail:  "a.baseList-thumb img",
				PostedAt:   "time.baseList-time",
				PriceRegex: `\(([0-9,]+원)\)$`,
			},
			IDExtractor: func(link string) (string, error) {
				return helpers.GetSplitPart(link, "no=", 1)
			},
		},
		{
			// PpomEn crawler configuration
			URL:       cfg.PpomEnURL,
			CacheKey:  "ppom_en_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://www.ppomppu.co.kr",
			Provider:  "PpomEn",
			Selectors: Selectors{
				DealList:   "tr.baseList.bbs_new1",
				Title:      "div.baseList-cover a.baseList-title",
				Link:       "div.baseList-cover a.baseList-title",
				Thumbnail:  "a.baseList-thumb img",
				PostedAt:   "time.baseList-time",
				PriceRegex: `\$([\d,.]+)`,
			},
			IDExtractor: func(link string) (string, error) {
				return helpers.GetSplitPart(link, "no=", 1)
			},
		},
		{
			// Quasar crawler configuration
			URL:       cfg.QuasarURL,
			CacheKey:  "quasar_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://quasarzone.com",
			Provider:  "Quasar",
			Selectors: Selectors{
				DealList:   "div.market-type-list.market-info-type-list.relative table tbody tr",
				Title:      "div.market-info-list-cont p.tit a.subject-link span.ellipsis-with-reply-cnt",
				Link:       "div.market-info-list-cont p.tit a.subject-link",
				Thumbnail:  "div.market-info-list div.thumb-wrap a.thumb img.maxImg",
				PostedAt:   "span.date",
				PriceRegex: `([0-9,]+원)`,
			},
			IDExtractor: func(link string) (string, error) {
				return helpers.GetSplitPart(link, "/", 6)
			},
		},
		{
			// Damoang crawler configuration
			URL:       cfg.DamoangURL,
			CacheKey:  "damoang_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://damoang.net",
			Provider:  "Damoang",
			Selectors: Selectors{
				DealList:   "section#bo_list ul.list-group.list-group-flush.border-bottom li:not(.hd-wrap):not(.da-atricle-row--notice)",
				Title:      "a.da-link-block.da-article-link.subject-ellipsis",
				Link:       "a.da-link-block.da-article-link.subject-ellipsis",
				PostedAt:   "span.orangered.da-list-date, div.wr-date.text-nowrap",
				PriceRegex: `\(([0-9,]+원)\)$`,
				// Custom handler for Damoang's postedAt extraction
				PostedAtHandler: func(s *goquery.Selection) string {
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
				},
			},
			IDExtractor: func(link string) (string, error) {
				return helpers.GetSplitPart(link, "/", 4)
			},
		},
		{
			// Arca crawler configuration
			URL:       cfg.ArcaURL,
			CacheKey:  "arca_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://arca.live",
			Provider:  "Arca",
			Selectors: Selectors{
				DealList:   "div.list-table.hybrid div.vrow.hybrid",
				Title:      "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
				Link:       "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
				Thumbnail:  "a.title.preview-image div.vrow-preview img",
				PostedAt:   "span.col-time time",
				PriceRegex: `\(([0-9,]+원)\)$`,
				RemoveElements: []ElementRemoval{
					{Selector: "span", ApplyToPath: "title"},
				},
			},
			IDExtractor: func(link string) (string, error) {
				baseLink := strings.Split(link, "?")[0]
				return helpers.GetSplitPart(baseLink, "/", 5)
			},
		},
		{
			// Coolandjoy crawler configuration
			URL:       cfg.CoolandjoyURL,
			CacheKey:  "coolandjoy_rate_limited",
			BlockTime: 500,
			BaseURL:   "https://coolenjoy.net",
			Provider:  "Coolandjoy",
			Selectors: Selectors{
				DealList:   "ul.na-table li",
				Title:      "a.na-subject",
				Link:       "a.na-subject",
				Thumbnail:  "", // No thumbnail
				PostedAt:   "div.float-left.float-md-none.d-md-table-cell.nw-6.nw-md-auto.f-sm.font-weight-normal.py-md-2.pr-md-1",
				PriceRegex: `\(([0-9,]+원)\)$`,
				RemoveElements: []ElementRemoval{
					{Selector: "i", ApplyToPath: "postedAt"},
					{Selector: "span", ApplyToPath: "postedAt"},
				},
			},
			IDExtractor: func(link string) (string, error) {
				return helpers.GetSplitPart(link, "/", 5)
			},
		},
	}

	// Create crawlers from configurations
	var crawlers []Crawler
	for _, config := range configurations {
		crawlers = append(crawlers, NewConfigurableCrawler(config, cacheSvc))
	}

	return crawlers
}
