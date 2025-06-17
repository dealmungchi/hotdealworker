package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// ProxyManager interface for managing proxies
type ProxyManager interface {
	UpdateProxies() error
	GetFastestProxy() (*ProxyInfo, error)
	GetTopProxies(n int) []ProxyInfo
}

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

// proxyInfo holds proxy information with latency
type proxyInfo struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	Type     string        `json:"type"`
	Country  string        `json:"country"`
	Latency  time.Duration `json:"latency"`
	LastTest time.Time     `json:"last_test"`
	Working  bool          `json:"working"`
}

// ProxyManagerImpl manages SOCKS5 proxies from multiple sources
type ProxyManagerImpl struct {
	proxies        []proxyInfo
	mutex          sync.RWMutex
	lastUpdate     time.Time
	updateInterval time.Duration
}

// NewProxyManager creates a new proxy manager
func NewProxyManager() *ProxyManagerImpl {
	return &ProxyManagerImpl{
		proxies:        make([]proxyInfo, 0),
		updateInterval: 30 * time.Minute, // Update every 30 minutes
	}
}

// fetchProxiesFromMultipleSources fetches SOCKS5 proxies from multiple sources
func (pm *ProxyManagerImpl) fetchProxiesFromMultipleSources() ([]proxyInfo, error) {
	log.Debug().Msg("Fetching SOCKS5 proxies from multiple sources")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Try multiple sources
	sources := []struct {
		url    string
		format string
	}{
		{"https://spys.me/socks.txt", "spys"}, // 가장 품질 좋은 소스 우선
		{"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks5.txt", "simple"},
		{"https://www.proxy-list.download/api/v1/get?type=socks5", "simple"},
		{"https://api.proxyscrape.com/v2/?request=get&protocol=socks5&timeout=10000&country=all&format=textplain", "simple"},
	}

	for _, source := range sources {
		log.Debug().Str("url", source.url).Msg("Trying proxy source")

		req, err := http.NewRequest("GET", source.url, nil)
		if err != nil {
			log.Debug().Err(err).Str("url", source.url).Msg("Failed to create request")
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/plain,text/html,*/*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Connection", "keep-alive")

		resp, err := client.Do(req)
		if err != nil {
			log.Debug().Err(err).Str("url", source.url).Msg("Failed to fetch URL")
			continue
		}
		defer resp.Body.Close()

		log.Debug().Int("status_code", resp.StatusCode).Str("url", source.url).Msg("Received HTTP response")
		if resp.StatusCode != 200 {
			log.Debug().Int("status_code", resp.StatusCode).Str("url", source.url).Msg("Non-200 status code")
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Debug().Err(err).Str("url", source.url).Msg("Failed to read response body")
			continue
		}

		bodyStr := string(body)
		log.Debug().Int("body_length", len(bodyStr)).Str("url", source.url).Msg("Read response body")

		// Check if this looks like HTML (error page)
		if strings.Contains(bodyStr, "<!DOCTYPE") || strings.Contains(bodyStr, "<html") {
			log.Debug().Str("url", source.url).Msg("Response looks like HTML, skipping")
			continue
		}

		// Try to parse proxies from this source
		proxies := pm.parseProxyText(bodyStr, source.url)
		if len(proxies) > 0 {
			log.Info().Int("count", len(proxies)).Str("source", source.url).Msg("Found proxies from source")
			return proxies, nil
		}
	}

	return nil, fmt.Errorf("failed to fetch proxies from all sources")
}

// parseProxyText parses proxy text from different sources
func (pm *ProxyManagerImpl) parseProxyText(bodyStr, source string) []proxyInfo {
	lines := strings.Split(bodyStr, "\n")
	log.Debug().Int("total_lines", len(lines)).Str("source", source).Msg("Parsing proxy text")

	var proxies []proxyInfo
	validCount := 0
	skippedCount := 0

	// Log first few lines to understand format
	for i := 0; i < len(lines) && i < 5; i++ {
		log.Debug().Int("line_number", i+1).Str("line", strings.TrimSpace(lines[i])).Str("source", source).Msg("Sample line")
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || len(line) < 7 { // minimum "1.1.1.1:1"
			skippedCount++
			continue
		}

		validCount++

		// Try different parsing methods based on source
		if strings.Contains(source, "spys.me") {
			// spys.me format: multiple proxies per line with country codes
			lineProxies := pm.parseSpysLine(line)
			proxies = append(proxies, lineProxies...)

			// Log first few parsed proxies from spys.me
			for _, proxy := range lineProxies {
				if len(proxies) <= 5 {
					log.Debug().
						Str("host", proxy.Host).
						Int("port", proxy.Port).
						Str("country", proxy.Country).
						Str("source", "spys.me").
						Msg("Parsed spys.me proxy")
				}
			}
		} else {
			// GitHub and other sources format: one proxy per line (IP:PORT)
			if proxy := pm.parseSingleProxy(line); proxy != nil {
				proxies = append(proxies, *proxy)

				// Log first few parsed proxies
				if len(proxies) <= 5 {
					log.Debug().
						Str("host", proxy.Host).
						Int("port", proxy.Port).
						Str("country", proxy.Country).
						Str("source", source).
						Msg("Parsed simple proxy")
				}
			}
		}
	}

	log.Info().
		Int("total_lines", len(lines)).
		Int("skipped_lines", skippedCount).
		Int("valid_lines", validCount).
		Int("parsed_proxies", len(proxies)).
		Str("source", source).
		Msg("Parsing completed")

	return proxies
}

// parseSingleProxy parses a single proxy from a line (IP:PORT format)
func (pm *ProxyManagerImpl) parseSingleProxy(line string) *proxyInfo {
	// Handle different formats: IP:PORT or just IP:PORT
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return nil
	}

	host := strings.TrimSpace(parts[0])
	portStr := strings.TrimSpace(parts[1])

	// Clean up port string (remove any trailing characters)
	if idx := strings.IndexAny(portStr, " \t\r\n"); idx != -1 {
		portStr = portStr[:idx]
	}

	// Validate IP format
	ipAddr := net.ParseIP(host)
	if ipAddr == nil {
		return nil
	}

	// Reject invalid/reserved IP addresses
	if !pm.isValidPublicIP(ipAddr) {
		return nil
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return nil
	}

	// Reject common non-proxy ports (be more selective)
	badPorts := map[int]bool{
		22:   true, // SSH
		23:   true, // Telnet
		25:   true, // SMTP
		53:   true, // DNS
		110:  true, // POP3
		143:  true, // IMAP
		443:  true, // HTTPS
		993:  true, // IMAPS
		995:  true, // POP3S
		3389: true, // RDP
		5432: true, // PostgreSQL
		3306: true, // MySQL
	}

	if badPorts[port] {
		return nil
	}

	// Accept most ports above 1024, but exclude very low and very high ports
	if port < 80 || port > 65000 {
		return nil
	}

	// Try to determine country from IP address
	country := pm.getCountryFromIP(host)

	return &proxyInfo{
		Host:    host,
		Port:    port,
		Type:    "socks5",
		Country: country,
		Working: false,
	}
}

// parseSpysLine parses multiple proxies from a spys.me format line
// Format: IP:PORT COUNTRY-INFO SIGN IP:PORT COUNTRY-INFO SIGN ...
func (pm *ProxyManagerImpl) parseSpysLine(line string) []proxyInfo {
	var proxies []proxyInfo

	fields := strings.Fields(line)
	for i := 0; i < len(fields); i++ {
		field := fields[i]

		// Check if this field looks like IP:PORT
		if !strings.Contains(field, ":") {
			continue
		}

		// Try to parse as IP:PORT
		parts := strings.Split(field, ":")
		if len(parts) != 2 {
			continue
		}

		host := strings.TrimSpace(parts[0])
		portStr := strings.TrimSpace(parts[1])

		// Validate IP format
		ipAddr := net.ParseIP(host)
		if ipAddr == nil {
			continue
		}

		// Reject invalid/reserved IP addresses
		if !pm.isValidPublicIP(ipAddr) {
			continue
		}

		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			continue
		}

		// Reject common non-proxy ports
		badPorts := map[int]bool{
			22: true, 25: true, 53: true, 110: true, 143: true,
			443: true, 993: true, 995: true, 3389: true, 5432: true, 3306: true,
		}
		if badPorts[port] {
			continue
		}

		// Accept most ports in reasonable range
		if port < 80 || port > 65000 {
			continue
		}

		// Try to extract country from next field if available
		country := "Unknown"
		if i+1 < len(fields) {
			nextField := fields[i+1]
			// spys.me format: "US-H", "RU-H!", "CN-H", etc.
			if parts := strings.Split(nextField, "-"); len(parts) >= 1 && len(parts[0]) == 2 {
				country = parts[0]
			}
		}

		proxy := proxyInfo{
			Host:    host,
			Port:    port,
			Type:    "socks5",
			Country: country,
			Working: false,
		}

		proxies = append(proxies, proxy)
	}

	return proxies
}

// isValidPublicIP checks if an IP address is valid for proxy use
func (pm *ProxyManagerImpl) isValidPublicIP(ip net.IP) bool {
	// Convert to IPv4
	ipv4 := ip.To4()
	if ipv4 == nil {
		return false // Not IPv4
	}

	// Filter out invalid/reserved ranges
	if ipv4[0] == 0 || // 0.0.0.0/8 ("this network")
		ipv4[0] == 127 || // 127.0.0.0/8 (loopback)
		ipv4[0] == 10 || // 10.0.0.0/8 (private)
		(ipv4[0] == 172 && ipv4[1] >= 16 && ipv4[1] <= 31) || // 172.16.0.0/12 (private)
		(ipv4[0] == 192 && ipv4[1] == 168) || // 192.168.0.0/16 (private)
		(ipv4[0] == 169 && ipv4[1] == 254) || // 169.254.0.0/16 (link-local)
		(ipv4[0] == 224) || // 224.0.0.0/4 (multicast)
		(ipv4[0] >= 240) { // 240.0.0.0/4 (reserved)
		return false
	}

	// Filter out broadcast addresses
	if ipv4[3] == 0 || ipv4[3] == 255 {
		return false
	}

	return true
}

// getCountryFromIP attempts to determine country from IP address
func (pm *ProxyManagerImpl) getCountryFromIP(ip string) string {
	// Parse IP to determine rough geographic location
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return "Unknown"
	}

	// Check for specific known IP ranges first
	ipStr := ip

	// Alibaba Cloud (China)
	if strings.HasPrefix(ipStr, "8.") || strings.HasPrefix(ipStr, "47.") || strings.HasPrefix(ipStr, "39.") {
		return "CN"
	}

	// AWS regions
	if strings.HasPrefix(ipStr, "3.") || strings.HasPrefix(ipStr, "13.") || strings.HasPrefix(ipStr, "18.") {
		return "US"
	}

	// Google Cloud
	if strings.HasPrefix(ipStr, "34.") || strings.HasPrefix(ipStr, "35.") {
		return "US"
	}

	// Simple heuristic based on IP ranges
	firstOctet := int(ipAddr.To4()[0])

	switch {
	case firstOctet >= 1 && firstOctet <= 2:
		return "US"
	case firstOctet >= 5 && firstOctet <= 23:
		return "US"
	case firstOctet >= 24 && firstOctet <= 39:
		return "US"
	case firstOctet >= 40 && firstOctet <= 79:
		return "EU"
	case firstOctet >= 80 && firstOctet <= 95:
		return "EU"
	case firstOctet >= 96 && firstOctet <= 126:
		return "US"
	case firstOctet >= 128 && firstOctet <= 159:
		return "US"
	case firstOctet >= 160 && firstOctet <= 191:
		return "Various"
	case firstOctet >= 192 && firstOctet <= 195:
		return "EU"
	case firstOctet >= 196 && firstOctet <= 197:
		return "AF"
	case firstOctet >= 198 && firstOctet <= 199:
		return "US"
	case firstOctet >= 200 && firstOctet <= 201:
		return "LA"
	case firstOctet >= 202 && firstOctet <= 203:
		return "AP"
	case firstOctet >= 210 && firstOctet <= 211:
		return "AP"
	case firstOctet >= 218 && firstOctet <= 222:
		return "AP"
	default:
		return "Unknown"
	}
}

// testProxyLatency tests the latency of a single proxy
func (pm *ProxyManagerImpl) testProxyLatency(proxy *proxyInfo) {
	// Test with a simple TCP connection first
	testStart := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", proxy.Host, proxy.Port), 5*time.Second)
	if err != nil {
		log.Debug().Str("proxy", fmt.Sprintf("%s:%d", proxy.Host, proxy.Port)).Err(err).Msg("TCP connection failed")
		proxy.Working = false
		proxy.Latency = time.Hour
		return
	}
	defer conn.Close()

	// Try a basic SOCKS5 handshake to verify it's actually a SOCKS proxy
	if !pm.testSOCKS5Handshake(conn) {
		log.Debug().Str("proxy", fmt.Sprintf("%s:%d", proxy.Host, proxy.Port)).Msg("SOCKS5 handshake failed")
		proxy.Working = false
		proxy.Latency = time.Hour
		return
	}

	// If both TCP and SOCKS5 work, mark as working
	proxy.Working = true
	proxy.Latency = time.Since(testStart)
	proxy.LastTest = time.Now()

	log.Debug().
		Str("host", proxy.Host).
		Int("port", proxy.Port).
		Dur("latency", proxy.Latency).
		Msg("Proxy working (SOCKS5 verified)")
}

// testSOCKS5Handshake performs a basic SOCKS5 handshake
func (pm *ProxyManagerImpl) testSOCKS5Handshake(conn net.Conn) bool {
	// Set a short timeout for the handshake
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{})

	// SOCKS5 authentication request: [VER, NMETHODS, METHODS]
	// VER=5, NMETHODS=1, METHODS=0 (no authentication)
	authReq := []byte{0x05, 0x01, 0x00}

	_, err := conn.Write(authReq)
	if err != nil {
		return false
	}

	// Read authentication response: [VER, METHOD]
	authResp := make([]byte, 2)
	_, err = conn.Read(authResp)
	if err != nil {
		return false
	}

	// Check if server supports SOCKS5 and no-auth method
	if authResp[0] != 0x05 || authResp[1] != 0x00 {
		return false
	}

	return true
}

// UpdateProxies fetches and tests proxies
func (pm *ProxyManagerImpl) UpdateProxies() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Check if we need to update
	if time.Since(pm.lastUpdate) < pm.updateInterval && len(pm.proxies) > 0 {
		return nil
	}

	log.Info().Msg("Updating proxy list from multiple sources")

	// Fetch new proxies with timeout
	newProxies, err := pm.fetchProxiesFromMultipleSources()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch proxies from all sources")
		// If we have existing proxies, keep using them
		if len(pm.proxies) > 0 {
			log.Info().Int("existing_count", len(pm.proxies)).Msg("Keeping existing proxies")
			return nil
		}
		return fmt.Errorf("failed to fetch proxies and no existing proxies available: %v", err)
	}

	if len(newProxies) == 0 {
		log.Warn().Msg("No proxies found from any source")
		// If we have existing proxies, keep using them
		if len(pm.proxies) > 0 {
			log.Info().Int("existing_count", len(pm.proxies)).Msg("Keeping existing proxies")
			return nil
		}
		return fmt.Errorf("no proxies found")
	}

	// Test proxies in batches to find working ones quickly
	var workingProxies []proxyInfo
	var mu sync.Mutex
	batchSize := 50
	targetCount := 5

	log.Info().
		Int("batchSize", batchSize).
		Int("targetCount", targetCount).
		Msg("Testing proxies in batches")

	for i := 0; i < len(newProxies) && len(workingProxies) < targetCount*2; i += batchSize {
		end := i + batchSize
		if end > len(newProxies) {
			end = len(newProxies)
		}

		batch := newProxies[i:end]
		log.Debug().
			Int("start", i+1).
			Int("end", end).
			Int("count", len(batch)).
			Msg("Testing batch")

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 10) // Limit concurrent tests

		for j := range batch {
			wg.Add(1)
			go func(proxy *proxyInfo) {
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

		log.Info().Int("working", len(workingProxies)).Msg("Batch complete")

		// If we have enough good proxies, we can stop early
		if len(workingProxies) >= targetCount {
			log.Info().Int("count", len(workingProxies)).Msg("Found enough working proxies, stopping early")
			break
		}
	}

	// Keep only top fastest proxies
	if len(workingProxies) > targetCount {
		workingProxies = workingProxies[:targetCount]
		log.Info().Int("count", targetCount).Msg("Selected top fastest proxies")
	}

	pm.proxies = workingProxies
	pm.lastUpdate = time.Now()

	fastestLatency := "none"
	if len(workingProxies) > 0 {
		fastestLatency = workingProxies[0].Latency.String()
	}

	log.Info().
		Int("count", len(workingProxies)).
		Str("fastest", fastestLatency).
		Msg("Updated proxy list")

	// Log top proxies for debugging
	for i, proxy := range workingProxies {
		log.Info().
			Int("rank", i+1).
			Str("host", proxy.Host).
			Int("port", proxy.Port).
			Str("country", proxy.Country).
			Dur("latency", proxy.Latency).
			Msg("Top proxy")
	}

	return nil
}

// GetFastestProxy returns the fastest working proxy
func (pm *ProxyManagerImpl) GetFastestProxy() (*ProxyInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Auto-update if needed
	if time.Since(pm.lastUpdate) > pm.updateInterval {
		pm.mutex.RUnlock()
		log.Debug().Msg("Proxy list is stale, attempting to update")
		if err := pm.UpdateProxies(); err != nil {
			log.Warn().Err(err).Msg("Failed to update proxies")
		}
		pm.mutex.RLock()
	}

	if len(pm.proxies) == 0 {
		log.Debug().Msg("No working proxies available")
		return nil, fmt.Errorf("no working proxies available - proxy service may be unavailable")
	}

	// Return the fastest (first in sorted list)
	proxy := &pm.proxies[0]
	log.Debug().
		Str("host", proxy.Host).
		Int("port", proxy.Port).
		Dur("latency", proxy.Latency).
		Msg("Returning fastest proxy")

	return &ProxyInfo{
		Host:     proxy.Host,
		Port:     proxy.Port,
		Type:     proxy.Type,
		Country:  proxy.Country,
		Latency:  proxy.Latency,
		LastTest: proxy.LastTest,
		Working:  proxy.Working,
	}, nil
}

// GetTopProxies returns the top N fastest proxies
func (pm *ProxyManagerImpl) GetTopProxies(n int) []ProxyInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if n > len(pm.proxies) {
		n = len(pm.proxies)
	}

	result := make([]ProxyInfo, n)
	for i := 0; i < n; i++ {
		proxy := &pm.proxies[i]
		result[i] = ProxyInfo{
			Host:     proxy.Host,
			Port:     proxy.Port,
			Type:     proxy.Type,
			Country:  proxy.Country,
			Latency:  proxy.Latency,
			LastTest: proxy.LastTest,
			Working:  proxy.Working,
		}
	}
	return result
}

// Global proxy manager instance
var globalProxyManager = NewProxyManager()

// InitializeProxyManager initializes the global proxy manager
func InitializeProxyManager() error {
	log.Info().Msg("Initializing proxy manager...")
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
