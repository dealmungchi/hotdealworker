package crawler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// BaseCrawler provides common functionality for all crawlers
type BaseCrawler struct {
	URL         string
	CacheKey    string
	CacheSvc    cache.CacheService
	BlockTime   time.Duration
	BaseURL     string
	Provider    string
	PriceRegex  string
	IDExtractor IDExtractorFunc
}

// fetchWithCache fetches a URL with caching and rate limiting
func (c *BaseCrawler) fetchWithCache() (io.Reader, error) {
	// Check if the crawler is rate limited
	if c.CacheSvc != nil && c.CacheKey != "" {
		_, err := c.CacheSvc.Get(c.CacheKey)
		if err == nil {
			return nil, fmt.Errorf("%s: %d초 동안 더 이상 요청을 보내지 않음", c.CacheKey, c.BlockTime/time.Second)
		}
	}

	// Fetch the page
	utf8Body, err := helpers.FetchWithRandomHeaders(c.URL)
	if err != nil {
		if c.CacheSvc != nil && c.CacheKey != "" && err.Error() != "" {
			if fmt.Sprintf("%v", err)[:12] == "rate limited" {
				// Set rate limiting cache
				if setErr := c.CacheSvc.Set(c.CacheKey, []byte(fmt.Sprintf("%d", c.BlockTime/time.Second)), c.BlockTime); setErr != nil {
					return nil, setErr
				}
			}
		}
		return nil, err
	}

	return utf8Body, nil
}

