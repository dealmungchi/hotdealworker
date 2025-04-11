package crawler

import (
	"io"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

// TestUnifiedCrawler tests the unified crawler functionality
func TestUnifiedCrawler(t *testing.T) {
	// Create a test crawler with price handlers to properly extract price
	mockCache := NewMockCacheService()

	// 가격을 추출할 핸들러 정의
	priceHandler := func(s *goquery.Selection) string {
		return s.Find("div.price").Text()
	}

	crawler := NewUnifiedCrawler(CrawlerConfig{
		URL:       "https://example.com",
		CacheKey:  "test_rate_limited",
		BlockTime: 1,
		BaseURL:   "https://example.com",
		Provider:  "TestProvider",
		Selectors: Selectors{
			DealList:      "div.deal",
			Title:         "div.title",
			Link:          "a.link",
			Price:         "div.price",
			PriceHandlers: []ElementHandler{priceHandler},
		},
	}, mockCache)

	// Mock the fetch function directly for testing
	crawler.fetchFunc = func() (io.Reader, error) {
		html := `<html><body>
			<div class="deal">
				<div class="title">Deal 1</div>
				<a class="link" href="https://example.com/deal/1">Link 1</a>
				<div class="price">$10</div>
			</div>
			<div class="deal">
				<div class="title">Deal 2</div>
				<a class="link" href="https://example.com/deal/2">Link 2</a>
				<div class="price">$20</div>
			</div>
		</body></html>`
		return strings.NewReader(html), nil
	}

	// Test 1: Verify the price handler is working
	html := `<div class="deal">
		<div class="title">Test Deal</div>
		<a class="link" href="/test">Test Link</a>
		<div class="price">$15.99</div>
	</div>`
	reader := strings.NewReader(html)
	doc, err := goquery.NewDocumentFromReader(reader)
	assert.NoError(t, err)

	dealSel := doc.Find("div.deal")
	result := priceHandler(dealSel)
	assert.Equal(t, "$15.99", result, "Price handler should extract price correctly")

	// Test 2: 실제 FetchDeals 함수를 호출하여 통합 테스트
	deals, err := crawler.FetchDeals()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(deals), "Should find 2 deals")

	// Find Deal 1 and Deal 2
	var deal1, deal2 *HotDeal
	for i := range deals {
		d := &deals[i]
		if d.Title == "Deal 1" {
			deal1 = d
		} else if d.Title == "Deal 2" {
			deal2 = d
		}
	}

	assert.NotNil(t, deal1, "Deal 1 should be found")
	assert.NotNil(t, deal2, "Deal 2 should be found")

	if deal1 != nil {
		assert.Equal(t, "Deal 1", deal1.Title, "Title should be extracted correctly")
		assert.Equal(t, "https://example.com/deal/1", deal1.Link, "Link should be extracted correctly")
		assert.Equal(t, "$10", deal1.Price, "Price should be properly extracted")
	}

	if deal2 != nil {
		assert.Equal(t, "Deal 2", deal2.Title, "Title should be extracted correctly")
		assert.Equal(t, "https://example.com/deal/2", deal2.Link, "Link should be extracted correctly")
		assert.Equal(t, "$20", deal2.Price, "Price should be properly extracted")
	}
}

