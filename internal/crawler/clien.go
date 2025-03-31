package crawler

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// ClienCrawler crawls hot deals from Clien
type ClienCrawler struct {
	BaseCrawler
}

// NewClienCrawler creates a new Clien crawler
func NewClienCrawler(url string, cacheSvc cache.CacheService) *ClienCrawler {
	return &ClienCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "clien_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *ClienCrawler) GetName() string {
	return "ClienCrawler"
}

// FetchDeals fetches deals from Clien
func (c *ClienCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("div.list_item.symph_row.jirum")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *ClienCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	if s.HasClass("blocked") {
		return nil, errors.New("selection -> class 'blocked' not found")
	}

	titleAttr, exists := s.Find("span.list_subject").Attr("title")
	if !exists || strings.TrimSpace(titleAttr) == "" {
		return nil, errors.New("title not found")
	}
	title := strings.TrimSpace(titleAttr)

	link, exists := s.Find("a[data-role='list-title-text']").Attr("href")
	if !exists || strings.TrimSpace(link) == "" {
		return nil, errors.New("link not found")
	}
	link = strings.TrimSpace(link)

	if strings.HasPrefix(link, "/") {
		link = "https://www.clien.net" + link
	}

	// Extract ID from the link
	id, err := helpers.GetSplitPart(strings.Split(link, "?")[0], "/", 6)
	if err != nil {
		return nil, err
	}

	var price string
	priceRegex := regexp.MustCompile(`\\(([0-9,]+원)\\)$`)
	if match := priceRegex.FindStringSubmatch(title); len(match) > 1 {
		price = match[1]
		title = strings.TrimSpace(strings.Replace(title, "("+price+")", "", 1))
	}

	thumbnail, _ := s.Find("div.list_img a.list_thumbnail img").Attr("src")
	thumbnail = strings.TrimSpace(thumbnail)
	data, err := helpers.FetchSimply(thumbnail)
	if err != nil {
		return nil, err
	}
	thumbnail = base64.StdEncoding.EncodeToString(data)

	postedAt := strings.TrimSpace(s.Find("div.list_time span.time.popover span.timestamp").Text())

	return &HotDeal{
		Id:        id,
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumbnail,
		PostedAt:  postedAt,
		Provider:  "Clien",
	}, nil
}
