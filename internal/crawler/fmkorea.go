package crawler

import (
	"errors"
	"strings"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// FMKoreaCrawler crawls hot deals from FMKorea
type FMKoreaCrawler struct {
	BaseCrawler
}

// NewFMKoreaCrawler creates a new FMKorea crawler
func NewFMKoreaCrawler(url string, cacheSvc cache.CacheService) *FMKoreaCrawler {
	return &FMKoreaCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "fm_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *FMKoreaCrawler) GetName() string {
	return "FMKoreaCrawler"
}

// FetchDeals fetches deals from FMKorea
func (c *FMKoreaCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("ul li.li")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *FMKoreaCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	a := s.Find("h3.title a")
	title := strings.TrimSpace(a.Text())
	link, exists := a.Attr("href")
	if !exists || title == "" {
		return nil, errors.New("title or link not found")
	}

	if strings.HasPrefix(link, "/") {
		link = "https://www.fmkorea.com" + link
	}

	id, err := helpers.GetSplitPart(link, "/", 3)
	if err != nil {
		return nil, err
	}

	price := strings.TrimSpace(s.Find(".hotdeal_info span a").Eq(1).Text())

	thumb, _ := s.Find("a img.thumb").Attr("data-original")
	if thumb == "" {
		thumb, _ = s.Find("a img.thumb").Attr("src")
	}
	if strings.HasPrefix(thumb, "//") {
		thumb = "https:" + thumb
	}

	postedAt := strings.TrimSpace(s.Find("div span.regdate").Text())

	return &HotDeal{
		Id:        id,
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumb,
		PostedAt:  postedAt,
		Provider:  "FMKorea",
	}, nil
}