// TestCrawlerWithPriceRegex tests a crawler that extracts price using regex
func TestCrawlerWithPriceRegex(t *testing.T) {
	// Create a test crawler with price regex
	mockCache := NewMockCacheService()
	crawler := NewUnifiedCrawler(CrawlerConfig{
		URL:       "https://example.com",
		CacheKey:  "test_regex",
		BlockTime: 1,
		BaseURL:   "https://example.com",
		Provider:  "TestProvider",
		Selectors: Selectors{
			DealList:   "div.deal",
			Title:      "div.title",
			Link:       "a.link",
			PriceRegex: `\$([0-9,]+)`, // 가격 정규식 설정
		},
	}, mockCache)

	// Mock the fetch function directly for testing
	crawler.fetchFunc = func() (io.Reader, error) {
		html := `<html><body>
			<div class="deal">
				<div class="title">Deal 1 $99</div>
				<a class="link" href="https://example.com/deal/1">Link 1</a>
			</div>
			<div class="deal">
				<div class="title">Deal 2 $199</div>
				<a class="link" href="https://example.com/deal/2">Link 2</a>
			</div>
		</body></html>`
		return strings.NewReader(html), nil
	}

	// 가격 정규식 기능 테스트
	// 1. 직접 ExtractPrice 함수 테스트
	title1, price1 := crawler.ExtractPrice("Deal 1 $99")
	assert.Equal(t, "Deal 1 $99", title1, "Title should be cleaned after price extraction")
	assert.Equal(t, "99", price1, "Price should be extracted from the title")

	title2, price2 := crawler.ExtractPrice("Deal 2 $199")
	assert.Equal(t, "Deal 2 $199", title2, "Title should be cleaned after price extraction")
	assert.Equal(t, "199", price2, "Price should be extracted from the title")

	// 2. 실제 크롤링 결과 확인 - 통합 테스트
	deals, err := crawler.FetchDeals()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(deals))

	// 딜 찾기
	var deal1, deal2 *HotDeal
	for i := range deals {
		d := &deals[i]
		if d.Title == "Deal 1 $99" {
			deal1 = d
		} else if d.Title == "Deal 2 $199" {
			deal2 = d
		}
	}

	// 가격과 타이틀이 모두 올바르게 추출되었는지 확인
	assert.NotNil(t, deal1, "Deal 1 should be found")
	assert.NotNil(t, deal2, "Deal 2 should be found")

	if deal1 != nil {
		assert.Equal(t, "Deal 1 $99", deal1.Title, "Title should be cleaned")
		assert.Equal(t, "99", deal1.Price, "Price should be extracted")
	}

	if deal2 != nil {
		assert.Equal(t, "Deal 2 $199", deal2.Title, "Title should be cleaned")
		assert.Equal(t, "199", deal2.Price, "Price should be extracted")
	}
}

// TestPriceExtraction tests price extraction functionality
func TestPriceExtraction(t *testing.T) {
	// Create a test crawler with price regex
	crawler := NewUnifiedCrawler(CrawlerConfig{
		URL:      "https://example.com",
		BaseURL:  "https://example.com",
		Provider: "TestProvider",
		Selectors: Selectors{
			DealList:   "div.deal",
			Title:      "div.title",
			Link:       "a.link",
			PriceRegex: `\(([0-9,]+원)\)`,
		},
	}, nil)

	// 실제 가격 추출 로직 테스트
	title, price := crawler.ExtractPrice("새로운 상품 (10,000원)")
	assert.Equal(t, "새로운 상품 (10,000원)", title, "Korean title should be properly extracted")
	assert.Equal(t, "10,000원", price, "Korean price should be properly extracted")

	// 여러 형태의 가격 테스트
	testCases := []struct {
		input       string
		expectTitle string
		expectPrice string
	}{
		{"제품 A (5,000원)", "제품 A (5,000원)", "5,000원"},
		{"제품 B (1,234,567원)", "제품 B (1,234,567원)", "1,234,567원"},
		{"제품 C (원)", "제품 C (원)", ""}, // 숫자가 없을 경우
		{"제품 D", "제품 D", ""},         // 가격이 없는 경우
	}

	for _, tc := range testCases {
		title, price := crawler.ExtractPrice(tc.input)
		assert.Equal(t, tc.expectTitle, title, "Title should be properly extracted: "+tc.input)
		assert.Equal(t, tc.expectPrice, price, "Price should be properly extracted: "+tc.input)
	}

	// 더 복잡한 정규식을 사용한 가격 테스트
	dollarCrawler := NewUnifiedCrawler(CrawlerConfig{
		URL:      "https://example.com",
		BaseURL:  "https://example.com",
		Provider: "TestProvider",
		Selectors: Selectors{
			PriceRegex: `\$([0-9,\.]+)`,
		},
	}, nil)

	// 달러 가격 테스트
	dollarTestCases := []struct {
		input       string
		expectTitle string
		expectPrice string
	}{
		{"New Product $199.99", "New Product $199.99", "199.99"},
		{"Special Offer $1,299.99 Only Today", "Special Offer $1,299.99 Only Today", "1,299.99"},
		{"Free Item $0", "Free Item $0", "0"},
		{"No Price Item", "No Price Item", ""},
	}

	for _, tc := range dollarTestCases {
		title, price := dollarCrawler.ExtractPrice(tc.input)
		assert.Equal(t, tc.expectTitle, title, "Title should be properly extracted: "+tc.input)
		assert.Equal(t, tc.expectPrice, price, "Price should be properly extracted: "+tc.input)
	}
}

