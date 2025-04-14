package helpers

import (
	"bytes"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"slices"
	"time"

	"golang.org/x/net/html/charset"
)

// HTTP client and header configurations
var (
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Safari/605.1.15",
	}

	referers = []string{
		"https://www.google.com/",
		"https://www.naver.com/",
		"https://www.daum.net/",
	}

	// HTTP client with timeout
	client = &http.Client{
		Timeout: 10 * time.Second,
	}
)

func FetchSimply(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a random User-Agent header
	rnd := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	req.Header.Set("User-Agent", userAgents[rnd.Intn(len(userAgents))])

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetchSimply unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// FetchWithRandomHeaders sends an HTTP GET request with randomized headers,
// converts the response body to UTF-8 (if needed), and returns it as an io.Reader.
func FetchWithRandomHeaders(url string) (io.Reader, error) {
	// Create a new random number generator for header selection
	rnd := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set browser-like headers
	req.Header.Set("User-Agent", userAgents[rnd.Intn(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("referer", referers[rnd.Intn(len(referers))])
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Priority", "u=0, i")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("Sec-Ch-Ua", "Chromium;v=134, Not:A-Brand;v=24, Google Chrome;v=134")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Sec-Fetch-User", "?1")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Check for rate limiting
	if slices.Contains([]int{http.StatusTooManyRequests, 430}, resp.StatusCode) {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("rate limited; retry after %s", retryAfter)
	}

	// Check for other error status codes
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch %s unexpected status code: %d", url, resp.StatusCode)
	}

	defer resp.Body.Close()

	// Read the entire response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Determine the encoding from Content-Type header and body content
	encoding, name, _ := charset.DetermineEncoding(bodyBytes, resp.Header.Get("Content-Type"))

	// If already UTF-8, return as is
	if name == "utf-8" || name == "UTF-8" {
		return bytes.NewReader(bodyBytes), nil
	}

	// Convert to UTF-8 if necessary
	utf8Reader := encoding.NewDecoder().Reader(bytes.NewReader(bodyBytes))
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, utf8Reader); err != nil {
		return nil, fmt.Errorf("failed to read converted UTF-8 body: %w", err)
	}

	return &buf, nil
}
