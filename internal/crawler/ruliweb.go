package crawler

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// RuliwebCrawler crawls hot deals from Ruliweb
type RuliwebCrawler struct {
	BaseCrawler
}

// NewRuliwebCrawler creates a new Ruliweb crawler
func NewRuliwebCrawler(url string, cacheSvc cache.CacheService) *RuliwebCrawler {
	return &RuliwebCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "ruliweb_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *RuliwebCrawler) GetName() string {
	return "RuliwebCrawler"
}

// FetchDeals fetches deals from Ruliweb
func (c *RuliwebCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("tr.table_body.normal")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *RuliwebCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	var title, link, price, thumbnail string

	if subj := s.Find("td.subject a.subject_link"); subj.Length() > 0 {
		title = strings.TrimSpace(subj.Text())
		if href, exists := subj.Attr("href"); exists {
			link = resolveURL(c.URL, href)
		}
	} else if subj := s.Find("div.title_wrapper a.subject_link"); subj.Length() > 0 {
		title = strings.TrimSpace(subj.Text())
		if href, exists := subj.Attr("href"); exists {
			link = resolveURL(c.URL, href)
		}
	}

	id, err := helpers.GetSplitPart(strings.Split(link, "?")[0], "/", 7)
	if err != nil {
		return nil, err
	}

	rePrice := regexp.MustCompile(`\\(([\d,]+)\\)$`)
	if m := rePrice.FindStringSubmatch(title); len(m) > 1 {
		price = strings.TrimSpace(m[1])
		title = strings.TrimSpace(strings.TrimSuffix(title, "("+m[1]+")"))
	}

	if img := s.Find("a.baseList-thumb img"); img.Length() > 0 {
		if src, exists := img.Attr("src"); exists {
			thumbnail = resolveURL(c.URL, strings.TrimSpace(src))
		}
	} else if thumb := s.Find("a.thumbnail"); thumb.Length() > 0 {
		if style, exists := thumb.Attr("style"); exists {
			thumbnail = extractURLFromStyle(style)
			thumbnail = resolveURL(c.URL, thumbnail)
		}
	}

	postedAt := strings.TrimSpace(s.Find("div.article_info span.time").Text())
	postedAt = strings.TrimSpace(strings.TrimPrefix(postedAt, "날짜"))

	if title != "" && link != "" {
		return &HotDeal{
			Id:        id,
			Title:     title,
			Link:      link,
			Price:     price,
			Thumbnail: thumbnail,
			PostedAt:  postedAt,
			Provider:  "Ruliweb",
		}, nil
	}

	return nil, nil
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(baseStr, href string) string {
	base, err := url.Parse(baseStr)
	if err != nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(ref).String()
}

// extractURLFromStyle extracts a URL from a CSS style attribute
func extractURLFromStyle(style string) string {
	re := regexp.MustCompile(`url\\((?:['"]?)(.*?)(?:['"]?)\\)`)
	if matches := re.FindStringSubmatch(style); len(matches) > 1 {
		return matches[1]
	}
	return ""
}
