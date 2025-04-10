package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// ChromeDBCrawler implements a crawler using ChromeDB
type ChromeDBCrawler struct {
	BaseCrawler
	Selectors           Selectors
	CustomHandlers      CustomHandlers
	ElementTransformers ElementTransformers
	// ChromeDB specific fields
	UseChrome  bool
	ChromeAddr string
}

// NewChromeDBCrawler creates a new ChromeDB-based crawler
func NewChromeDBCrawler(config CrawlerConfig, cacheSvc cache.CacheService, chromeAddr string) *ChromeDBCrawler {
	chromeDBCrawler := &ChromeDBCrawler{
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
		UseChrome:           true,
		ChromeAddr:          chromeAddr,
	}

	// Check ChromeDB connection on initialization
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(chromeAddr)
	if err != nil {
		fmt.Printf("Warning: ChromeDB connection check failed: %v\n", err)
		fmt.Printf("Check if ChromeDB is running at: %s\n", chromeAddr)
	} else {
		fmt.Printf("ChromeDB connection successful: %s, status: %d\n", chromeAddr, resp.StatusCode)
		resp.Body.Close()
	}

	return chromeDBCrawler
}

// fetchWithChromeDB fetches a URL using ChromeDB - optimized for FMKorea
func (c *ChromeDBCrawler) fetchWithChromeDB() (io.Reader, error) {
	// Check if the crawler is rate limited
	if c.CacheSvc != nil && c.CacheKey != "" {
		_, err := c.CacheSvc.Get(c.CacheKey)
		if err == nil {
			return nil, fmt.Errorf("%s: %d초 동안 더 이상 요청을 보내지 않음", c.CacheKey, c.BlockTime/time.Second)
		}
	}

	fmt.Printf("DEBUG: Fetching URL with ChromeDB: %s\n", c.URL)

	// FMKorea needs ChromeDB, direct HTTP requests will be rate limited
	httpClient := &http.Client{Timeout: 60 * time.Second}

	// Try evaluate endpoint first - most reliable for getting HTML content
	evalPayload := map[string]interface{}{
		"url": c.URL,
		"gotoOptions": map[string]interface{}{
			"waitUntil": "domcontentloaded",
			"timeout":   10000,
		},
		"evaluate": "document.documentElement.outerHTML",
	}

	evalData, _ := json.Marshal(evalPayload)
	evalReq, err := http.NewRequest("POST", c.ChromeAddr+"/evaluate", bytes.NewBuffer(evalData))
	if err == nil {
		evalReq.Header.Set("Content-Type", "application/json")

		evalResp, evalErr := httpClient.Do(evalReq)
		if evalErr == nil && evalResp.StatusCode == http.StatusOK {
			defer evalResp.Body.Close()
			evalBytes, _ := io.ReadAll(evalResp.Body)

			if len(evalBytes) > 0 {
				fmt.Printf("DEBUG: Eval endpoint successful, got %d bytes\n", len(evalBytes))
				var result map[string]interface{}
				if err := json.Unmarshal(evalBytes, &result); err == nil {
					if data, ok := result["data"].(string); ok && len(data) > 0 {
						fmt.Printf("DEBUG: Found HTML in evaluate result\n")
						return strings.NewReader(data), nil
					}
				}
				// If we couldn't extract HTML from JSON, return raw content
				return bytes.NewReader(evalBytes), nil
			}
		}
	}

	// If evaluate endpoint failed, try content endpoint
	contentPayload := map[string]interface{}{
		"url": c.URL,
		"gotoOptions": map[string]interface{}{
			"waitUntil": "domcontentloaded",
			"timeout":   45000,
		},
	}

	contentData, _ := json.Marshal(contentPayload)
	contentReq, err := http.NewRequest("POST", c.ChromeAddr+"/content", bytes.NewBuffer(contentData))
	if err == nil {
		contentReq.Header.Set("Content-Type", "application/json")

		contentResp, contentErr := httpClient.Do(contentReq)
		if contentErr == nil && contentResp.StatusCode == http.StatusOK {
			defer contentResp.Body.Close()
			contentBytes, _ := io.ReadAll(contentResp.Body)

			if len(contentBytes) > 0 {
				fmt.Printf("DEBUG: Content endpoint successful, got %d bytes\n", len(contentBytes))
				return bytes.NewReader(contentBytes), nil
			}
		}
	}

	// If all direct endpoints failed, try custom function
	funcPayload := map[string]interface{}{
		"code": `
		module.exports = async ({ page, context }) => {
			try {
				console.log("Using direct HTML extraction method");
				
				// Set up basic browser configuration
				await page.setViewport({ width: 1280, height: 800 });
				await page.setUserAgent('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36');
				await page.setExtraHTTPHeaders({
					'Accept-Language': 'ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7',
					'Referer': 'https://google.co.kr/',
					'Cache-Control': 'no-cache',
					'Pragma': 'no-cache',
					'Sec-Ch-Ua': '"Google Chrome";v="119", "Chromium";v="119", "Not?A_Brand";v="24"',
					'Sec-Ch-Ua-Mobile': '?0',
					'Sec-Ch-Ua-Platform': '"Windows"',
				});
				
				// Navigate to the URL
				await page.goto(context.url, { waitUntil: 'domcontentloaded', timeout: 45000 });
				await page.waitForTimeout(5000);
				
				// Scroll to trigger lazy loading
				await page.evaluate(() => window.scrollBy(0, 500));
				await page.waitForTimeout(2000);
				
				// Get HTML directly using DOM API 
				const html = await page.evaluate(() => document.documentElement.outerHTML);
				console.log("HTML length from evaluate:", html.length);
				
				// Return the HTML as a string, not wrapped in an object
				return html;
			} catch (e) {
				console.error("Error:", e);
				return "<html><body>Error: " + e.message + "</body></html>";
			}
		}`,
		"context": map[string]interface{}{
			"url": c.URL,
		},
	}

	funcData, _ := json.Marshal(funcPayload)
	funcReq, _ := http.NewRequest("POST", c.ChromeAddr+"/function", bytes.NewBuffer(funcData))
	funcReq.Header.Set("Content-Type", "application/json")

	funcResp, funcErr := httpClient.Do(funcReq)
	if funcErr == nil && funcResp.StatusCode == http.StatusOK {
		defer funcResp.Body.Close()
		funcBytes, _ := io.ReadAll(funcResp.Body)

		if len(funcBytes) > 0 {
			fmt.Printf("DEBUG: Function endpoint returned %d bytes\n", len(funcBytes))
			return bytes.NewReader(funcBytes), nil
		}
	}

	// Final fallback - try direct scrape
	scrapeURL := fmt.Sprintf("%s/scrape?url=%s", c.ChromeAddr, c.URL)
	scrapeReq, _ := http.NewRequest("GET", scrapeURL, nil)

	scrapeResp, scrapeErr := httpClient.Do(scrapeReq)
	if scrapeErr == nil && scrapeResp.StatusCode == http.StatusOK {
		defer scrapeResp.Body.Close()
		scrapeBytes, _ := io.ReadAll(scrapeResp.Body)

		if len(scrapeBytes) > 0 {
			fmt.Printf("DEBUG: Scrape endpoint returned %d bytes\n", len(scrapeBytes))
			return bytes.NewReader(scrapeBytes), nil
		}
	}

	// If all attempts failed
	if c.CacheSvc != nil && c.CacheKey != "" {
		shortBlockTime := 30 * time.Second
		if setErr := c.CacheSvc.Set(c.CacheKey, []byte("30"), shortBlockTime); setErr != nil {
			fmt.Printf("DEBUG: Failed to set rate limit cache: %v\n", setErr)
		}
	}

	return nil, fmt.Errorf("all fetch attempts failed")
}

