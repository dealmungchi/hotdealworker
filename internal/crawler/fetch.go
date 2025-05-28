package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/services/cache"
)

// ProxyInfo holds proxy information with latency
type ProxyInfo struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Type     string        `json:"type"`
	Country  string        `json:"country"`
	Latency  time.Duration `json:"latency"`
	LastTest time.Time     `json:"last_test"`
	Working  bool          `json:"working"`
}

// ProxyManager manages SOCKS5 proxies from spys.one
type ProxyManager struct {
	proxies        []ProxyInfo
	mutex          sync.RWMutex
	lastUpdate     time.Time
	updateInterval time.Duration
}

// NewProxyManager creates a new proxy manager
func NewProxyManager() *ProxyManager {
	return &ProxyManager{
		proxies:        make([]ProxyInfo, 0),
		updateInterval: 30 * time.Minute, // Update every 30 minutes
	}
}

// fetchProxiesFromSpysOne fetches SOCKS5 proxies from spys.me
func (pm *ProxyManager) fetchProxiesFromSpysOne() ([]ProxyInfo, error) {
	logger.Debug("Fetching SOCKS5 proxies from spys.me")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with proper headers
	req, err := http.NewRequest("GET", "https://spys.me/socks.txt", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/plain,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spys.me: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("spys.me returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var proxies []ProxyInfo

	// Parse text format - each line contains IP:PORT CountryCode-Anonymity(Noa/Anm/Hia)-SSL_support(S)-Google_passed(+)
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "Proxy list") || strings.Contains(line, "Http proxy") || strings.Contains(line, "Support by") || strings.Contains(line, "BTC") || strings.Contains(line, "IP address:Port") || strings.Contains(line, "Free SOCKS5") {
			continue // Skip empty lines, comments, and header lines
		}

		// Split line by space to separate IP:PORT from metadata
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}

		// Extract IP:PORT from first field
		ipPortField := fields[0]
		parts := strings.Split(ipPortField, ":")
		if len(parts) != 2 {
			continue // Skip invalid format
		}

		host := strings.TrimSpace(parts[0])
		portStr := strings.TrimSpace(parts[1])

		// Validate IP format
		if net.ParseIP(host) == nil {
			continue // Skip invalid IP
		}

		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			continue // Skip invalid port
		}

		// Extract country code from second field if available
		country := "Unknown"
		if len(fields) >= 2 {
			countryField := fields[1]
			// Format is usually like "US-H", "RU-H!", etc.
			if parts := strings.Split(countryField, "-"); len(parts) >= 1 {
				country = parts[0]
			}
		}

		proxy := ProxyInfo{
			Host:    host,
			Port:    port,
			Type:    "socks5",
			Country: country,
			Working: false, // Will be tested
		}

		proxies = append(proxies, proxy)
	}

	logger.Info("Found %d SOCKS5 proxies from spys.me", len(proxies))
	return proxies, nil
}

// testProxyLatency tests the latency of a single proxy
func (pm *ProxyManager) testProxyLatency(proxy *ProxyInfo) {
	// Create SOCKS5 dialer with shorter timeout for faster testing
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", proxy.Host, proxy.Port), 5*time.Second)
	if err != nil {
		proxy.Working = false
		proxy.Latency = time.Hour // Very high latency for failed connections
		return
	}
	defer conn.Close()

	// Additional HTTP test through proxy with shorter timeout
	proxyURL := fmt.Sprintf("socks5://%s:%d", proxy.Host, proxy.Port)
	transport := &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	// Test with a simple HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/ip", nil)
	if err != nil {
		proxy.Working = false
		proxy.Latency = time.Hour
		return
	}

	testStart := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		proxy.Working = false
		proxy.Latency = time.Hour
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		proxy.Working = true
		proxy.Latency = time.Since(testStart)
		proxy.LastTest = time.Now()
		logger.Debug("Proxy %s:%d (%s) working, latency: %v", proxy.Host, proxy.Port, proxy.Country, proxy.Latency)
	} else {
		proxy.Working = false
		proxy.Latency = time.Hour
	}
}

