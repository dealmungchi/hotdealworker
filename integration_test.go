package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/internal/crawler"
	"sjsage522/hotdealworker/services/cache"
	"sjsage522/hotdealworker/services/publisher"

	"github.com/PuerkitoBio/goquery"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// This is a simple test HTML that mimics a hot deal listing page
const testHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Test Hot Deals</title>
</head>
<body>
    <div class="list">
        <div class="item">
            <h3 class="title"><a href="/deal/1">Test Deal 1</a></h3>
            <div class="price">$10.99</div>
            <div class="thumb"><img src="/img/1.jpg" alt="Thumbnail" /></div>
            <div class="date">2023-01-01 12:00:00</div>
        </div>
        <div class="item">
            <h3 class="title"><a href="/deal/2">Test Deal 2</a></h3>
            <div class="price">$20.99</div>
            <div class="thumb"><img src="/img/2.jpg" alt="Thumbnail" /></div>
            <div class="date">2023-01-02 12:00:00</div>
        </div>
    </div>
</body>
</html>
`

// TestCrawler is a simple crawler implementation for testing
type TestCrawler struct {
	URL       string
	CacheKey  string
	CacheSvc  cache.CacheService
	BlockTime time.Duration
	server    *httptest.Server
}

// Ensure TestCrawler implements crawler.Crawler
var _ crawler.Crawler = (*TestCrawler)(nil)

// MockCacheService implements a simple in-memory cache for testing
type MockCacheService struct {
	cache map[string][]byte
}

// Ensure MockCacheService implements cache.CacheService
var _ cache.CacheService = (*MockCacheService)(nil)

func (m *MockCacheService) Get(key string) ([]byte, error) {
	if val, ok := m.cache[key]; ok {
		return val, nil
	}
	return nil, errors.New("cache miss")
}

func (m *MockCacheService) Set(key string, value []byte, expiration time.Duration) error {
	m.cache[key] = value
	return nil
}

func (m *MockCacheService) Delete(key string) error {
	delete(m.cache, key)
	return nil
}

func NewTestCrawler(server *httptest.Server, cacheSvc cache.CacheService) *TestCrawler {
	return &TestCrawler{
		URL:       server.URL,
		CacheKey:  "test_rate_limited",
		CacheSvc:  cacheSvc,
		BlockTime: 1 * time.Second,
		server:    server,
	}
}

func (c *TestCrawler) GetName() string {
	return "TestCrawler"
}

func (c *TestCrawler) FetchDeals() ([]crawler.HotDeal, error) {
	utf8Body, err := helpers.FetchWithRandomHeaders(c.server.URL)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		return nil, err
	}

	dealSelections := doc.Find("div.item")

	// Process deals directly since we can't use BaseCrawler
	dealChan := make(chan *crawler.HotDeal, dealSelections.Length())
	var wg sync.WaitGroup

	dealSelections.Each(func(i int, s *goquery.Selection) {
		wg.Add(1)
		go func(s *goquery.Selection) {
			defer wg.Done()

			// Process the deal in the goroutine
			deal := c.processDeal(s)
			if deal != nil {
				dealChan <- deal
			}
		}(s)
	})

	wg.Wait()
	close(dealChan)

	// Collect the processed deals
	var deals []crawler.HotDeal
	for deal := range dealChan {
		if deal != nil {
			deals = append(deals, *deal)
		}
	}

	return deals, nil
}

func (c *TestCrawler) processDeal(s *goquery.Selection) *crawler.HotDeal {
	titleSel := s.Find("h3.title a")
	title := titleSel.Text()
	link, _ := titleSel.Attr("href")
	if link != "" {
		link = c.server.URL + link
	}

	price := s.Find("div.price").Text()
	thumb, _ := s.Find("div.thumb img").Attr("src")
	if thumb != "" {
		thumb = c.server.URL + thumb
	}

	postedAt := s.Find("div.date").Text()

	return &crawler.HotDeal{
		Title:     title,
		Link:      link,
		Price:     price,
		Thumbnail: thumb,
		PostedAt:  postedAt,
	}
}

// TestIntegration tests the entire application flow
func TestIntegration(t *testing.T) {
	// Skip this test if running in CI or without Redis/Memcached
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Create a test server that serves the test HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, testHTML)
	}))
	defer server.Close()

	// Set up the context
	ctx := context.Background()

	// Skip Redis test conditions
	redisAddr := "localhost:6379"
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})
	defer redisClient.Close()

	// Check if Redis is available by attempting a ping, skip test if not
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis is not available, skipping integration test")
	}

	// Create a test channel name
	testChannel := "test_hotdeals"

	// Create a subscription to the test channel
	pubsub := redisClient.Subscribe(ctx, testChannel)
	defer pubsub.Close()

	// Create a channel to receive the decoded and unmarshaled HotDeal objects
	messages := make(chan []crawler.HotDeal, 1)

	// Start a goroutine to receive and process messages
	go func() {
		// Wait for a message
		message, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
			return
		}

		// Decode the base64 message payload
		decoded, err := base64.StdEncoding.DecodeString(message.Payload)
		if err != nil {
			t.Errorf("Failed to decode base64 message: %v", err)
			return
		}

		// Parse the JSON message into a slice of HotDeal objects
		var deals []crawler.HotDeal
		err = json.Unmarshal(decoded, &deals)
		if err != nil {
			t.Errorf("Failed to parse JSON message: %v", err)
			return
		}

		// Send the decoded deals to the channel
		messages <- deals
	}()

	// Create a mock cache service for testing
	mockCache := &MockCacheService{
		cache: make(map[string][]byte),
	}

	// Create a test crawler with the mock server and mock cache service
	testCrawler := NewTestCrawler(server, mockCache)

	// Create a Redis publisher pointing to the same Redis instance we're subscribing to
	redisPublisher := publisher.NewRedisPublisher(ctx, redisAddr, 0)
	defer redisPublisher.Close()

	// Set a longer timeout for potentially slow test environments
	timeout := 10 * time.Second

	// Create a channel to signal errors from the publisher goroutine
	errChan := make(chan error, 1)

	// Create a separate goroutine for publishing to avoid blocking
	go func() {
		// Fetch deals from the crawler
		deals, err := testCrawler.FetchDeals()
		if err != nil {
			errChan <- fmt.Errorf("failed to fetch deals: %w", err)
			return
		}

		if len(deals) != 2 {
			errChan <- fmt.Errorf("expected 2 deals, got %d", len(deals))
			return
		}

		// Marshal deals to JSON
		dealsJSON, err := json.Marshal(deals)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal deals to JSON: %w", err)
			return
		}

		// Add a small delay to ensure subscriber is ready
		time.Sleep(200 * time.Millisecond)

		// Publish the deals to Redis
		if err := redisPublisher.Publish(testChannel, dealsJSON); err != nil {
			errChan <- fmt.Errorf("failed to publish deals to Redis: %w", err)
			return
		}

		// Signal success
		errChan <- nil
	}()

	// Wait for either success, an error, or a timeout
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Error in publisher goroutine: %v", err)
		}
	case <-time.After(timeout / 2):
		t.Fatal("Timed out waiting for publisher to complete")
	}

	// Use a significantly longer timeout for slower environments or Redis latency
	messageReceiveTimeout := 30 * time.Second

	// Wait for the message to be received with timeout
	select {
	case receivedDeals := <-messages:
		if !assert.NotNil(t, receivedDeals, "Received deals should not be nil") {
			t.FailNow()
		}

		// Assert the expected number of deals
		if !assert.Len(t, receivedDeals, 2, "Expected 2 deals to be received") {
			t.FailNow()
		}

		expectedDeal1 := crawler.HotDeal{
			Title:     "Test Deal 1",
			Price:     "$10.99",
			Link:      server.URL + "/deal/1",
			Thumbnail: server.URL + "/img/1.jpg",
			PostedAt:  "2023-01-01 12:00:00",
		}
		expectedDeal2 := crawler.HotDeal{
			Title:     "Test Deal 2",
			Price:     "$20.99",
			Link:      server.URL + "/deal/2",
			Thumbnail: server.URL + "/img/2.jpg",
			PostedAt:  "2023-01-02 12:00:00",
		}

		// Assert the received deals contain the expected deals
		assert.Contains(t, receivedDeals, expectedDeal1, "Received deals should contain expected deal 1")
		assert.Contains(t, receivedDeals, expectedDeal2, "Received deals should contain expected deal 2")
	case <-time.After(messageReceiveTimeout):
		t.Fatalf("Timed out waiting for message after %v", messageReceiveTimeout)
	}
}
