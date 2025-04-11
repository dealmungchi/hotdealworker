package crawler

import (
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

// TestBaseCrawler tests the base crawler functionality
func TestBaseCrawler(t *testing.T) {
	// Create a test crawler
	mockCache := NewMockCacheService()
	crawler := BaseCrawler{
		URL:       "https://example.com",
		CacheKey:  "test_rate_limited",
		CacheSvc:  mockCache,
		BlockTime: 1 * time.Second,
	}

	// Test processing deals
	html := `<html><body>
		<div class="deal">
			<div class="title">Deal 1</div>
			<div class="price">$10</div>
		</div>
		<div class="deal">
			<div class="title">Deal 2</div>
			<div class="price">$20</div>
		</div>
	</body></html>`

	reader := strings.NewReader(html)
	doc, err := goquery.NewDocumentFromReader(reader)
	assert.NoError(t, err)

	dealSelections := doc.Find("div.deal")

	deals := crawler.processDeals(dealSelections, func(s *goquery.Selection) (*HotDeal, error) {
		title := s.Find("div.title").Text()
		price := s.Find("div.price").Text()
		return &HotDeal{
			Title: title,
			Price: price,
		}, nil
	})

	// Sort the deals by title to ensure consistent order for testing
	// since goroutines may complete in any order
	sort.Slice(deals, func(i, j int) bool {
		return deals[i].Title < deals[j].Title
	})

	assert.Equal(t, 2, len(deals))
	assert.Equal(t, "Deal 1", deals[0].Title)
	assert.Equal(t, "$10", deals[0].Price)
	assert.Equal(t, "Deal 2", deals[1].Title)
	assert.Equal(t, "$20", deals[1].Price)
}

// TestGetName tests the GetName function
func TestGetName(t *testing.T) {
	crawler := BaseCrawler{}
	name := crawler.GetName()
	assert.Equal(t, "BaseCrawler", name)
}
