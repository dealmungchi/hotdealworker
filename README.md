# HotDeal Worker

A robust Go application that crawls multiple e-commerce and community websites for hot deals and publishes them to Redis streams in real-time.

## Key Features

- **Concurrent Multi-site Crawling**: Parallel processing of multiple hot deal sites
- **Rate Limiting Prevention**: Effective crawling speed control using Memcached
- **Flexible Crawler Architecture**: Configuration-driven crawlers that reuse common functionality
- **Headless Browser Support**: Processing for sites requiring JavaScript execution
- **Redis Stream Publishing**: Scalable real-time data distribution
- **Memory Optimization**: Automatic stream trimming for efficient memory management

## Supported Sites

| Site | Status | URL |
|------|--------|-----|
| FMKorea(펨코) | Supported | https://www.fmkorea.com |
| Damoang(다모앙) | Supported | https://damoang.net |
| Arca Live(아카라이브) | Supported | https://arca.live |
| Quasar Zone(퀘이사존) | Supported | https://quasarzone.com |
| Coolandjoy(쿨앤조이) | Supported | https://coolenjoy.net |
| Clien(클리앙) | Supported | https://www.clien.net |
| Ppomppu(뽐뿌) | Supported | https://www.ppomppu.co.kr |
| Ppomppu English(해외뽐뿌) | Supported | https://www.ppomppu.co.kr |
| Ruliweb(루리웹) | Supported | https://bbs.ruliweb.com |
| Dealbada(딜바다) | Supported | https://www.dealbada.com |
| Missycoupons(미씨쿠폰) | Supported | https://www.missycoupons.com |
| Malltail(몰테일) | Supported | https://post.malltail.com |
| Bbasak(빠삭) | Supported | https://bbasak.com |
| City(시티) | Supported | https://www.city.kr |
| Eomisae(어미새) | Supported | https://eomisae.co.kr |
| Zod(조드) | Supported | https://zod.kr |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| REDIS_ADDR | Redis server address | localhost:6379 |
| REDIS_DB | Redis database number | 0 |
| REDIS_STREAM | Redis stream prefix | streamHotdeals |
| REDIS_STREAM_COUNT | Redis stream count | 1 |
| REDIS_STREAM_MAX_LENGTH | Maximum entries per Redis stream | 500 |
| MEMCACHE_ADDR | Memcached server address | localhost:11211 |
| CRAWL_INTERVAL_SECONDS | Crawling interval in seconds | 60 |
| USE_CHROME_DB | Enable headless browser for JavaScript sites | false |
| CHROME_DB_ADDR | ChromeDB service address | http://localhost:3000 |
| *_URL | Site-specific crawling URLs | Default site URLs |

## Installation and Usage

### Basic Setup

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

### Docker Deployment

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
    "provider": "Provider"
  },
  ...
]
```

## Testing

```bash
# Run all tests
make test

# Run unit tests only
make unit-test

