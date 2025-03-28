package crawler

import (
	"errors"
	"strings"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// CoolandjoyCrawler crawls hot deals from Coolandjoy
type CoolandjoyCrawler struct {
	BaseCrawler
}

// NewCoolandjoyCrawler creates a new Coolandjoy crawler
func NewCoolandjoyCrawler(url string, cacheSvc cache.CacheService) *CoolandjoyCrawler {
	return &CoolandjoyCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "coolandjoy_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *CoolandjoyCrawler) GetName() string {
	return "CoolandjoyCrawler"
}

// FetchDeals fetches deals from Coolandjoy
func (c *CoolandjoyCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("ul.na-table li")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *CoolandjoyCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	subjectSel := s.Find("a.na-subject")
	title := strings.TrimSpace(subjectSel.Text())
	link, exists := subjectSel.Attr("href")
	if !exists || title == "" {
		return nil, errors.New("title or link not found")
	}

	if strings.HasPrefix(link, "/") {
		link = "https://coolenjoy.net" + link
	}

	// Extract ID from the link
	id, err := helpers.GetSplitPart(link, "/", 5)
	if err != nil {
		return nil, err
	}

	price := strings.TrimSpace(s.Find("div.float-right font").Text())

	thumbnail := ""
	imgSel := s.Find("a.thumb img")
	if src, ok := imgSel.Attr("src"); ok {
		thumbnail = src
	}

	postedAtSel := s.Find("div.float-left.float-md-none.d-md-table-cell.nw-6.nw-md-auto.f-sm.font-weight-normal.py-md-2.pr-md-1")
	postedAtSel.Find("i").Remove()
	postedAtSel.Find("span").Remove()
	postedAt := strings.TrimSpace(postedAtSel.Text())

	return &HotDeal{
		Id:        id,
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumbnail,
		PostedAt:  postedAt,
		Provider:  "Coolandjoy",
	}, nil
}
