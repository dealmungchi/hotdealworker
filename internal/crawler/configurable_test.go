package crawler

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

// mockCacheService is a mock implementation of cache.CacheService for testing
type mockCacheService struct {
	data map[string][]byte
}

func newMockCacheService() *mockCacheService {
	return &mockCacheService{
		data: make(map[string][]byte),
	}
}

func (m *mockCacheService) Get(key string) ([]byte, error) {
	if data, ok := m.data[key]; ok {
		return data, nil
	}
	return nil, io.EOF
}

func (m *mockCacheService) Set(key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCacheService) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func TestConfigurableCrawler_ProcessDeal(t *testing.T) {
	mockCache := newMockCacheService()

	// Create a configurable crawler for testing
	crawler := NewConfigurableCrawler(CrawlerConfig{
		URL:       "https://example.com",
		CacheKey:  "test_rate_limited",
		BlockTime: 500,
		BaseURL:   "https://example.com",
		Provider:  "Test",
		Selectors: Selectors{
			DealList:   ".item",
			Title:      "h3.title",
			Link:       "a.link",
			Thumbnail:  "img.thumbnail",
			PostedAt:   ".date",
			PriceRegex: `\(([0-9,]+원)\)$`,
		},
		IDExtractor: func(link string) (string, error) {
			return "123", nil
		},
	}, mockCache)

	// Create a test HTML document
	html := `
		<div class="item">
			<h3 class="title">Test Deal (10,000원)</h3>
			<a class="link" href="/deals/123">View Deal</a>
			<img class="thumbnail" src="/images/test.jpg" />
			<span class="date">2023-01-01</span>
		</div>
	`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	assert.NoError(t, err)

	// Process the deal
	deal, err := crawler.processDeal(doc.Find(".item"))
	assert.NoError(t, err)
	assert.NotNil(t, deal)

	// Verify the deal properties
	assert.Equal(t, "123", deal.Id)
	assert.Equal(t, "Test Deal", deal.Title)
	assert.Equal(t, "https://example.com/deals/123", deal.Link)
	assert.Equal(t, "10,000원", deal.Price)
	assert.Equal(t, "2023-01-01", deal.PostedAt)
	assert.Equal(t, "Test", deal.Provider)
}

func TestConfigurableCrawler_ExtractPrice(t *testing.T) {
	// Create a configurable crawler for testing
	crawler := &BaseCrawler{
		PriceRegex: `\(([0-9,]+원)\)$`,
	}

	// Test cases
	testCases := []struct {
		title         string
		expectedTitle string
		expectedPrice string
	}{
		{
			title:         "Test Deal (10,000원)",
			expectedTitle: "Test Deal",
			expectedPrice: "10,000원",
		},
		{
			title:         "Test Deal Without Price",
			expectedTitle: "Test Deal Without Price",
			expectedPrice: "",
		},
		{
			title:         "Test Deal (10,000원) with extra parentheses",
			expectedTitle: "Test Deal (10,000원) with extra parentheses",
			expectedPrice: "",
		},
	}

	for _, tc := range testCases {
		title, price := crawler.ExtractPrice(tc.title)
		assert.Equal(t, tc.expectedTitle, title)
		assert.Equal(t, tc.expectedPrice, price)
	}
}

