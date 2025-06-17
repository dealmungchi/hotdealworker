package crawler

type Selectors struct {
	DealList  Selector
	Title     Selector
	Link      Selector
	Price     Selector
	Thumbnail Selector
	PostedAt  Selector
	Category  Selector
}

type Selector struct {
	tag     string
	filter  TagFilter
	handler TagHandler
}

type TagFilter func(string) bool

type TagHandler func(string) string

type HotDeal struct {
	Id            string `json:"id"`
	Title         string `json:"title"`
	Link          string `json:"link"`
	Price         string `json:"price,omitempty"`
	Thumbnail     string `json:"thumbnail,omitempty"`
	ThumbnailLink string `json:"thumbnail_link,omitempty"`
	PostedAt      string `json:"posted_at,omitempty"`
	Category      string `json:"category"`
	Provider      string `json:"provider"`
}

type CrawlerConfig struct {
	DealURL   string
	URL       string
	Provider  string
	UseChrome bool
	Selectors Selectors
}
