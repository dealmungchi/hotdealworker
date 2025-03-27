package crawler

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"sjsage522/hotdealworker/services/cache"
)

// PpomCrawler crawls hot deals from Ppomppu
type PpomCrawler struct {
	BaseCrawler
}

// NewPpomCrawler creates a new Ppomppu crawler
func NewPpomCrawler(url string, cacheSvc cache.CacheService) *PpomCrawler {
	return &PpomCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "ppom_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *PpomCrawler) GetName() string {
	return "PpomCrawler"
}

// FetchDeals fetches deals from Ppomppu
func (c *PpomCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("tr.baseList.bbs_new1")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *PpomCrawler) processDeal(s *goquery.Selection) *HotDeal {
	titleElem := s.Find("div.baseList-cover a.baseList-title")
	if titleElem.Length() == 0 {
		return nil
	}
	
	titleText := strings.TrimSpace(titleElem.Text())
	if titleText == "" {
		return nil
	}

	re := regexp.MustCompile(`\\(([^)]+)\\)$`)
	priceMatch := re.FindStringSubmatch(titleText)
	price := ""
	if len(priceMatch) > 1 {
		price = strings.TrimSpace(priceMatch[1])
		titleText = strings.TrimSpace(strings.TrimSuffix(titleText, "("+priceMatch[1]+")"))
	}

	link, exists := titleElem.Attr("href")
	if !exists || strings.TrimSpace(link) == "" {
		return nil
	}
	
	base, err := url.Parse(c.URL)
	if err == nil {
		if ref, err := url.Parse(link); err == nil {
			link = base.ResolveReference(ref).String()
		}
	}

	thumbElem := s.Find("a.baseList-thumb img")
	thumbnail := ""
	if thumbElem.Length() > 0 {
		if src, exists := thumbElem.Attr("src"); exists {
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

	postedAt := strings.TrimSpace(s.Find("time.baseList-time").Text())

	return &HotDeal{
		Title:     titleText,
		Link:      link,
		Price:     price,
		Thumbnail: thumbnail,
		PostedAt:  postedAt,
	}
}