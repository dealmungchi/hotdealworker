package crawler

// HotDeal represents a scraped hot deal
type HotDeal struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Price     string `json:"price,omitempty"`
	Thumbnail string `json:"thumbnail,omitempty"`
	PostedAt  string `json:"posted_at,omitempty"`
	Provider  string `json:"provider"`
}

// Crawler interface defines the contract for all crawler implementations
type Crawler interface {
	// FetchDeals retrieves hot deals from a source
	FetchDeals() ([]HotDeal, error)

	// GetName returns the crawler's name for logging and identification
	GetName() string
}