// UpdateProxies fetches and tests proxies
func (pm *ProxyManager) UpdateProxies() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if we need to update
	if time.Since(pm.lastUpdate) < pm.updateInterval && len(pm.proxies) > 0 {
		return nil
	}

	logger.Info("Updating proxy list from spys.one")

	// Fetch new proxies
	newProxies, err := pm.fetchProxiesFromSpysOne()
	if err != nil {
		return fmt.Errorf("failed to fetch proxies: %v", err)
	}

	if len(newProxies) == 0 {
		return fmt.Errorf("no proxies found")
	}

	// Test proxies in batches to find top 5 quickly
	var workingProxies []ProxyInfo
	var mu sync.Mutex
	batchSize := 50  // Test 50 proxies at a time
	targetCount := 5 // Stop when we have 5 good proxies

	logger.Info("Testing proxies in batches of %d to find top %d...", batchSize, targetCount)

	for i := 0; i < len(newProxies) && len(workingProxies) < targetCount*2; i += batchSize {
		end := i + batchSize
		if end > len(newProxies) {
			end = len(newProxies)
		}

		batch := newProxies[i:end]
		logger.Debug("Testing batch %d-%d (%d proxies)...", i+1, end, len(batch))

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 20)

		for j := range batch {
			wg.Add(1)
			go func(proxy *ProxyInfo) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				pm.testProxyLatency(proxy)
				if proxy.Working {
					mu.Lock()
					workingProxies = append(workingProxies, *proxy)
					mu.Unlock()
				}
			}(&batch[j])
		}

		wg.Wait()

		// Sort current working proxies by latency
		sort.Slice(workingProxies, func(i, j int) bool {
			return workingProxies[i].Latency < workingProxies[j].Latency
		})

		logger.Info("Batch complete. Found %d working proxies so far", len(workingProxies))

		// If we have enough good proxies, we can stop early
		if len(workingProxies) >= targetCount*2 {
			logger.Info("Found enough proxies (%d), stopping early", len(workingProxies))
			break
		}
	}

	// Keep only top 5 fastest proxies
	if len(workingProxies) > targetCount {
		workingProxies = workingProxies[:targetCount]
		logger.Info("Selected top %d fastest proxies", targetCount)
	}

	pm.proxies = workingProxies
	pm.lastUpdate = time.Now()

	logger.Info("Updated proxy list: %d proxies selected (fastest: %v)",
		len(workingProxies),
		func() string {
			if len(workingProxies) > 0 {
				return workingProxies[0].Latency.String()
			}
			return "none"
		}())

	// Log top proxies for debugging
	for i, proxy := range workingProxies {
		logger.Info("Top proxy #%d: %s:%d (%s) - %v", i+1, proxy.Host, proxy.Port, proxy.Country, proxy.Latency)
	}

	return nil
}

// GetFastestProxy returns the fastest working proxy
func (pm *ProxyManager) GetFastestProxy() (*ProxyInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Auto-update if needed
	if time.Since(pm.lastUpdate) > pm.updateInterval {
		pm.mutex.RUnlock()
		if err := pm.UpdateProxies(); err != nil {
			pm.mutex.RLock()
			logger.Warn("Failed to update proxies: %v", err)
		} else {
			pm.mutex.RLock()
		}
	}

	if len(pm.proxies) == 0 {
		return nil, fmt.Errorf("no working proxies available")
	}

	// Return the fastest (first in sorted list)
	return &pm.proxies[0], nil
}

// GetTopProxies returns the top N fastest proxies
func (pm *ProxyManager) GetTopProxies(n int) []ProxyInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if n > len(pm.proxies) {
		n = len(pm.proxies)
	}

	result := make([]ProxyInfo, n)
	copy(result, pm.proxies[:n])
	return result
}

