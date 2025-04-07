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

// ProcessorFunc defines the function signature for processing a single deal
type ProcessorFunc func(*goquery.Selection) (*HotDeal, error)

// IDExtractorFunc defines the function signature for extracting an ID from a URL
type IDExtractorFunc func(string) (string, error)

// CustomElementHandlerFunc defines a function to customize extraction logic for elements
type CustomElementHandlerFunc func(*goquery.Selection) string

// ElementRemoval defines elements to remove from a selection before extracting text
type ElementRemoval struct {
	Selector    string // Selector to find elements to remove
	ApplyToPath string // The path to apply this to (e.g., "title", "postedAt")
}

// Selectors contains CSS selectors for various elements in the page
type Selectors struct {
	DealList    string
	Title       string
	Link        string
	Price       string
	Thumbnail   string
	PostedAt    string
	PriceRegex  string
	ThumbRegex  string
	ClassFilter string
}

// CustomHandlers contains custom handlers for element processing
type CustomHandlers struct {
	// Map paths to custom handlers
	ElementHandlers map[string]CustomElementHandlerFunc
}

// ElementTransformers contains configurations for transforming elements
type ElementTransformers struct {
	// Elements to remove from selections
	RemoveElements []ElementRemoval
}

// CrawlerConfig contains configuration for a crawler
type CrawlerConfig struct {
	URL                 string
	CacheKey            string
	BlockTime           int
	BaseURL             string
	Provider            string
	Selectors           Selectors
	IDExtractor         IDExtractorFunc
	CustomHandlers      CustomHandlers
	ElementTransformers ElementTransformers
}
