package crawler

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"sjsage522/hotdealworker/helpers"
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

// ResolveURL resolves a relative URL against the base URL
func (c *BaseCrawler) ResolveURL(href string) string {
	if href == "" {
		return ""
	}

	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	if !strings.HasPrefix(href, "http") {
		baseURL := c.BaseURL
		if baseURL == "" {
			baseURL = c.URL
		}

		base, err := url.Parse(baseURL)
		if err != nil {
			return href
		}

		ref, err := url.Parse(href)
		if err != nil {
			return href
		}

		return base.ResolveReference(ref).String()
	}

	return href
}

// ExtractPrice extracts the price from a title using the configured regex
func (c *BaseCrawler) ExtractPrice(title string) (string, string) {
	if c.PriceRegex == "" || title == "" {
		return title, ""
	}

	re := regexp.MustCompile(c.PriceRegex)
	if match := re.FindStringSubmatch(title); len(match) > 1 {
		price := strings.TrimSpace(match[1])
		cleanTitle := strings.TrimSpace(strings.Replace(title, "("+match[1]+")", "", 1))
		return cleanTitle, price
	}

	return title, ""
}

// ProcessImage fetches an image and converts it to base64
func (c *BaseCrawler) ProcessImage(imageURL string) (string, error) {
	imageURL = c.ResolveURL(imageURL)
	if imageURL == "" {
		return "", nil
	}

	data, err := helpers.FetchSimply(imageURL)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// CreateDeal creates a HotDeal with the given properties
func (c *BaseCrawler) CreateDeal(id, title, link, price, thumbnail, postedAt string) *HotDeal {
	return &HotDeal{
		Id:        id,
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumbnail,
		PostedAt:  postedAt,
		Provider:  c.Provider,
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