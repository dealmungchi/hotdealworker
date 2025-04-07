package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/internal/crawler"
	"sjsage522/hotdealworker/services/publisher"

	"github.com/stretchr/testify/assert"
)

// MockCrawler implements the crawler.Crawler interface for testing
type MockCrawler struct {
	name     string
	deals    []crawler.HotDeal
	fetchErr error
}

// Ensure MockCrawler implements crawler.Crawler
var _ crawler.Crawler = (*MockCrawler)(nil)

func (m *MockCrawler) FetchDeals() ([]crawler.HotDeal, error) {
	return m.deals, m.fetchErr
}

func (m *MockCrawler) GetName() string {
	return m.name
}

func (m *MockCrawler) GetProvider() string {
	return "Test"
}

// MockPublisher implements the publisher.Publisher interface for testing
type MockPublisher struct {
	mu       sync.Mutex
	messages map[string][]byte
	stream   string
}

// Ensure MockPublisher implements publisher.Publisher
var _ publisher.Publisher = (*MockPublisher)(nil)

func NewMockPublisher() *MockPublisher {
	return &MockPublisher{
		messages: make(map[string][]byte),
		stream:   "test_stream",
	}
}

func (m *MockPublisher) Publish(key string, message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Copy the message to ensure thread safety
	messageCopy := make([]byte, len(message))
	copy(messageCopy, message)

	m.messages[m.stream] = messageCopy
	return nil
}

func (m *MockPublisher) TrimStreams() error {
	return nil
}

func (m *MockPublisher) Close() error {
	return nil
}

// MockLogger implements the helpers.LoggerInterface for testing
type MockLogger struct {
	mu     sync.Mutex
	errors []string
	infos  []string
}

// Ensure MockLogger implements helpers.LoggerInterface
var _ helpers.LoggerInterface = (*MockLogger)(nil)

func NewMockLogger() *MockLogger {
	return &MockLogger{
		errors: make([]string, 0),
		infos:  make([]string, 0),
	}
}

func (m *MockLogger) LogError(crawlerName string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, crawlerName+": "+err.Error())
}

func (m *MockLogger) LogInfo(format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.infos = append(m.infos, fmt.Sprintf(format, args...))
}

// TestWorkerCrawlAndPublish tests the crawlAndPublish method
func TestWorkerCrawlAndPublish(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	mockPublisher := NewMockPublisher()

	// Create a worker with a mock crawler that returns deals
	deals := []crawler.HotDeal{
		{
			Title: "Test Deal 1",
			Link:  "https://example.com/deal1",
			Price: "$10",
		},
	}

	mockCrawler := &MockCrawler{
		name:     "TestCrawler",
		deals:    deals,
		fetchErr: nil,
	}

	w := NewWorker(
		ctx,
		[]crawler.Crawler{mockCrawler},
		mockPublisher,
		mockLogger,
		1*time.Second,
	)

	// Run the crawlAndPublish method
	w.crawlAndPublish(mockCrawler)

	// Give a small amount of time for any asynchronous operations to complete
	time.Sleep(50 * time.Millisecond)

	// Verify that the deals were published
	assert.Contains(t, mockPublisher.messages, "test_stream", "Channel should exist in messages")

	// Verify the message contains both deals
	messageContent := string(mockPublisher.messages["test_stream"])
	assert.Contains(t, messageContent, "Test Deal 1", "Message should contain first deal")

	// Ensure no errors were logged
	assert.Empty(t, mockLogger.errors, "No errors should have been logged")
}

// TestWorkerWithError tests error handling in the worker
func TestWorkerWithError(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	mockPublisher := NewMockPublisher()

	// Create a worker with a mock crawler that returns an error
	mockCrawler := &MockCrawler{
		name:     "ErrorCrawler",
		deals:    nil,
		fetchErr: errors.New("test error"),
	}

	w := NewWorker(
		ctx,
		[]crawler.Crawler{mockCrawler},
		mockPublisher,
		mockLogger,
		1*time.Second,
	)

	// Run the crawlAndPublish method
	w.crawlAndPublish(mockCrawler)

	// Give a small amount of time for any asynchronous operations to complete
	time.Sleep(50 * time.Millisecond)

	// Verify that the error was logged
	assert.NotEmpty(t, mockLogger.errors, "An error should have been logged")
	assert.Contains(t, mockLogger.errors[0], "ErrorCrawler", "Error should mention the crawler name")
	assert.Contains(t, mockLogger.errors[0], "test error", "Error should contain the error message")

	// Verify that no messages were published
	assert.Empty(t, mockPublisher.messages, "No messages should have been published")
}

// TestWorkerRunCrawlers tests the runCrawlers method
func TestWorkerRunCrawlers(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	mockPublisher := NewMockPublisher()

	// Create a worker with multiple mock crawlers
	crawler1 := &MockCrawler{
		name: "TestCrawler1",
		deals: []crawler.HotDeal{
			{
				Title: "Test Deal 1",
				Link:  "https://example.com/deal1",
				Price: "$10",
			},
		},
		fetchErr: nil,
	}

	crawler2 := &MockCrawler{
		name: "TestCrawler2",
		deals: []crawler.HotDeal{
			{
				Title: "Test Deal 2",
				Link:  "https://example.com/deal2",
				Price: "$20",
			},
		},
		fetchErr: nil,
	}

	w := NewWorker(
		ctx,
		[]crawler.Crawler{crawler1, crawler2},
		mockPublisher,
		mockLogger,
		1*time.Second,
	)

	// Run the runCrawlers method
	w.runCrawlers()

	// Wait a short time for all goroutines to complete and messages to be processed
	time.Sleep(300 * time.Millisecond)

	// Verify that a crawler published to the stream
	assert.Contains(t, mockPublisher.messages, "test_stream", "Channel should exist in messages")

	// Get the message content
	messageContent := string(mockPublisher.messages["test_stream"])

	// Due to race conditions, we can't guarantee which crawler's result will be in the stream
	// We just need to make sure one of them succeeded
	hasDeals := strings.Contains(messageContent, "Test Deal 1") ||
		strings.Contains(messageContent, "Test Deal 2")

	if !assert.True(t, hasDeals, "Message should contain at least one deal") {
		t.Logf("Actual message content: %s", messageContent)
	}

	// Ensure no errors were logged
	assert.Empty(t, mockLogger.errors, "No errors should have been logged")
}
