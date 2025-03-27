package crawler

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"sjsage522/hotdealworker/services/cache"
)

// ArcaCrawler crawls hot deals from Arca Live
type ArcaCrawler struct {
	BaseCrawler
}

// NewArcaCrawler creates a new Arca crawler
func NewArcaCrawler(url string, cacheSvc cache.CacheService) *ArcaCrawler {
	return &ArcaCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "arca_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *ArcaCrawler) GetName() string {
	return "ArcaCrawler"
}

// FetchDeals fetches deals from Arca
func (c *ArcaCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("div.list-table.hybrid div.vrow.hybrid")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *ArcaCrawler) processDeal(s *goquery.Selection) *HotDeal {
	titleSel := s.Find("div.vrow-inner div.vrow-top.deal a.title.hybrid-title")
	titleSel.Find("span").Remove()
	title := strings.TrimSpace(titleSel.Text())
	link, exists := titleSel.Attr("href")
	if !exists || title == "" {
		return nil
	}
	
	if strings.HasPrefix(link, "/") {
		link = "https://arca.live" + link
	}
	
	price := strings.TrimSpace(s.Find("a.title.hybrid-bottom div.vrow-bottom.deal span.deal-price").Text())
	
	thumb, _ := s.Find("a.title.preview-image div.vrow-preview img").Attr("src")
	if thumb != "" && strings.HasPrefix(thumb, "//") {
		thumb = "https:" + thumb
	}

	postedAt, _ := s.Find("span.col-time time").Attr("datetime")

	return &HotDeal{
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumb,
		PostedAt:  postedAt,
	}
}