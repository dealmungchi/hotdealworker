package crawler

import (
	"fmt"
	"strings"
	"time"

	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// ConfigurableCrawler is a crawler that can be configured with selectors
type ConfigurableCrawler struct {
	BaseCrawler
	Selectors Selectors
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
		Selectors: config.Selectors,
	}
}

// FetchDeals fetches deals based on the configuration
func (c *ConfigurableCrawler) FetchDeals() ([]HotDeal, error) {
	fmt.Printf("[%s] Starting to fetch deals from %s\n", c.GetName(), c.URL)

	// Fetch the page with rate limiting
	utf8Body, err := c.fetchWithCache()
	if err != nil {
		fmt.Printf("[%s] Error fetching page: %v\n", c.GetName(), err)
		return nil, err
	}

	// Parse the HTML document
	doc, err := c.createDocument(utf8Body)
	if err != nil {
		fmt.Printf("[%s] Error parsing HTML: %v\n", c.GetName(), err)
		return nil, err
	}

	// Find all deal items
	dealSelections := doc.Find(c.Selectors.DealList)
	fmt.Printf("[%s] Found %d potential deal items\n", c.GetName(), dealSelections.Length())

	deals := c.processDeals(dealSelections, c.processDeal)
	fmt.Printf("[%s] Successfully processed %d deals\n", c.GetName(), len(deals))

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
	for _, removal := range c.Selectors.RemoveElements {
		if removal.ApplyToPath == path {
			clone.Find(removal.Selector).Remove()
		}
	}
	
	return clone
}

// processDeal processes a single deal based on the configuration
func (c *ConfigurableCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
	// Skip if the element has a class to filter out
	if c.Selectors.ClassFilter != "" && s.HasClass(c.Selectors.ClassFilter) {
		fmt.Printf("[%s] Skipping element with filtered class: %s\n", c.Provider, c.Selectors.ClassFilter)
		return nil, nil
	}

	// Extract title
	var title string
	titleSel := s.Find(c.Selectors.Title)
	if titleSel.Length() == 0 {
		fmt.Printf("[%s] Title selector not found: %s\n", c.Provider, c.Selectors.Title)
		return nil, nil
	}
	
	// Clean the title selection if needed
	cleanTitleSel := c.cleanSelection(titleSel, "title")
	
	if titleAttr, exists := cleanTitleSel.Attr("title"); exists && titleAttr != "" {
		title = titleAttr
	} else {
		title = cleanTitleSel.Text()
	}
	title = strings.TrimSpace(title)
	if title == "" {
		fmt.Printf("[%s] Title is empty\n", c.Provider)
		return nil, nil
	}
	fmt.Printf("[%s] Found title: %s\n", c.Provider, title)

	// Extract link
	linkSel := s.Find(c.Selectors.Link)
	if linkSel.Length() == 0 {
		fmt.Printf("[%s] Link selector not found: %s\n", c.Provider, c.Selectors.Link)
		return nil, nil
	}
	
	link, exists := linkSel.Attr("href")
	if !exists || strings.TrimSpace(link) == "" {
		fmt.Printf("[%s] Link href not found\n", c.Provider)
		return nil, nil
	}
	link = c.ResolveURL(strings.TrimSpace(link))
	fmt.Printf("[%s] Found link: %s\n", c.Provider, link)

	// Extract ID from the link
	var id string
	var err error
	if c.IDExtractor != nil {
		id, err = c.IDExtractor(link)
		if err != nil {
			fmt.Printf("[%s] Error extracting ID from link: %v\n", c.Provider, err)
			return nil, err
		}
		fmt.Printf("[%s] Extracted ID: %s\n", c.Provider, id)
	}

	// Extract price from title if regex is set
	var price string
	if c.PriceRegex != "" {
		oldTitle := title
		title, price = c.ExtractPrice(title)
		if price != "" {
			fmt.Printf("[%s] Extracted price: %s from title: %s\n", c.Provider, price, oldTitle)
		}
	}

	// Extract thumbnail
	var thumbnail string
	if c.Selectors.Thumbnail != "" {
		thumbSel := s.Find(c.Selectors.Thumbnail)
		if thumbSel.Length() == 0 {
			fmt.Printf("[%s] Thumbnail selector not found: %s\n", c.Provider, c.Selectors.Thumbnail)
		} else {
			if src, exists := thumbSel.Attr("src"); exists {
				fmt.Printf("[%s] Found thumbnail src: %s\n", c.Provider, src)
				thumbnail, _ = c.ProcessImage(src)
			} else if style, exists := thumbSel.Attr("style"); exists && c.Selectors.ThumbRegex != "" {
				thumbURL := c.ExtractURLFromStyle(style)
				fmt.Printf("[%s] Extracted thumbnail from style: %s\n", c.Provider, thumbURL)
				thumbnail, _ = c.ProcessImage(thumbURL)
			} else {
				fmt.Printf("[%s] No thumbnail attributes found\n", c.Provider)
			}
		}
	}

	// Extract posted time
	var postedAt string
	
	// Use custom handler if defined, otherwise use default handling
	if c.Selectors.PostedAtHandler != nil {
		postedAt = c.Selectors.PostedAtHandler(s)
	} else {
		postedAtSel := s.Find(c.Selectors.PostedAt)
		if postedAtSel.Length() == 0 {
			fmt.Printf("[%s] PostedAt selector not found: %s\n", c.Provider, c.Selectors.PostedAt)
		}
		
		// Clean the posted time selection if needed
		cleanPostedAtSel := c.cleanSelection(postedAtSel, "postedAt")
		postedAt = strings.TrimSpace(cleanPostedAtSel.Text())
	}
	
	if postedAt != "" {
		fmt.Printf("[%s] Found posted time: %s\n", c.Provider, postedAt)
	}

	fmt.Printf("[%s] Successfully processed deal: %s\n", c.Provider, title)
	return c.CreateDeal(id, title, link, price, thumbnail, postedAt), nil
}