# Run integration tests only
make integration-test
```

## Project Structure

- `config/`: Application configuration
- `services/`: Service layer (cache, publisher, worker)
- `internal/crawler/`: Crawler interface and implementations
- `helpers/`: Utility functions (HTTP, logging, etc.)

## Architecture

### Streaming Architecture

HotDeal Worker uses Redis Streams for publishing hot deal data:

1. **Multiple Streams**: Distributes data across multiple Redis Streams for improved load balancing
   - `REDIS_STREAM_COUNT` configuration controls number of streams
   - Messages are randomly assigned to streams

2. **Stream Trimming**: Automatically manages memory usage
   - Each stream is trimmed after every crawling cycle
   - `REDIS_STREAM_MAX_LENGTH` controls maximum entries per stream
   - Prevents unbounded growth of Redis memory usage

3. **Base64 Encoding**: All messages are Base64 encoded for consistent storage
   - Preserves binary data and special characters
   - Simplifies client processing

### Crawler Architecture

HotDeal Worker implements a modular crawler architecture:

1. **BaseCrawler**: Provides shared functionality across all crawlers
   - Rate limiting handling
   - URL resolution (relative to absolute)
   - Thumbnail image processing
   - Price extraction

2. **UnifiedCrawler**: Flexible configuration-driven crawler
   - Create site-specific crawlers through configuration
   - CSS selector-based operation
   - Support for custom handlers and element transformations
   - Reusable modular components

3. **Chrome-Enabled Crawling**: Headless browser-based crawler
   - Handles sites requiring JavaScript execution
   - Uses ChromeDB for page rendering
   - Processes fully rendered DOM content
   - Configurable with same selector approach as standard crawlers

4. **Site-Specific Crawlers**: Handle unique site requirements
   - Each crawler in separate file for better organization
   - Configuration-based approach for reusing common logic
   - Custom handlers for special extraction requirements
   - Option to use either standard HTTP or Chrome-based crawling

### Adding a New Crawler

To add a new crawler:

1. Create a new file in `internal/crawler` named after the site (e.g., `mynewsite.go`)
2. Implement a constructor function that returns a `UnifiedCrawler`:

```go
package crawler

import (
	"strings"

	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// NewMySiteCrawler creates a crawler for MySite
func NewMySiteCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.MySiteURL, // Add to config struct
		CacheKey:     "mysite_rate_limited",
		BlockTime:    500,
		BaseURL:      "https://www.mysite.com",
		Provider:     "MySite",
		UseChrome:    false, // Use standard HTTP crawler
		ChromeDBAddr: cfg.ChromeDBAddr, // For sites requiring JavaScript
		Selectors: Selectors{
			DealList:    "div.deal-list div.item",
			Title:       "h3.title",
			Link:        "a.deal-link",
			Thumbnail:   "img.thumbnail",
			PostedAt:    "span.date",
			PriceRegex:  `\$([0-9,.]+)`,
			ThumbRegex:  ``, // If needed
			ClassFilter: "", // Optional: filter out items with this class
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 5)
		},
	}, cacheSvc)
}
```

3. Add the crawler to `factory.go`:

```go
func CreateCrawlers(cfg config.Config, cacheSvc cache.CacheService) []Crawler {
	crawlers := []Crawler{}
	
	// Add existing crawlers
	crawlers = append(crawlers, NewClienCrawler(cfg, cacheSvc))
	// ...
	
	// Add your new crawler
	crawlers = append(crawlers, NewMySiteCrawler(cfg, cacheSvc))

	return crawlers
}
```

4. Add the site URL to `config.go` and environment variables

### Custom Element Handlers

For sites with complex HTML structures, you can add custom element handlers to the `Selectors` struct:

```go
Selectors: Selectors{
    DealList:    "div.deal-list div.item",
    Title:       "h3.title",
    Link:        "a.deal-link",
    Thumbnail:   "img.thumbnail",
    PostedAt:    "span.date",
    PriceRegex:  `\$([0-9,.]+)`,
    
    // Custom handlers for specific elements
    TitleHandlers: []ElementHandler{
        func(s *goquery.Selection) string {
            // Custom logic for title extraction
            mainTitle := s.Find(".main-title").Text()
            subTitle := s.Find(".subtitle").Text()
            return strings.TrimSpace(mainTitle + " - " + subTitle)
        },
    },
    PostedAtHandlers: []ElementHandler{
        func(s *goquery.Selection) string {
            // Custom date handling
            rawDate := s.Find(".date-field").Text()
            if strings.Contains(rawDate, "ago") {
                // Convert relative date to absolute
                return convertRelativeDate(rawDate)
            }
            return strings.TrimSpace(rawDate)
        },
    },
}
```

### Scalability Benefits

This architecture provides several advantages:

- **Easy Site Addition**: Add new crawlers using just configuration
- **Improved Maintainability**: Reuse shared logic and eliminate duplication
- **High Extensibility**: Custom handlers and transformers for special cases
- **Testable Design**: Easy testing with configuration-based crawling

## License

MIT License