func TestConfigurableCrawler_ResolveURL(t *testing.T) {
	// Create a configurable crawler for testing
	crawler := &BaseCrawler{
		URL:     "https://example.com/deals",
		BaseURL: "https://example.com",
	}

	// Test cases
	testCases := []struct {
		href     string
		expected string
	}{
		{
			href:     "/deals/123",
			expected: "https://example.com/deals/123",
		},
		{
			href:     "//example.com/deals/123",
			expected: "https://example.com/deals/123",
		},
		{
			href:     "https://other.com/deals/123",
			expected: "https://other.com/deals/123",
		},
		{
			href:     "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		result := crawler.ResolveURL(tc.href)
		assert.Equal(t, tc.expected, result)
	}
}

func TestPostedAtHandlerFunc(t *testing.T) {
	mockCache := newMockCacheService()

	// Create a handler function for testing
	handler := func(s *goquery.Selection) string {
		// First try the first selector: span.orangered.da-list-date
		postedAt := s.Find("span.orangered.da-list-date").Text()
		if postedAt == "" {
			// If that fails, try the second approach with removal
			postedAtSel := s.Find("div.wr-date.text-nowrap")
			// Clone to avoid modifying the original selection
			postedAtSelClone := postedAtSel.Clone()
			// Remove unwanted elements
			postedAtSelClone.Find("i").Remove()
			postedAtSelClone.Find("span").Remove()
			postedAt = strings.TrimSpace(postedAtSelClone.Text())
		} else {
			postedAt = strings.TrimSpace(postedAt)
		}
		return postedAt
	}

	// Test directly with the handler function
	// Test case 1: First selector (span.orangered.da-list-date) has content
	html1 := `
		<li>
			<a href="/link">Test Title</a>
			<span class="orangered da-list-date">  2023-01-01  </span>
			<div class="wr-date text-nowrap">
				<i class="icon"></i>
				<span class="text">Irrelevant</span>
				Wrong date
			</div>
		</li>
	`
	doc1, err := goquery.NewDocumentFromReader(strings.NewReader(html1))
	assert.NoError(t, err)

	// Call the handler directly
	postedAt1 := handler(doc1.Find("li"))
	assert.Equal(t, "2023-01-01", postedAt1)

	// Test case 2: First selector is empty, use second selector with cleanup
	html2 := `
		<li>
			<a href="/link">Test Title</a>
			<span class="orangered da-list-date"></span>
			<div class="wr-date text-nowrap">
				<i class="icon"></i>
				<span class="text">Irrelevant</span>
				2023-02-01
			</div>
		</li>
	`
	doc2, err := goquery.NewDocumentFromReader(strings.NewReader(html2))
	assert.NoError(t, err)

	// Call the handler directly
	postedAt2 := handler(doc2.Find("li"))
	assert.Equal(t, "2023-02-01", postedAt2)

	// Test case 3: Both selectors are empty
	html3 := `
		<li>
			<a href="/link">Test Title</a>
			<span class="other-class"></span>
			<div class="other-date"></div>
		</li>
	`
	doc3, err := goquery.NewDocumentFromReader(strings.NewReader(html3))
	assert.NoError(t, err)

	// Call the handler directly
	postedAt3 := handler(doc3.Find("li"))
	assert.Equal(t, "", postedAt3)

	// Now test the integration with a configurable crawler
	// Create a configurable crawler with the custom handler
	crawler := NewConfigurableCrawler(CrawlerConfig{
		URL:       "https://damoang.net",
		CacheKey:  "damoang_test",
		BlockTime: 500,
		BaseURL:   "https://damoang.net",
		Provider:  "Damoang",
		Selectors: Selectors{
			DealList: "li",
			Title:    "a.title",
			Link:     "a.title",
			PostedAt: "span.orangered.da-list-date, div.wr-date.text-nowrap",
		},
		IDExtractor: func(link string) (string, error) {
			return "123", nil
		},
		CustomHandlers: CustomHandlers{
			ElementHandlers: map[string]CustomElementHandlerFunc{
				"postedAt": handler,
			},
		},
	}, mockCache)

	// Create a test HTML document with all required elements
	html4 := `
		<li>
			<a class="title" href="/link">Test Title</a>
			<span class="orangered da-list-date">2023-01-01</span>
		</li>
	`
	doc4, err := goquery.NewDocumentFromReader(strings.NewReader(html4))
	assert.NoError(t, err)

	// Process the deal
	deal, err := crawler.processDeal(doc4.Find("li"))
	assert.NoError(t, err)
	assert.NotNil(t, deal)
	assert.Equal(t, "2023-01-01", deal.PostedAt)
}
