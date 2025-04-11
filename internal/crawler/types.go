package crawler

import "github.com/PuerkitoBio/goquery"

// HotDeal represents a scraped hot deal
type HotDeal struct {
	Id            string `json:"id"`
	Title         string `json:"title"`
	Link          string `json:"link"`
	Price         string `json:"price,omitempty"`
	Thumbnail     string `json:"thumbnail,omitempty"`
	ThumbnailLink string `json:"thumbnail_link,omitempty"`
	PostedAt      string `json:"posted_at,omitempty"`
	Provider      string `json:"provider"`
}

// Crawler interface defines the contract for all crawler implementations
type Crawler interface {
	// FetchDeals retrieves hot deals from a source
	FetchDeals() ([]HotDeal, error)

	// GetName returns the crawler's name for logging and identification
	GetName() string

	// GetProvider returns the provider name for the crawler
	GetProvider() string
}

// ElementHandler defines a function to process a DOM element and return a string value
type ElementHandler func(*goquery.Selection) string

// CustomElementHandlerFunc defines a custom handler for a specific element
type CustomElementHandlerFunc func(*goquery.Selection) string

// ProcessorFunc defines the function signature for processing a single deal
type ProcessorFunc func(*goquery.Selection) (*HotDeal, error)

// IDExtractorFunc defines the function signature for extracting an ID from a URL
type IDExtractorFunc func(string) (string, error)

// Selectors contains CSS selectors and handlers for various elements in the page
type Selectors struct {
	// CSS Selectors
	DealList    string
	Title       string
	Link        string
	Price       string
	Thumbnail   string
	PostedAt    string
	PriceRegex  string
	ThumbRegex  string
	ClassFilter string

	// Element handlers for each field
	TitleHandlers     []ElementHandler
	LinkHandlers      []ElementHandler
	PriceHandlers     []ElementHandler
	ThumbnailHandlers []ElementHandler
	PostedAtHandlers  []ElementHandler
}

// CrawlerConfig contains configuration for a crawler
type CrawlerConfig struct {
	URL          string
	CacheKey     string
	BlockTime    int
	BaseURL      string
	Provider     string
	Selectors    Selectors
	IDExtractor  IDExtractorFunc
	UseChrome    bool
	ChromeDBAddr string
}