// Global proxy manager instance
var globalProxyManager = NewProxyManager()

// InitializeProxyManager initializes the global proxy manager
func InitializeProxyManager() error {
	logger.Info("Initializing proxy manager...")
	return globalProxyManager.UpdateProxies()
}

// GetProxyStats returns current proxy statistics
func GetProxyStats() map[string]interface{} {
	globalProxyManager.mutex.RLock()
	defer globalProxyManager.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_proxies": len(globalProxyManager.proxies),
		"last_update":   globalProxyManager.lastUpdate,
	}

	if len(globalProxyManager.proxies) > 0 {
		stats["fastest_latency"] = globalProxyManager.proxies[0].Latency
		stats["fastest_proxy"] = fmt.Sprintf("%s:%d",
			globalProxyManager.proxies[0].Host,
			globalProxyManager.proxies[0].Port)
	}

	return stats
}

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
// FLARESOLVERR METHODS (UPDATED WITH PROXY MANAGER)
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

// fetchWithFlareSolverr fetches URL using FlareSolverr with dynamic proxy selection
func (c *UnifiedCrawler) fetchWithFlareSolverr() (io.Reader, error) {
	client := &http.Client{Timeout: 120 * time.Second}

	// First try without proxy
	payload := map[string]interface{}{
		"cmd":        "request.get",
		"url":        c.URL,
		"maxTimeout": 20000,
	}

	reader, err := c.executeFlareSolverrRequest(client, payload)
	if err == nil {
		logger.Info("[%s] FlareSolverr succeeded without proxy", c.Provider)
		return reader, nil
	}

	logger.Warn("[%s] FlareSolverr failed without proxy: %v, trying with fastest proxy", c.Provider, err)

	// Try with fastest proxy
	if err := globalProxyManager.UpdateProxies(); err != nil {
		logger.Warn("[%s] Failed to update proxy list: %v", c.Provider, err)
	}

	fastestProxy, err := globalProxyManager.GetFastestProxy()
	if err != nil {
		logger.Warn("[%s] No working proxies available: %v", c.Provider, err)
		return nil, fmt.Errorf("FlareSolverr failed and no proxies available: %v", err)
	}

	// Retry with fastest proxy
	proxyURL := fmt.Sprintf("socks5://%s:%d", fastestProxy.Host, fastestProxy.Port)
	payload["proxy"] = map[string]interface{}{
		"url": proxyURL,
	}

	logger.Info("[%s] Trying FlareSolverr with fastest proxy: %s (latency: %v)",
		c.Provider, proxyURL, fastestProxy.Latency)

	reader, err = c.executeFlareSolverrRequest(client, payload)
	if err == nil {
		logger.Info("[%s] FlareSolverr succeeded with proxy %s", c.Provider, proxyURL)
		return reader, nil
	}

	// If fastest proxy fails, try top 3 proxies
	logger.Warn("[%s] Fastest proxy failed: %v, trying top 3 proxies", c.Provider, err)

	topProxies := globalProxyManager.GetTopProxies(3)
	for i, proxy := range topProxies {
		if i == 0 {
			continue // Skip first one (already tried)
		}

		proxyURL := fmt.Sprintf("socks5://%s:%d", proxy.Host, proxy.Port)
		payload["proxy"] = map[string]interface{}{
			"url": proxyURL,
		}

		logger.Debug("[%s] Trying proxy %d/3: %s (latency: %v)",
			c.Provider, i+1, proxyURL, proxy.Latency)

		reader, err = c.executeFlareSolverrRequest(client, payload)
		if err == nil {
			logger.Info("[%s] FlareSolverr succeeded with proxy %s", c.Provider, proxyURL)
			return reader, nil
		}

		logger.Debug("[%s] Proxy %s failed: %v", c.Provider, proxyURL, err)
	}

	return nil, fmt.Errorf("FlareSolverr failed with all available proxies")
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
