package crawler

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"sjsage522/hotdealworker/services/cache"
)

// PpomEnCrawler crawls hot deals from Ppomppu English forum
type PpomEnCrawler struct {
	BaseCrawler
}

// NewPpomEnCrawler creates a new Ppomppu English forum crawler
func NewPpomEnCrawler(url string, cacheSvc cache.CacheService) *PpomEnCrawler {
	return &PpomEnCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "ppom_en_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *PpomEnCrawler) GetName() string {
	return "PpomEnCrawler"
}

// FetchDeals fetches deals from Ppomppu English forum
func (c *PpomEnCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("tr.baseList")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *PpomEnCrawler) processDeal(s *goquery.Selection) *HotDeal {
	if s.Find("td.baseList-numb img[alt='해외포럼 아이콘']").Length() > 0 {
		return nil
	}

	titleSelection := s.Find("a.baseList-title")
	title := strings.TrimSpace(titleSelection.Text())
	if title == "" {
		return nil
	}

	link, exists := titleSelection.Attr("href")
	if !exists {
		return nil
	}
	
	base, err := url.Parse(c.URL)
	if err == nil {
		if ref, err := url.Parse(link); err == nil {
			link = base.ResolveReference(ref).String()
		}
	}

	thumbnail := ""
	thumbSelection := s.Find("a.baseList-thumb img")
	if thumbSelection.Length() > 0 {
		if src, exists := thumbSelection.Attr("src"); exists {
			thumbnail = strings.TrimSpace(src)
			if !strings.HasPrefix(thumbnail, "http") {
				if base, err := url.Parse(c.URL); err == nil {
					if ref, err := url.Parse(thumbnail); err == nil {
						thumbnail = base.ResolveReference(ref).String()
					}
				}
			}
		}
	}

	price := ""
	if start := strings.Index(title, "("); start != -1 {
		if end := strings.Index(title[start:], ")"); end != -1 {
			price = strings.TrimSpace(title[start+1 : start+end])
		}
	}

	postedAt := strings.TrimSpace(s.Find("td.baseList-space time.baseList-time").Text())

	return &HotDeal{
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumbnail,
		PostedAt:  postedAt,
	}
}