package crawler

import (
	"strings"
	"time"

	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// ConfigurableCrawler is a crawler that can be configured with selectors
type ConfigurableCrawler struct {
	BaseCrawler
	Selectors           Selectors
	CustomHandlers      CustomHandlers
	ElementTransformers ElementTransformers
}

// NewConfigurableCrawler creates a new configurable crawler
func NewConfigurableCrawler(config CrawlerConfig, cacheSvc cache.CacheService) *ConfigurableCrawler {
	return &ConfigurableCrawler{
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
		Selectors:           config.Selectors,
		CustomHandlers:      config.CustomHandlers,
		ElementTransformers: config.ElementTransformers,
	}
}

// FetchDeals fetches deals based on the configuration
func (c *ConfigurableCrawler) FetchDeals() ([]HotDeal, error) {
	// Fetch the page with rate limiting
	utf8Body, err := c.fetchWithCache()
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

// cleanSelection removes specified elements from a selection before getting text
func (c *ConfigurableCrawler) cleanSelection(sel *goquery.Selection, path string) *goquery.Selection {
	if sel.Length() == 0 {
		return sel
	}

	// Clone the selection to avoid modifying the original
	clone := sel.Clone()

	// Apply element removals for this path
	for _, removal := range c.ElementTransformers.RemoveElements {
		if removal.ApplyToPath == path {
			clone.Find(removal.Selector).Remove()
		}
	}

	return clone
}

// processElement extracts text from an element using custom handlers or default method
func (c *ConfigurableCrawler) processElement(s *goquery.Selection, path string, selector string) string {
	// Use custom handler if defined
	if c.CustomHandlers.ElementHandlers != nil {
		if handler, exists := c.CustomHandlers.ElementHandlers[path]; exists && handler != nil {
			return handler(s)
		}
	}

	// Use default handling
	elementSel := s.Find(selector)
	if elementSel.Length() > 0 {
		// Clean the selection if needed
		cleanSel := c.cleanSelection(elementSel, path)
		return strings.TrimSpace(cleanSel.Text())
	}

	return ""
}

// processDeal processes a single deal based on the configuration
func (c *ConfigurableCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	// Skip if the element has a class to filter out
	if c.Selectors.ClassFilter != "" && s.HasClass(c.Selectors.ClassFilter) {
		return nil, nil
	}

	// Extract title
	var title string
	titleSel := s.Find(c.Selectors.Title)
	if titleSel.Length() == 0 {
		return nil, nil
	}

	// Use custom handler if defined, else use default processing
	if handler, exists := c.CustomHandlers.ElementHandlers["title"]; exists && handler != nil {
		title = handler(s)
	} else {
		// Clean the title selection if needed
		cleanTitleSel := c.cleanSelection(titleSel, "title")

		if titleAttr, exists := cleanTitleSel.Attr("title"); exists && titleAttr != "" {
			title = titleAttr
		} else {
			title = cleanTitleSel.Text()
		}
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, nil
	}

	// Extract link
	linkSel := s.Find(c.Selectors.Link)
	if linkSel.Length() == 0 {
		return nil, nil
	}

	link, exists := linkSel.Attr("href")
	if !exists || strings.TrimSpace(link) == "" {
		return nil, nil
	}
	link = c.ResolveURL(strings.TrimSpace(link))

	// Extract ID from the link
	var id string
	var err error
	if c.IDExtractor != nil {
		id, err = c.IDExtractor(link)
		if err != nil {
			return nil, err
		}
	}

	// Extract price from title if regex is set
	var price string
	if c.PriceRegex != "" {
		title, price = c.ExtractPrice(title)
	}

	// Extract thumbnail
	var thumbnail, thumbnailLink string
	if c.Selectors.Thumbnail != "" {
		thumbSel := s.Find(c.Selectors.Thumbnail)
		if thumbSel.Length() > 0 {
			if src, exists := thumbSel.Attr("src"); exists {
				thumbnail, thumbnailLink, _ = c.ProcessImage(src)
			} else if style, exists := thumbSel.Attr("style"); exists && c.Selectors.ThumbRegex != "" {
				thumbURL := c.ExtractURLFromStyle(style)
				thumbnail, thumbnailLink, _ = c.ProcessImage(thumbURL)
			}
		}
	}

	// Extract posted time
	postedAt := c.processElement(s, "postedAt", c.Selectors.PostedAt)

	return c.CreateDeal(id, title, link, price, thumbnail, thumbnailLink, postedAt), nil
}