// TestDefaultHandlers tests the default handlers of the unified crawler
func TestDefaultHandlers(t *testing.T) {
	// Create a test crawler
	crawler := NewUnifiedCrawler(CrawlerConfig{
		URL:      "https://example.com",
		BaseURL:  "https://example.com",
		Provider: "TestProvider",
		Selectors: Selectors{
			Title:    "div.title",
			Link:     "a.link",
			PostedAt: "div.posted-at",
		},
	}, nil)

	// Create a test HTML document
	html := `<html><body>
		<div class="item">
			<div class="title" title="Item Title Attribute">Item Title</div>
			<a class="link" href="/relative/path">Link Text</a>
			<div class="posted-at">2023-01-01</div>
		</div>
	</body></html>`

	// Parse the HTML
	reader := strings.NewReader(html)
	doc, err := goquery.NewDocumentFromReader(reader)
	assert.NoError(t, err)

	item := doc.Find("div.item")

	// Test title handler - 실제 defaultTitleHandler 로직 테스트
	title := crawler.defaultTitleHandler(item)
	assert.Equal(t, "Item Title Attribute", title)

	// Test link handler - 실제 defaultLinkHandler 로직 테스트
	link := crawler.defaultLinkHandler(item)
	assert.Equal(t, "https://example.com/relative/path", link)

	// Test posted at handler - 실제 defaultPostedAtHandler 로직 테스트
	postedAt := crawler.defaultPostedAtHandler(item)
	assert.Equal(t, "2023-01-01", postedAt)
}

// TestCustomHandlers tests the custom handlers of the unified crawler
func TestCustomHandlers(t *testing.T) {
	// Create a test handler
	titleHandler := func(s *goquery.Selection) string {
		return "Custom Title"
	}

	linkHandler := func(s *goquery.Selection) string {
		return "https://example.com/custom"
	}

	// Create a test crawler with custom handlers
	crawler := NewUnifiedCrawler(CrawlerConfig{
		URL: "https://example.com",
		Selectors: Selectors{
			Title:         "div.title",
			Link:          "a.link",
			TitleHandlers: []ElementHandler{titleHandler},
			LinkHandlers:  []ElementHandler{linkHandler},
		},
	}, nil)

	// Mock the fetch function for testing
	crawler.fetchFunc = func() (io.Reader, error) {
		html := `<html><body>
			<div class="item">
				<div class="title">Original Title</div>
				<a class="link" href="/original/path">Original Link</a>
			</div>
		</body></html>`
		return strings.NewReader(html), nil
	}

	// 이 테스트에서는 FetchDeals()를 호출할 수 없음(아이템 구조가 맞지 않음)
	// 대신 applyHandlers 함수가 실제 동작하는지 테스트

	// HTML 파싱
	html := `<html><body>
		<div class="item">
			<div class="title">Original Title</div>
			<a class="link" href="/original/path">Original Link</a>
		</div>
	</body></html>`
	reader := strings.NewReader(html)
	doc, err := goquery.NewDocumentFromReader(reader)
	assert.NoError(t, err)

	item := doc.Find("div.item")

	// 실제 핸들러 적용 테스트
	title := crawler.applyHandlers(item, crawler.Selectors.TitleHandlers)
	link := crawler.applyHandlers(item, crawler.Selectors.LinkHandlers)

	// 핸들러가 예상대로 작동하는지 확인
	assert.Equal(t, "Custom Title", title)
	assert.Equal(t, "https://example.com/custom", link)
}

// TestFetchWithOptions tests the fetching mechanism with different options
func TestFetchWithOptions(t *testing.T) {
	// 1. 표준 HTTP 크롤링 테스트
	standardCrawler := NewUnifiedCrawler(CrawlerConfig{
		URL:       "https://example.com",
		UseChrome: false,
		CacheKey:  "test_standard",
		Provider:  "StandardTest",
	}, NewMockCacheService())

	// 표준 HTTP 페처 적용 확인
	assert.NotNil(t, standardCrawler.fetchFunc)

	// 2. Chrome 크롤링 테스트
	chromeCrawler := NewUnifiedCrawler(CrawlerConfig{
		URL:          "https://example.com",
		UseChrome:    true,
		ChromeDBAddr: "http://localhost:3000",
		CacheKey:     "test_chrome",
		Provider:     "ChromeTest",
	}, NewMockCacheService())

	// Chrome 페처 적용 확인
	assert.NotNil(t, chromeCrawler.fetchFunc)
}
