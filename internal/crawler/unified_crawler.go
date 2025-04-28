package crawler

import (
	"io"
	"strings"
	"time"

	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// UnifiedCrawler is a crawler that handles both standard HTTP and Chrome crawling
type UnifiedCrawler struct {
	BaseCrawler
	Selectors    Selectors
	ChromeDBAddr string
	fetchFunc    func() (io.Reader, error) // 크롤러별로 사용할 fetch 함수
}

// NewUnifiedCrawler creates a new unified crawler
func NewUnifiedCrawler(config CrawlerConfig, cacheSvc cache.CacheService) *UnifiedCrawler {
	unified := &UnifiedCrawler{
		BaseCrawler: BaseCrawler{
			URL:         config.URL,
			CacheKey:    config.CacheKey,
			CacheSvc:    cacheSvc,
			BlockTime:   time.Duration(config.BlockTime) * time.Second,
			BaseURL:     config.BaseURL,
			Provider:    config.Provider,
			IDExtractor: config.IDExtractor,
			PriceRegex:  config.Selectors.PriceRegex,
		},
		Selectors:    config.Selectors,
		ChromeDBAddr: config.ChromeDBAddr,
	}

	// 크롤러 타입에 따라 fetch 함수 설정
	if config.UseChrome && unified.ChromeDBAddr != "" {
		logger.Info("Using ChromeDB for %s", config.Provider)
		unified.fetchFunc = unified.fetchWithChromeDB
	} else {
		logger.Info("Using standard fetch for %s", config.Provider)
		unified.fetchFunc = unified.fetchWithCache
	}

	return unified
}

// FetchDeals fetches deals using the unified approach
func (c *UnifiedCrawler) FetchDeals() ([]HotDeal, error) {
	// Fetch the page using appropriate method
	utf8Body, err := c.fetchFunc()
	if err != nil {
		return nil, err
	}

	// Parse the HTML document
	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, err
	}

	// Find all deal items
	dealSelections := doc.Find(c.Selectors.DealList)

	// Process deals
	deals := c.processDeals(dealSelections, c.processDeal)

	return deals, nil
}

// applyHandlers applies a series of handlers to a selection
func (c *UnifiedCrawler) applyHandlers(s *goquery.Selection, handlers []ElementHandler) string {
	if len(handlers) == 0 {
		return ""
	}

	result := ""
	for _, handler := range handlers {
		if handler != nil {
			// Apply the handler and get the result
			result = handler(s)
			// If we got a result, we can stop applying handlers
			if result != "" {
				break
			}
		}
	}
	return result
}

// defaultTitleHandler is the default handler for extracting titles
func (c *UnifiedCrawler) defaultTitleHandler(s *goquery.Selection) string {
	titleSel := s.Find(c.Selectors.Title)

	if titleSel.Length() == 0 {
		return ""
	}

	var title string
	if titleAttr, exists := titleSel.Attr("title"); exists && titleAttr != "" {
		title = titleAttr
	} else {
		title = titleSel.Text()
	}

	return strings.TrimSpace(title)
}

// defaultLinkHandler is the default handler for extracting links
func (c *UnifiedCrawler) defaultLinkHandler(s *goquery.Selection) string {
	linkSel := s.Find(c.Selectors.Link)
	if linkSel.Length() == 0 {
		return ""
	}

	link, exists := linkSel.Attr("href")
	if !exists {
		return ""
	}

	return c.ResolveURL(strings.TrimSpace(link))
}

// defaultThumbnailHandler is the default handler for extracting thumbnails
func (c *UnifiedCrawler) defaultThumbnailHandler(s *goquery.Selection) (string, string) {
	thumbSel := s.Find(c.Selectors.Thumbnail)

	if thumbSel.Length() == 0 {
		return "", ""
	}

	if src, exists := thumbSel.Attr("src"); exists {
		thumbnail, thumbnailLink, _ := c.ProcessImage(src)
		return thumbnail, thumbnailLink
	} else if style, exists := thumbSel.Attr("style"); exists && c.Selectors.ThumbRegex != "" {
		thumbURL := c.ExtractURLFromStyle(style)
		thumbnail, thumbnailLink, _ := c.ProcessImage(thumbURL)
		return thumbnail, thumbnailLink
	}

	return "", ""
}

