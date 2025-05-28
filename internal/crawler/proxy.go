package crawler

import (
	"context"
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

	"sjsage522/hotdealworker/logger"
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

// ProxyManager manages SOCKS5 proxies from spys.me
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

	logger.Info("Updating proxy list from spys.me")

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

// UpdateGlobalProxies updates the global proxy manager
func UpdateGlobalProxies() error {
	return globalProxyManager.UpdateProxies()
}

// GetFastestGlobalProxy returns the fastest working proxy from global manager
func GetFastestGlobalProxy() (*ProxyInfo, error) {
	return globalProxyManager.GetFastestProxy()
}

// GetTopGlobalProxies returns the top N fastest proxies from global manager
func GetTopGlobalProxies(n int) []ProxyInfo {
	return globalProxyManager.GetTopProxies(n)
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