// fetchWithChromeDB fetches a URL using ChromeDB
func (c *UnifiedCrawler) fetchWithChromeDB() (io.Reader, error) {
	httpClient := &http.Client{Timeout: 60 * time.Second}

	// Try evaluate endpoint first - most reliable for getting HTML content
	evalPayload := map[string]interface{}{
		"url": c.URL,
		"gotoOptions": map[string]interface{}{
			"waitUntil": "networkidle0", // Wait for all network activity to stop
			"timeout":   45000,
		},
		"evaluate": "document.documentElement.outerHTML",
	}

	evalData, _ := json.Marshal(evalPayload)
	evalReq, err := http.NewRequest("POST", c.ChromeDBAddr+"/evaluate", bytes.NewBuffer(evalData))
	if err == nil {
		evalReq.Header.Set("Content-Type", "application/json")

		evalResp, evalErr := httpClient.Do(evalReq)
		if evalErr == nil && evalResp.StatusCode == http.StatusOK {
			defer evalResp.Body.Close()
			evalBytes, _ := io.ReadAll(evalResp.Body)

			if len(evalBytes) > 0 {
				logger.Debug("Eval endpoint successful, got %d bytes", len(evalBytes))
				var result map[string]interface{}
				if err := json.Unmarshal(evalBytes, &result); err == nil {
					if data, ok := result["data"].(string); ok && len(data) > 0 {
						logger.Debug("Found HTML in evaluate result")
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
			"waitUntil": "networkidle0", // Wait for all network activity to stop
			"timeout":   45000,
		},
	}

	contentData, _ := json.Marshal(contentPayload)
	contentReq, err := http.NewRequest("POST", c.ChromeDBAddr+"/content", bytes.NewBuffer(contentData))
	if err == nil {
		contentReq.Header.Set("Content-Type", "application/json")

		contentResp, contentErr := httpClient.Do(contentReq)
		if contentErr == nil && contentResp.StatusCode == http.StatusOK {
			defer contentResp.Body.Close()
			contentBytes, _ := io.ReadAll(contentResp.Body)

			if len(contentBytes) > 0 {
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
				await page.goto(context.url, { waitUntil: 'networkidle0', timeout: 45000 });
				
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
	funcReq, _ := http.NewRequest("POST", c.ChromeDBAddr+"/function", bytes.NewBuffer(funcData))
	funcReq.Header.Set("Content-Type", "application/json")

	funcResp, funcErr := httpClient.Do(funcReq)
	if funcErr == nil && funcResp.StatusCode == http.StatusOK {
		defer funcResp.Body.Close()
		funcBytes, _ := io.ReadAll(funcResp.Body)

		if len(funcBytes) > 0 {
			logger.Debug("Function endpoint returned %d bytes", len(funcBytes))
			return bytes.NewReader(funcBytes), nil
		}
	}

	// Final fallback - try direct scrape
	scrapeURL := fmt.Sprintf("%s/scrape?url=%s", c.ChromeDBAddr, c.URL)
	scrapeReq, _ := http.NewRequest("GET", scrapeURL, nil)

	scrapeResp, scrapeErr := httpClient.Do(scrapeReq)
	if scrapeErr == nil && scrapeResp.StatusCode == http.StatusOK {
		defer scrapeResp.Body.Close()
		scrapeBytes, _ := io.ReadAll(scrapeResp.Body)

		if len(scrapeBytes) > 0 {
			logger.Debug("Scrape endpoint returned %d bytes", len(scrapeBytes))
			return bytes.NewReader(scrapeBytes), nil
		}
	}

	// If all attempts failed
	if c.CacheSvc != nil && c.CacheKey != "" {
		shortBlockTime := 30 * time.Second
		if setErr := c.CacheSvc.Set(c.CacheKey, []byte("30"), shortBlockTime); setErr != nil {
			logger.Debug("Failed to set rate limit cache: %v", setErr)
		}
	}

	return nil, fmt.Errorf("all fetch attempts failed")
}

// createDocument creates a goquery document from a reader
func (c *BaseCrawler) createDocument(reader io.Reader) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 오류: %v", err)
	}
	return doc, nil
}

// processDeals processes deals in parallel using goroutines
func (c *BaseCrawler) processDeals(selections *goquery.Selection, processor ProcessorFunc) []HotDeal {
	dealChan := make(chan *HotDeal, selections.Length())
	var wg sync.WaitGroup

	selections.Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		go func(s *goquery.Selection) {
			defer wg.Done()

			// Process the deal in the goroutine
			deal, err := processor(s)
			if err != nil {
				// Log the error but continue processing other deals
				// We don't return the error because it would stop all crawling
				// Instead, we just skip this deal
				logger.Error("[%s] Error processing deal: %v", c.Provider, err)
				return
			}

			if deal != nil {
				dealChan <- deal
			}
		}(s)
	})

	wg.Wait()
	close(dealChan)

	// Collect the processed deals
	var deals []HotDeal
	for deal := range dealChan {
		if deal != nil {
			deals = append(deals, *deal)
		}
	}

	return deals
}

// GetName returns the crawler's type name for logging
func (c *BaseCrawler) GetName() string {
	// This will be overridden by concrete implementations
	// But fallback to reflect-based name if not
	if c.Provider != "" {
		return c.Provider + "Crawler"
	}
	return reflect.TypeOf(c).Elem().Name()
}

// GetProvider returns the provider name for the crawler
func (c *BaseCrawler) GetProvider() string {
	return c.Provider
}

// ResolveURL resolves a relative URL against the base URL
func (c *BaseCrawler) ResolveURL(href string) string {
	if href == "" {
		return ""
	}

	// 이미 스킴이 있는 절대 URL
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	// 프로토콜 상대 URL
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	// 스킴 없는 절대 URL (도메인처럼 보이는 경우만)
	if isLikelyDomainURL(href) {
		return "https://" + href
	}

	// 상대 경로 처리
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = c.URL
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		logger.Error("Error parsing base URL '%s': %v", baseURL, err)
		return href
	}

	ref, err := url.Parse(href)
	if err != nil {
		logger.Error("Error parsing href '%s': %v", href, err)
		return href
	}

	return base.ResolveReference(ref).String()
}

// ExtractPrice extracts the price from a title using the configured regex
func (c *BaseCrawler) ExtractPrice(title string) (string, string) {
	if c.PriceRegex == "" || title == "" {
		return title, ""
	}

	re := regexp.MustCompile(c.PriceRegex)
	if match := re.FindStringSubmatch(title); len(match) > 1 {
		price := strings.TrimSpace(match[1])
		return title, price
	}

	return title, ""
}

// ProcessImage fetches an image and converts it to base64
func (c *BaseCrawler) ProcessImage(imageURL string) (string, string, error) {
	imageURL = c.ResolveURL(imageURL)
	if imageURL == "" {
		return "", "", nil
	}

	data, err := helpers.FetchSimply(imageURL)
	if err != nil {
		logger.Error("Error fetching image: %v", err)
		return "", "", nil
	}

	return base64.StdEncoding.EncodeToString(data), imageURL, nil
}

// CreateDeal creates a HotDeal with the given properties
func (c *BaseCrawler) CreateDeal(id, title, link, price, thumbnail, thumbnailLink, postedAt string) *HotDeal {
	return &HotDeal{
		Id:            id,
		Title:         title,
		Link:          link,
		Price:         price,
		Thumbnail:     thumbnail,
		ThumbnailLink: thumbnailLink,
		PostedAt:      postedAt,
		Provider:      c.Provider,
	}
}

// ExtractURLFromStyle extracts a URL from a CSS style attribute
func (c *BaseCrawler) ExtractURLFromStyle(style string) string {
	re := regexp.MustCompile(`url\((?:['"]?)(.*?)(?:['"]?)\)`)
	if matches := re.FindStringSubmatch(style); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func isLikelyDomainURL(href string) bool {
	domainLike := regexp.MustCompile(`^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/|$)`)
	return domainLike.MatchString(href)
}

// Debug logs a debug message
func (b *BaseCrawler) Debug(format string, v ...interface{}) {
	logger.Debug("[%s] "+format, append([]interface{}{b.GetName()}, v...)...)
}
