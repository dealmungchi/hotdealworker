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
| REDIS_CHANNEL | Redis publish channel | hotdeals |
| MEMCACHE_ADDR | Memcached server address | localhost:11211 |
| CRAWL_INTERVAL_SECONDS | Crawling interval (in seconds) | 60 |
| FMKOREA_URL | FMKorea crawling URL | http://www.fmkorea.com/hotdeal |
| DAMOANG_URL | Damoang crawling URL | https://damoang.net/economy |
| ARCA_URL | Arca Live crawling URL | https://arca.live/b/hotdeal |
| QUASAR_URL | Quasar Zone crawling URL | https://quasarzone.com/bbs/qb_saleinfo |
| COOLANDJOY_URL | Coolandjoy crawling URL | https://coolenjoy.net/bbs/jirum |
| CLIEN_URL | Clien crawling URL | https://www.clien.net/service/board/jirum |
| PPOM_URL | Ppomppu crawling URL | https://www.ppomppu.co.kr/zboard/zboard.php?id=ppomppu |
| PPOMEN_URL | Ppomppu English crawling URL | https://www.ppomppu.co.kr/zboard/zboard.php?id=ppomppu4 |
| RULIWEB_URL | Ruliweb crawling URL | https://bbs.ruliweb.com/market/board/1020?view=thumbnail&page=1 |

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
   - Reusable modular components  

3. **Site-specific Crawlers**: Handle site-specific needs  
   - Config-based approach: reuse common logic with only config differences  

### Scalability

This architecture provides the following benefits:

- **Easy to add new sites**: Add new crawlers using just configuration  
- **Improved maintainability**: Reuse shared logic and remove duplication  
- **Testable design**: Easy to test due to configuration-based crawling  

## License

MIT License