// defaultPostedAtHandler is the default handler for extracting posted time
func (c *UnifiedCrawler) defaultPostedAtHandler(s *goquery.Selection) string {
	postedAtSel := s.Find(c.Selectors.PostedAt)
	if postedAtSel.Length() == 0 {
		return ""
	}

	return strings.TrimSpace(postedAtSel.Text())
}

func (c *UnifiedCrawler) defaultCategoryHandler(s *goquery.Selection) string {
	categorySel := s.Find(c.Selectors.Category)
	if categorySel.Length() == 0 {
		return ""
	}

	return strings.TrimSpace(categorySel.Text())
}

// processDeal processes a single deal based on the configuration
func (c *UnifiedCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	// Skip if the element has a class to filter out
	if c.Selectors.ClassFilter != "" && s.HasClass(c.Selectors.ClassFilter) {
		return nil, nil
	}

	// Extract title
	var title string
	if len(c.Selectors.TitleHandlers) > 0 {
		title = strings.TrimSpace(c.applyHandlers(s, c.Selectors.TitleHandlers))
	} else {
		title = c.defaultTitleHandler(s)
	}

	if title == "" {
		return nil, nil
	}

	// Extract link
	var link string
	if len(c.Selectors.LinkHandlers) > 0 {
		link = c.applyHandlers(s, c.Selectors.LinkHandlers)
	} else {
		link = c.defaultLinkHandler(s)
	}

	link = strings.TrimSpace(link)
	if link == "" {
		return nil, nil
	}

	// Extract ID from the link
	var id string
	var err error
	if c.IDExtractor != nil {
		id, err = c.IDExtractor(link)
		if err != nil {
			return nil, err
		}
	}

	// Extract price
	var price string

	// 1. First try to extract price from the price handler if defined and the Price selector is set
	if len(c.Selectors.PriceHandlers) > 0 {
		price = c.applyHandlers(s, c.Selectors.PriceHandlers)
	}

	// 2. If no price was found and there's a regex pattern, try to extract from the title
	if price == "" && c.PriceRegex != "" {
		title, price = c.ExtractPrice(title)
	}

	// Extract thumbnail
	var thumbnail, thumbnailLink string
	if len(c.Selectors.ThumbnailHandlers) > 0 {
		// For thumbnail handlers, we need a special approach since they return two values
		for _, handler := range c.Selectors.ThumbnailHandlers {
			if handler != nil {
				result := handler(s)
				if result != "" {
					// Assuming the handler returns a JSON string with thumbnail and thumbnailLink
					parts := strings.Split(result, "|")
					if len(parts) == 2 {
						thumbnail = parts[0]
						thumbnailLink = parts[1]
						break
					}
				}
			}
		}
	} else if c.Selectors.Thumbnail != "" {
		thumbnail, thumbnailLink = c.defaultThumbnailHandler(s)
	}

	// Extract posted time
	var postedAt string
	if len(c.Selectors.PostedAtHandlers) > 0 {
		postedAt = c.applyHandlers(s, c.Selectors.PostedAtHandlers)
	} else {
		postedAt = c.defaultPostedAtHandler(s)
	}

	var category string
	if len(c.Selectors.CategoryHandlers) > 0 {
		category = c.applyHandlers(s, c.Selectors.CategoryHandlers)
	} else {
		category = c.defaultCategoryHandler(s)
	}

	if category != "" {
		category = classifyCategory(category)
	}

	return c.CreateDeal(id, title, link, price, thumbnail, thumbnailLink, postedAt, category), nil
}
