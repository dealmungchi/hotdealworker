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

// ChromeDBStrategy represents different strategies for fetching content
type ChromeDBStrategy struct {
	Name        string
	Endpoint    string
	Payload     map[string]interface{}
	Method      string
	ProcessFunc func([]byte) (io.Reader, error)
}

// ============================================================================
// FETCH METHODS
// ============================================================================

// fetchWithCache fetches a URL with caching and rate limiting
func (c *BaseCrawler) fetchWithCache() (io.Reader, error) {
	// Check rate limiting
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

// fetchWithChromeDB fetches a URL using FlareSolverr first, falling back to ChromeDB if needed
func (c *UnifiedCrawler) fetchWithChromeDB() (io.Reader, error) {
	// Step 1: Try FlareSolverr first for Cloudflare-protected sites
	if err := c.checkFlareSolverr(); err == nil {
		logger.Debug("[%s] FlareSolverr available, attempting Cloudflare bypass", c.Provider)
		reader, err := c.fetchWithFlareSolverr()
		if err == nil && reader != nil {
			logger.Info("[%s] FlareSolverr bypass successful", c.Provider)
			return reader, nil
		}
		logger.Warn("[%s] FlareSolverr failed: %v, falling back to ChromeDB", c.Provider, err)
	} else {
		logger.Debug("[%s] FlareSolverr not available: %v", c.Provider, err)
	}

	// Step 2: Fallback to ChromeDB
	if err := c.checkChromeDBHealth(); err != nil {
		logger.Error("[%s] ChromeDB health check failed: %v", c.Provider, err)
		return nil, fmt.Errorf("both FlareSolverr and ChromeDB unavailable")
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}

	// ChromeDB strategies (only the working ones)
	strategies := []ChromeDBStrategy{
		// Strategy 1: Network idle (best for dynamic content)
		{
			Name:     "networkidle-content",
			Endpoint: "/content",
			Method:   "POST",
			Payload: map[string]interface{}{
				"url": c.URL,
				"gotoOptions": map[string]interface{}{
					"waitUntil": "networkidle0",
					"timeout":   45000,
				},
			},
			ProcessFunc: c.processRawResponse,
		},

		// Strategy 2: Basic load (faster, works for static content)
		{
			Name:     "basic-content",
			Endpoint: "/content",
			Method:   "POST",
			Payload: map[string]interface{}{
				"url": c.URL,
				"gotoOptions": map[string]interface{}{
					"waitUntil": "load",
					"timeout":   20000,
				},
			},
			ProcessFunc: c.processRawResponse,
		},

		// Strategy 3: Simple scrape (last resort)
		{
			Name:        "scrape-fallback",
			Endpoint:    "/scrape",
			Method:      "GET",
			Payload:     nil,
			ProcessFunc: c.processRawResponse,
		},
	}

	// Try each ChromeDB strategy
	for i, strategy := range strategies {
		logger.Debug("[%s] Trying ChromeDB strategy %d/%d: %s", c.Provider, i+1, len(strategies), strategy.Name)

		reader, err := c.executeStrategy(httpClient, strategy)
		if err == nil && reader != nil {
			logger.Info("[%s] ChromeDB strategy %s succeeded", c.Provider, strategy.Name)
			return reader, nil
		}

		logger.Debug("[%s] ChromeDB strategy %s failed: %v", c.Provider, strategy.Name, err)

		// Brief delay between attempts
		if i < len(strategies)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	// Set rate limit if all strategies failed
	if c.CacheSvc != nil && c.CacheKey != "" {
		blockTime := 60 * time.Second
		if setErr := c.CacheSvc.Set(c.CacheKey, []byte("60"), blockTime); setErr != nil {
			logger.Debug("[%s] Failed to set rate limit cache: %v", c.Provider, setErr)
		}
	}

	return nil, fmt.Errorf("all fetch strategies failed for URL: %s", c.URL)
}

// ============================================================================
// FLARESOLVERR METHODS
// ============================================================================

// checkFlareSolverr checks if FlareSolverr is available
func (c *UnifiedCrawler) checkFlareSolverr() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8191")
	if err != nil {
		return fmt.Errorf("FlareSolverr not available: %v", err)
	}
	defer resp.Body.Close()
	return nil
}

// fetchWithFlareSolverr fetches URL using FlareSolverr to bypass Cloudflare
func (c *UnifiedCrawler) fetchWithFlareSolverr() (io.Reader, error) {
	client := &http.Client{Timeout: 120 * time.Second}

	// First try without proxy
	payload := map[string]interface{}{
		"cmd":        "request.get",
		"url":        c.URL,
		"maxTimeout": 10000,
	}

	reader, err := c.executeFlareSolverrRequest(client, payload)
	if err == nil {
		return reader, nil
	}

	logger.Warn("[%s] FlareSolverr failed without proxy: %v, retrying with proxy", c.Provider, err)

	// Retry with proxy
	payload["proxy"] = map[string]interface{}{
		// TODO: proxy setup
		"url": "socks5://3.15.212.96:1080",
	}

	return c.executeFlareSolverrRequest(client, payload)
}

// executeFlareSolverrRequest executes a single FlareSolverr request
func (c *UnifiedCrawler) executeFlareSolverrRequest(client *http.Client, payload map[string]interface{}) (io.Reader, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Content-Type":    "application/json",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
	}

	req, err := http.NewRequest("POST", "http://localhost:8191/v1", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the entire response body into memory
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the FlareSolverr response
	var flareResp struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		Solution struct {
			Response string                   `json:"response"`
			Cookies  []map[string]interface{} `json:"cookies"`
		} `json:"solution"`
	}

	if err := json.Unmarshal(body, &flareResp); err != nil {
		return nil, fmt.Errorf("failed to parse FlareSolverr response: %v", err)
	}

	if flareResp.Status != "ok" {
		return nil, fmt.Errorf("FlareSolverr error: %s", flareResp.Message)
	}

	// Use the response content from the solution
	if flareResp.Solution.Response == "" {
		return nil, fmt.Errorf("no content in FlareSolverr response")
	}

	// Log the response for debugging
	logger.Debug("[%s] FlareSolverr response status: %s, message: %s", c.Provider, flareResp.Status, flareResp.Message)
	logger.Debug("[%s] FlareSolverr response size: %d bytes", c.Provider, len(flareResp.Solution.Response))

	return bytes.NewReader([]byte(flareResp.Solution.Response)), nil
}

