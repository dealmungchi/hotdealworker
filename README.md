# HotDeal Worker

A worker program that crawls hot deal information and publishes it to Redis.

## Features

- Concurrent crawling of multiple hot deal sites  
- Rate limiting prevention and handling (using Memcached)  
- JSON to Base64-encoded message publishing (Redis pub/sub)  
- Environment variable configuration  
- Logging and error handling  

## Supported Sites

- FMKorea - **in progress**
- Damoang  
- Arca Live  
- Quasar Zone  
- Coolandjoy  
- Clien  
- Ppomppu  
- Ppomppu English  
- Ruliweb  

## Environments

| Variable | Description | Default |
|----------|-------------|---------|
| REDIS_ADDR | Redis server address | localhost:6379 |
| REDIS_DB | Redis database number | 0 |
| REDIS_STREAM | Redis stream prefix | streamHotdeals |
| REDIS_STREAM_COUNT | Redis stream count | 1 |
| REDIS_STREAM_MAX_LENGTH | Maximum number of entries to keep in each Redis stream | 100 |
| MEMCACHE_ADDR | Memcached server address | localhost:11211 |
| CRAWL_INTERVAL_SECONDS | Crawling interval (in seconds) | 60 |
| FMKOREA_URL | FMKorea crawling URL | http://www.fmkorea.com |
| DAMOANG_URL | Damoang crawling URL | https://damoang.net |
| ARCA_URL | Arca Live crawling URL | https://arca.live |
| QUASAR_URL | Quasar Zone crawling URL | https://quasarzone.com |
| COOLANDJOY_URL | Coolandjoy crawling URL | https://coolenjoy.net |
| CLIEN_URL | Clien crawling URL | https://www.clien.net/service |
| PPOM_URL | Ppomppu crawling URL | https://www.ppomppu.co.kr |
| PPOMEN_URL | Ppomppu English crawling URL | https://www.ppomppu.co.kr |
| RULIWEB_URL | Ruliweb crawling URL | https://bbs.ruliweb.com |

## Installation

### Basic

```bash
# Clone the repository
git clone https://github.com/yourusername/hotdealworker.git
cd hotdealworker

# Install dependencies
go mod download

# Build
go build -o hotdealworker

# Run
./hotdealworker
```

### Docker

```bash
# Run with Docker Compose
docker compose up -d
```

## Message Structure

Messages published to Redis are Base64-encoded JSON arrays with the following structure:

```json
[
  {
    "id": "1",
    "title": "Product Name",
    "link": "Product Link",
    "price": "Price",
    "thumbnail": "Thumbnail Image (Base64encoded)",
    "posted_at": "Posted DateTime",
    "provider": "provider"
  },
  ...
]
```

## Tests

```bash
# Run all tests
make test

# Run unit tests only
make unit-test

# Run integration tests only
make integration-test
```

## Modules

- `config/`: Application configuration
- `services/`: Service layer (cache, publisher, worker)
- `internal/crawler/`: Crawler interface and implementations
- `helpers/`: Utility functions (HTTP, logging, etc.)

## Architecture

### Streaming Architecture

HotDeal Worker uses Redis Streams for publishing hot deal data:

1. **Multiple Streams**: Distributes data across multiple Redis Streams for better load balancing
   - Configuration option `REDIS_STREAM_COUNT` controls number of streams
   - Messages are randomly assigned to one of these streams

2. **Stream Trimming**: Automatically manages memory usage with stream trimming
   - Each stream is trimmed after every crawling cycle
   - Configuration option `REDIS_STREAM_MAX_LENGTH` (default: 100) controls max entries
   - Prevents unbounded growth of Redis memory usage

3. **Base64 Encoding**: All messages are Base64 encoded for consistent storage
   - Preserves binary data and special characters
   - Simplifies message handling and client processing

### Crawler Structure

HotDeal Worker uses the following crawler architecture:

1. **BaseCrawler**: Provides shared functionality for all crawlers  
   - Rate limiting handling  
   - URL resolution (relative to absolute)  
   - Thumbnail image handling  
   - Price extraction  

2. **ConfigurableCrawler**: Flexible, configuration-driven crawler  
   - Allows crawler creation per site via configuration only  
   - Operates based on CSS selectors  
   - Supports custom handlers and element transformations
   - Reusable modular components  

3. **Site-specific Crawlers**: Handle site-specific needs  
   - Each crawler in its own file for better organization
   - Config-based approach: reuse common logic with only config differences
   - Custom handlers for sites with special extraction requirements