// FetchDeals fetches deals using ChromeDB
func (c *ChromeDBCrawler) FetchDeals() ([]HotDeal, error) {
	fmt.Printf("DEBUG: Starting ChromeDBCrawler.FetchDeals() for %s\n", c.Provider)

	// Fetch the page with ChromeDB
	var utf8Body io.Reader
	var err error

	utf8Body, err = c.fetchWithChromeDB()
	if err != nil {
		return nil, err
	}

	// Parse the HTML document
	doc, err := c.createDocument(utf8Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find all deal items
	dealSelections := doc.Find(c.Selectors.DealList)
	fmt.Printf("DEBUG: Found %d deal items with selector: %s\n", dealSelections.Length(), c.Selectors.DealList)

	// Process deals (reuse existing processing logic)
	deals := c.processDeals(dealSelections, c.processDeal)
	fmt.Printf("DEBUG: Processed %d deals successfully\n", len(deals))

	return deals, nil
}

// cleanSelection removes specified elements from a selection before getting text
func (c *ChromeDBCrawler) cleanSelection(sel *goquery.Selection, path string) *goquery.Selection {
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
func (c *ChromeDBCrawler) processElement(s *goquery.Selection, path string, selector string) string {
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

// processDeal processes a single deal based on the configuration (reused from ConfigurableCrawler)
func (c *ChromeDBCrawler) processDeal(s *goquery.Selection) (*HotDeal, error) {
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
