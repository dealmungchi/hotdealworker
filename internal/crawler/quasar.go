package crawler

import (
	"errors"
	"strings"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// QuasarCrawler crawls hot deals from Quasar Zone
type QuasarCrawler struct {
	BaseCrawler
}

// NewQuasarCrawler creates a new Quasar Zone crawler
func NewQuasarCrawler(url string, cacheSvc cache.CacheService) *QuasarCrawler {
	return &QuasarCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "quasar_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *QuasarCrawler) GetName() string {
	return "QuasarCrawler"
}

// FetchDeals fetches deals from Quasar Zone
func (c *QuasarCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("div.market-type-list.market-info-type-list.relative table tbody tr")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *QuasarCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	titleSel := s.Find("div.market-info-list-cont p.tit a.subject-link span.ellipsis-with-reply-cnt")
	title := strings.TrimSpace(titleSel.Text())
	link, exists := s.Find("div.market-info-list-cont p.tit a.subject-link").Attr("href")
	if !exists || title == "" {
		return nil, errors.New("title or links not found")
	}

	if strings.HasPrefix(link, "/") {
		link = "https://quasarzone.com" + link
	}

	id, err := helpers.GetSplitPart(link, "/", 6)
	if err != nil {
		return nil, err
	}

	price := strings.TrimSpace(s.Find("div.market-info-list-cont div.market-info-sub p").First().Find("span.text-orange").Text())

	thumb, _ := s.Find("div.market-info-list div.thumb-wrap a.thumb img.maxImg").Attr("src")
	if thumb != "" && strings.HasPrefix(thumb, "//") {
		thumb = "https:" + thumb
	}

	postedAt := strings.TrimSpace(s.Find("span.date").Text())

	return &HotDeal{
		Id:        id,
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumb,
		PostedAt:  postedAt,
		Provider:  "Quasar",
	}, nil
}
