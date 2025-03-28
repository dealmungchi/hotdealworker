package crawler

import (
	"errors"
	"strings"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// DamoangCrawler crawls hot deals from Damoang
type DamoangCrawler struct {
	BaseCrawler
}

// NewDamoangCrawler creates a new Damoang crawler
func NewDamoangCrawler(url string, cacheSvc cache.CacheService) *DamoangCrawler {
	return &DamoangCrawler{
		BaseCrawler: BaseCrawler{
			URL:       url,
			CacheKey:  "damoang_rate_limited",
			CacheSvc:  cacheSvc,
			BlockTime: 500 * time.Second,
		},
	}
}

// GetName returns the crawler name
func (c *DamoangCrawler) GetName() string {
	return "DamoangCrawler"
}

// FetchDeals fetches deals from Damoang
func (c *DamoangCrawler) FetchDeals() ([]HotDeal, error) {
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		return nil, err
	}

	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("section#bo_list ul.list-group.list-group-flush.border-bottom li:not(.hd-wrap):not(.da-atricle-row--notice)")
	return c.processDeals(dealSelections, c.processDeal), nil
}

// processDeal processes a single deal
func (c *DamoangCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	a := s.Find("a.da-link-block.da-article-link.subject-ellipsis")
	title := strings.TrimSpace(a.Text())
	link, exists := a.Attr("href")
	if !exists || title == "" {
		return nil, errors.New("title or link not found")
	}

	if strings.HasPrefix(link, "/") {
		link = "https://damoang.net" + link
	}

	id, err := helpers.GetSplitPart(link, "/", 4)
	if err != nil {
		return nil, err
	}

	postedAt := s.Find("span.orangered.da-list-date").Text()
	if postedAt == "" {
		postedAtSel := s.Find("div.wr-date.text-nowrap")
		postedAtSel.Find("i").Remove()
		postedAtSel.Find("span").Remove()
		postedAt = strings.TrimSpace(postedAtSel.Text())
	} else {
		postedAt = strings.TrimSpace(postedAt)
	}

	return &HotDeal{
		Id:        id,
		Title:     title,
		Link:      link,
		Price:     "",
		Thumbnail: "",
		PostedAt:  postedAt,
		Provider:  "Damoang",
	}, nil
}