// ============================================================================
// CHROMEDB METHODS
// ============================================================================

// checkChromeDBHealth checks if ChromeDB is available
func (c *UnifiedCrawler) checkChromeDBHealth() error {
	if c.ChromeDBAddr == "" {
		return fmt.Errorf("ChromeDB address not configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(c.ChromeDBAddr + "/")
	if err != nil {
		return fmt.Errorf("ChromeDB server not reachable at %s: %v", c.ChromeDBAddr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("ChromeDB server error (status %d)", resp.StatusCode)
	}

	logger.Debug("[%s] ChromeDB health check passed (status %d)", c.Provider, resp.StatusCode)
	return nil
}

// executeStrategy executes a single ChromeDB strategy
func (c *UnifiedCrawler) executeStrategy(client *http.Client, strategy ChromeDBStrategy) (io.Reader, error) {
	var req *http.Request
	var err error

	if strategy.Method == "POST" && strategy.Payload != nil {
		data, marshalErr := json.Marshal(strategy.Payload)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", marshalErr)
		}

		req, err = http.NewRequest("POST", c.ChromeDBAddr+strategy.Endpoint, bytes.NewBuffer(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "HotDealWorker/1.0")

	} else if strategy.Method == "GET" {
		if strategy.Endpoint == "/scrape" {
			req, err = http.NewRequest("GET", fmt.Sprintf("%s/scrape?url=%s", c.ChromeDBAddr, url.QueryEscape(c.URL)), nil)
		} else {
			req, err = http.NewRequest("GET", c.ChromeDBAddr+strategy.Endpoint, nil)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create GET request: %v", err)
		}
		req.Header.Set("User-Agent", "HotDealWorker/1.0")

	} else {
		return nil, fmt.Errorf("unsupported method %s or missing payload", strategy.Method)
	}

	logger.Debug("[%s] Making %s request to %s", c.Provider, strategy.Method, req.URL.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	logger.Debug("[%s] Response status: %d", c.Provider, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		// Try to read response body for more details
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 && len(body) < 500 {
			logger.Debug("[%s] Error response body: %s", c.Provider, string(body))
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	logger.Debug("[%s] Response size: %d bytes", c.Provider, len(responseBytes))

	if len(responseBytes) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return strategy.ProcessFunc(responseBytes)
}

// ============================================================================
// RESPONSE PROCESSORS
// ============================================================================

// processRawResponse processes raw response data
func (c *UnifiedCrawler) processRawResponse(data []byte) (io.Reader, error) {
	if len(data) < 50 {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	// Check if it looks like HTML
	dataStr := string(data)
	if strings.Contains(strings.ToLower(dataStr), "<html") ||
		strings.Contains(strings.ToLower(dataStr), "<!doctype") ||
		strings.Contains(strings.ToLower(dataStr), "<body") {
		logger.Debug("[%s] Response appears to be HTML: %d bytes", c.Provider, len(data))
		return bytes.NewReader(data), nil
	}

	// Log preview for debugging
	preview := dataStr
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	logger.Debug("[%s] Response doesn't look like HTML. Preview: %s", c.Provider, preview)

	return nil, fmt.Errorf("response doesn't appear to be valid HTML")
}

// ============================================================================
// DOCUMENT AND DEAL PROCESSING
// ============================================================================

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

			deal, err := processor(s)
			if err != nil {
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

// ============================================================================
// UTILITY METHODS
// ============================================================================

// GetName returns the crawler's type name for logging
func (c *BaseCrawler) GetName() string {
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

	var data []byte
	var err error
	if c.Provider == "Bbasak" {
		data, err = helpers.FetchSimply(imageURL, http.Header{
			"Referer": []string{"https://bbasak.com/bbs/board.php?bo_table=bbasak1"},
		})
	} else {
		data, err = helpers.FetchSimply(imageURL)
	}

	if err != nil {
		logger.Warn("Error fetching image: %v", err)
		return "", "", nil
	}

	return base64.StdEncoding.EncodeToString(data), imageURL, nil
}

// CreateDeal creates a HotDeal with the given properties
func (c *BaseCrawler) CreateDeal(id, title, link, price, thumbnail, thumbnailLink, postedAt, category string) *HotDeal {
	return &HotDeal{
		Id:            id,
		Title:         title,
		Link:          link,
		Price:         price,
		Thumbnail:     thumbnail,
		ThumbnailLink: thumbnailLink,
		PostedAt:      postedAt,
		Category:      category,
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

// Debug logs a debug message
func (b *BaseCrawler) Debug(format string, v ...interface{}) {
	logger.Debug("[%s] "+format, append([]interface{}{b.GetName()}, v...)...)
}

// isLikelyDomainURL checks if a string looks like a domain URL
func isLikelyDomainURL(href string) bool {
	domainLike := regexp.MustCompile(`^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/|$)`)
	return domainLike.MatchString(href)
}

// ============================================================================
// CATEGORY CLASSIFICATION
// ============================================================================

// classifyCategory classifies deal categories into standardized groups
func classifyCategory(category string) string {
	if category == "" {
		return "기타"
	}

	mapping := map[string][]string{
		"전자제품/디지털/PC/하드웨어": {
			"PC/하드웨어", "PC관련", "컴퓨터", "디지털", "PC제품", "전자제품", "가전제품", "가전",
			"모바일", "노트북/모바일", "휴대폰", "A/V", "VR", "게임H/W", "PC 하드웨어", "모바일 / 가젯",
		},
		"소프트웨어/게임": {
			"SW/게임", "게임", "게임/SW", "게임S/W", "게임 / SW",
		},
		"생활용품/인테리어/주방": {
			"생활용품", "인테리어", "주방용품", "생활/식품", "가구인테리어",
		},
		"식품/먹거리": {
			"식품", "음식", "먹거리", "식품/건강", "식품/식당",
		},
		"의류/패션/잡화": {
			"의류", "의류/잡화", "패션/의류", "패션소품", "잡화", "신발", "가방/지갑",
			"명품", "시계/쥬얼리", "패션잡화",
		},
		"화장품/뷰티": {
			"화장품", "뷰티/미용", "화장품/바디",
		},
		"도서/미디어/콘텐츠": {
			"도서", "서적", "도서/미디어",
		},
		"카메라/사진": {
			"카메라", "카메라/사진",
		},
		"상품권/쿠폰/포인트": {
			"상품권/쿠폰", "모바일/상품권", "쿠폰", "포인트/래플", "패키지/이용권",
		},
		"출산/육아": {
			"육아", "출산육아", "육아용품",
		},
		"반려동물": {
			"애완용품", "반려동물용품",
		},
		"스포츠/아웃도어/레저": {
			"등산/캠핑", "스포츠용품", "레저용품",
		},
		"건강/비타민": {
			"비타민/의약", "영양제",
		},
		"여행/서비스": {
			"여행", "여행/서비스",
		},
		"이벤트/응모/바이럴": {
			"이벤트", "응모", "바이럴",
		},
		"학용품/사무용품": {
			"학용/사무용품",
		},
	}

	// 매핑 확인
	for bigCategory, keywords := range mapping {
		for _, keyword := range keywords {
			if strings.Contains(category, keyword) {
				return bigCategory
			}
		}
	}

	return "기타"
}