### Customization System

The crawler architecture includes a flexible customization system:

1. **CustomHandlers**: Allow specialized element processing
   - Map-based approach mapping elements to handler functions
   - Can provide custom extraction logic for any element type
   - Enables special processing for complex site structures

2. **ElementTransformers**: Provide HTML element manipulations
   - Support for removing specific elements from selections
   - Configurable by element path (title, postedAt, etc.)
   - Enhances extraction accuracy by cleaning unwanted elements

### Adding a New Crawler

To add a new crawler:

1. Create a new file in `internal/crawler` named after the site (e.g., `mynewsite.go`)
2. Implement a constructor function that returns a `ConfigurableCrawler`:

```go
package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"

	"github.com/PuerkitoBio/goquery"
)

// NewMySiteCrawler creates a crawler for MySite
func NewMySiteCrawler(cfg config.Config, cacheSvc cache.CacheService) *ConfigurableCrawler {
	// Define custom handlers if needed
	customHandlers := CustomHandlers{
		ElementHandlers: map[string]CustomElementHandlerFunc{
			"postedAt": func(s *goquery.Selection) string {
				// Custom extraction logic
				return strings.TrimSpace(s.Find(".date-element").Text())
			},
		},
	}

	// Define element transformers if needed
	elementTransformers := ElementTransformers{
		RemoveElements: []ElementRemoval{
			{Selector: "span.junk", ApplyToPath: "title"},
		},
	}

	return NewConfigurableCrawler(CrawlerConfig{
		URL:       cfg.MySiteURL, // Add to config struct
		CacheKey:  "mysite_rate_limited",
		BlockTime: 500,
		BaseURL:   "https://www.mysite.com",
		Provider:  "MySite",
		Selectors: Selectors{
			DealList:   "div.deal-list div.item",
			Title:      "h3.title",
			Link:       "a.deal-link",
			Thumbnail:  "img.thumbnail",
			PostedAt:   "span.date",
			PriceRegex: `\$([0-9,.]+)`,
			ThumbRegex: ``, // If needed
		},
		CustomHandlers:     customHandlers,
		ElementTransformers: elementTransformers,
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 5)
		},
	}, cacheSvc)
}
```

3. Add the crawler to `factory.go`:

```go
func createConfigurableCrawlers(cfg config.Config, cacheSvc cache.CacheService) []Crawler {
	var crawlers []Crawler
	
	// Add existing crawlers
	crawlers = append(crawlers, NewClienCrawler(cfg, cacheSvc))
	...
	
	// Add your new crawler
	crawlers = append(crawlers, NewMySiteCrawler(cfg, cacheSvc))

	return crawlers
}
```

4. Add the site URL to `config.go` and `.env`

### Using Custom Handlers

Custom handlers allow specialized processing for any element. Use them when the default extraction isn't sufficient:

```go
// Define a custom handler for complex posted dates
CustomHandlers{
	ElementHandlers: map[string]CustomElementHandlerFunc{
		"postedAt": func(s *goquery.Selection) string {
			// Detect the format and process accordingly
			rawDate := s.Find(".date-field").Text()
			if strings.Contains(rawDate, "ago") {
				// Convert relative date to absolute
				return convertRelativeDate(rawDate)
			}
			return strings.TrimSpace(rawDate)
		},
		"title": func(s *goquery.Selection) string {
			// Combine multiple elements for title
			mainTitle := s.Find(".main-title").Text()
			subTitle := s.Find(".subtitle").Text()
			return strings.TrimSpace(mainTitle + " - " + subTitle)
		},
	},
}
```

### Using Element Transformers

Element transformers modify selections before text extraction. They're useful for removing unwanted elements:

```go
// Remove elements that interfere with clean text extraction
ElementTransformers{
	RemoveElements: []ElementRemoval{
		// Remove price information from title
		{Selector: "span.price-tag", ApplyToPath: "title"},
		
		// Remove icons from posted date
		{Selector: "i.icon", ApplyToPath: "postedAt"},
		{Selector: "span.label", ApplyToPath: "postedAt"},
	},
}
```

### Scalability

This architecture provides the following benefits:

- **Easy to add new sites**: Add new crawlers using just configuration  
- **Improved maintainability**: Reuse shared logic and remove duplication  
- **Highly extensible**: Custom handlers and transformers for special cases
- **Testable design**: Easy to test due to configuration-based crawling  

## License

MIT License