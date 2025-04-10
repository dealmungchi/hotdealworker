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

// fetchWithChromeDB fetches a URL using ChromeDB
func (c *ChromeDBCrawler) fetchWithChromeDB() (io.Reader, error) {
	// Check if the crawler is rate limited
	if c.CacheSvc != nil && c.CacheKey != "" {
		_, err := c.CacheSvc.Get(c.CacheKey)
		if err == nil {
			return nil, fmt.Errorf("%s: %d초 동안 더 이상 요청을 보내지 않음", c.CacheKey, c.BlockTime/time.Second)
		}
	}
	
	fmt.Printf("DEBUG: Fetching URL with ChromeDB: %s\n", c.URL)
	
	// Try direct HTTP request first as a fallback
	fmt.Printf("DEBUG: Trying direct HTTP request first to %s\n", c.URL)
	httpClient := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", c.URL, nil)
	if err == nil {
		// Add browser-like headers to avoid blocking
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		resp, err := httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			bodyBytes, err := io.ReadAll(resp.Body)
			if err == nil && len(bodyBytes) > 0 && (strings.Contains(string(bodyBytes), "<html") || strings.Contains(string(bodyBytes), "<body")) {
				fmt.Printf("DEBUG: Direct HTTP request successful, got %d bytes\n", len(bodyBytes))
				return bytes.NewReader(bodyBytes), nil
			}
		}
	}
	
	// Try a simpler function for better compatibility
	fmt.Printf("DEBUG: Trying simplified function approach\n")
	
	simplePayload := map[string]interface{}{
		"code": `module.exports = async ({ page, context }) => {
			await page.setViewport({ width: 1280, height: 800 });
			await page.setUserAgent('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36');
			
			try {
				await page.goto(context.url, { timeout: 30000 });
				await page.waitForTimeout(2000); // Wait a bit for dynamic content
				return await page.content();
			} catch (e) {
				console.error('Error loading page:', e);
				return '<html><body>Error: ' + e.message + '</body></html>';
			}
		}`,
		"context": map[string]interface{}{
			"url": c.URL,
		},
	}
	
	simpleData, _ := json.Marshal(simplePayload)
	simpleReq, _ := http.NewRequest("POST", c.ChromeAddr+"/function", bytes.NewBuffer(simpleData))
	simpleReq.Header.Set("Content-Type", "application/json")
	
	simpleResp, err := httpClient.Do(simpleReq)
	if err == nil && simpleResp.StatusCode == http.StatusOK {
		defer simpleResp.Body.Close()
		
		simpleBytes, err := io.ReadAll(simpleResp.Body)
		if err == nil && len(simpleBytes) > 0 {
			simpleContent := string(simpleBytes)
			
			// Check if it's JSON response
			if strings.HasPrefix(strings.TrimSpace(simpleContent), "{") {
				var result map[string]interface{}
				if json.Unmarshal(simpleBytes, &result) == nil {
					if data, ok := result["data"].(string); ok && len(data) > 0 {
						simpleContent = data
					} else if data, ok := result["result"].(string); ok && len(data) > 0 {
						simpleContent = data
					}
				}
			}
			
			if strings.Contains(simpleContent, "<html") || strings.Contains(simpleContent, "<body") {
				fmt.Printf("DEBUG: Simple function approach succeeded, got %d bytes\n", len(simpleContent))
				return strings.NewReader(simpleContent), nil
			}
		}
	}
	
	if simpleResp != nil {
		simpleResp.Body.Close()
	}
	
	// If everything fails, try with a more complex approach
	fmt.Printf("DEBUG: Trying function endpoint with more complex settings\n")
	
	functionPayload := map[string]interface{}{
		"code": `
		module.exports = async ({ page, context }) => {
			// Configure the browser for better compatibility
			await page.setUserAgent('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36');
			await page.setExtraHTTPHeaders({
				'Accept-Language': 'en-US,en;q=0.9,ko;q=0.8',
				'Referer': 'https://google.com/'
			});
			
			// Enable JavaScript
			await page.setJavaScriptEnabled(true);
			
			const url = context.url;
			console.log('Navigating to:', url);
			
			try {
				// Try to navigate with longer timeout and different wait options
				const response = await page.goto(url, { 
					waitUntil: 'domcontentloaded', // Try with domcontentloaded instead of networkidle2
					timeout: 45000 
				});
				
				if (!response) {
					throw new Error('Failed to get response from page');
				}
				
				console.log('Page loaded with status:', response.status());
				
				if (response.status() >= 400) {
					throw new Error('Page returned error status: ' + response.status());
				}
				
				// Wait for any key content to load
				await page.waitForTimeout(3000);
				
				// Take a screenshot for debugging
				await page.screenshot({path: '/tmp/debug.png'});
				
				// Get the page content
				const content = await page.content();
				
				if (!content || content.length === 0) {
					throw new Error('Page content is empty');
				}
				
				console.log('Content length:', content.length);
				
				// Return a structured result
				return {
					success: true,
					content: content,
					url: page.url(), // The final URL after redirects
					status: response.status()
				};
			} catch (error) {
				console.error('Navigation error:', error.message);
				
				// Try to get any content that loaded before the error
				try {
					const content = await page.content();
					return {
						success: false,
						error: error.message,
						content: content,
						url: page.url()
					};
				} catch (contentError) {
					return {
						success: false,
						error: error.message + ' (content error: ' + contentError.message + ')'
					};
				}
			}
		}`,
		"context": map[string]interface{}{
			"url": c.URL,
		},
	}
	
	// Convert payload to JSON
	functionData, err := json.Marshal(functionPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal function payload: %w", err)
	}
	
	// Create request to ChromeDB with longer timeout
	client := &http.Client{Timeout: 90 * time.Second}
	functionReq, err := http.NewRequest("POST", c.ChromeAddr+"/function", bytes.NewBuffer(functionData))
	if err != nil {
		return nil, fmt.Errorf("failed to create function request: %w", err)
	}
	
	functionReq.Header.Set("Content-Type", "application/json")
	
	// Send request to ChromeDB
	functionResp, functionErr := client.Do(functionReq)
	
	if functionErr != nil {
		// If request fails, set rate limiting cache
		if c.CacheSvc != nil && c.CacheKey != "" {
			if setErr := c.CacheSvc.Set(c.CacheKey, []byte(fmt.Sprintf("%d", c.BlockTime/time.Second)), c.BlockTime); setErr != nil {
				return nil, fmt.Errorf("failed to set rate limit cache: %w, original error: %w", setErr, functionErr)
			}
		}
		return nil, fmt.Errorf("failed to fetch from ChromeDB function: %w", functionErr)
	}
	
	defer functionResp.Body.Close()
	
	fmt.Printf("DEBUG: Function endpoint response status: %d\n", functionResp.StatusCode)
	
	// Check response status
	if functionResp.StatusCode != http.StatusOK {
		// Set rate limiting cache on error
		if c.CacheSvc != nil && c.CacheKey != "" {
			if setErr := c.CacheSvc.Set(c.CacheKey, []byte(fmt.Sprintf("%d", c.BlockTime/time.Second)), c.BlockTime); setErr != nil {
				return nil, setErr
			}
		}
		return nil, fmt.Errorf("ChromeDB function endpoint returned non-OK status: %d", functionResp.StatusCode)
	}
	
	// Read response body
	bodyBytes, err := io.ReadAll(functionResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ChromeDB function response: %w", err)
	}
	
	fmt.Printf("DEBUG: Received %d bytes from /function endpoint\n", len(bodyBytes))
	
	// If we received no data or empty data, set rate limiting and fail
	if len(bodyBytes) == 0 {
		if c.CacheSvc != nil && c.CacheKey != "" {
			if setErr := c.CacheSvc.Set(c.CacheKey, []byte(fmt.Sprintf("%d", c.BlockTime/time.Second)), c.BlockTime); setErr != nil {
				fmt.Printf("DEBUG: Failed to set rate limit cache: %v\n", setErr)
			}
		}
		return nil, fmt.Errorf("empty response from ChromeDB function endpoint (0 bytes)")
	}
	
	// Format depends on endpoint, may be direct HTML or JSON with result field
	content := string(bodyBytes)
	
	// If it's JSON, try to extract the HTML content from various fields
	if strings.HasPrefix(strings.TrimSpace(content), "{") {
		fmt.Printf("DEBUG: Response appears to be JSON\n")
		var result map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &result); err == nil {
			// Log the structure for debugging
			keys := []string{}
			for k := range result {
				keys = append(keys, k)
			}
			fmt.Printf("DEBUG: JSON response has keys: %v\n", keys)
			
			// Try various possible field names for HTML content
			if data, ok := result["data"].(map[string]interface{}); ok {
				// If data is an object, look for content field
				if htmlContent, ok := data["content"].(string); ok && len(htmlContent) > 0 {
					fmt.Printf("DEBUG: Found HTML content in 'data.content' field (%d bytes)\n", len(htmlContent))
					content = htmlContent
				}
			} else if htmlContent, ok := result["content"].(string); ok && len(htmlContent) > 0 {
				fmt.Printf("DEBUG: Found HTML content in 'content' field (%d bytes)\n", len(htmlContent))
				content = htmlContent
			} else if htmlContent, ok := result["data"].(string); ok && len(htmlContent) > 0 {
				fmt.Printf("DEBUG: Found HTML content in 'data' field (%d bytes)\n", len(htmlContent))
				content = htmlContent
			} else if htmlContent, ok := result["result"].(string); ok && len(htmlContent) > 0 {
				fmt.Printf("DEBUG: Found HTML content in 'result' field (%d bytes)\n", len(htmlContent))
				content = htmlContent
			} else if htmlContent, ok := result["html"].(string); ok && len(htmlContent) > 0 {
				fmt.Printf("DEBUG: Found HTML content in 'html' field (%d bytes)\n", len(htmlContent))
				content = htmlContent
			} else {
				fmt.Printf("DEBUG: Could not find HTML content in JSON response structure\n")
			}
		} else {
			fmt.Printf("DEBUG: Failed to parse JSON: %v\n", err)
		}
	}
	
	// Final validation with better diagnostics
	if !strings.Contains(content, "<html") && !strings.Contains(content, "<body") {
		// Log a snippet of what we did receive to diagnose the issue
		contentPreview := content
		if len(contentPreview) > 200 {
			contentPreview = contentPreview[:200] + "..."
		}
		fmt.Printf("DEBUG: Received non-HTML response: %s\n", contentPreview)
		
		// Set rate limiting cache on error
		if c.CacheSvc != nil && c.CacheKey != "" {
			if setErr := c.CacheSvc.Set(c.CacheKey, []byte(fmt.Sprintf("%d", c.BlockTime/time.Second)), c.BlockTime); setErr != nil {
				fmt.Printf("DEBUG: Failed to set rate limit cache: %v\n", setErr)
			}
		}
		
		return nil, fmt.Errorf("invalid or empty HTML response from ChromeDB (received %d bytes)", len(content))
	}
	
	fmt.Printf("DEBUG: Successfully received HTML content (%d bytes)\n", len(content))
	return strings.NewReader(content), nil
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