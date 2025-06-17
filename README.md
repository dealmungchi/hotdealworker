# Deal Crawler

핫딜 크롤러 애플리케이션입니다. 다양한 쇼핑몰과 커뮤니티에서 핫딜 정보를 수집하고 Redis 스트림으로 발행합니다.

## 아키텍처

### 서비스 컨테이너 패턴

이 애플리케이션은 서비스 컨테이너 패턴을 사용하여 의존성 관리를 단순화했습니다.

```
services/
├── container.go         # 전역 서비스 컨테이너
├── cache/              # 캐시 서비스 (Memcache)
│   ├── cache.go        # 인터페이스 정의
│   └── memcache.go     # Memcache 구현
├── publisher/          # 메시지 발행 서비스 (Redis)
│   ├── publisher.go    # 인터페이스 정의
│   └── redis.go        # Redis 구현
├── proxy/              # 프록시 관리 서비스
│   └── proxy.go        # SOCKS5 프록시 관리
└── worker/             # 워커 서비스
    ├── types.go        # 타입 정의
    └── worker.go       # 크롤링 워커
```

### 서비스 초기화

```go
// main.go에서 서비스 초기화
if err := services.Initialize(ctx); err != nil {
    log.Fatal().Err(err).Msg("Failed to initialize services")
}
defer services.Cleanup()
```

### 서비스 접근

애플리케이션의 어느 곳에서든 서비스에 접근할 수 있습니다:

```go
// 캐시 서비스 사용
cache := services.GetCache()
cache.Set("key", []byte("value"), time.Hour)

// 퍼블리셔 서비스 사용
publisher := services.GetPublisher()
publisher.Publish("deal_id", dealJSON)

// 프록시 서비스 사용
proxy := services.GetProxy()
fastestProxy, err := proxy.GetFastestProxy()
```

## 컴포넌트

### 1. 캐시 서비스 (Memcache)

```go
type CacheService interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte, expiration time.Duration) error
    Delete(key string) error
}
```

### 2. 퍼블리셔 서비스 (Redis)

```go
type Publisher interface {
    Publish(key string, message []byte) error
    TrimStreams() error
    Close() error
}
```

### 3. 프록시 매니저

```go
type ProxyManager interface {
    UpdateProxies() error
    GetFastestProxy() (*ProxyInfo, error)
    GetTopProxies(n int) []ProxyInfo
}
```

### 4. 크롤러

```go
type Crawler struct {
    CrawlerConfig
}

func (c *Crawler) Crawl() ([]HotDeal, error)
```

## 설정

환경변수를 통해 설정할 수 있습니다. 모든 환경변수는 선택사항이며, 설정하지 않으면 기본값이 사용됩니다:

```bash
# Memcache 설정 (기본값: localhost:11211)
MEMCACHE_ADDR=localhost:11211

# Redis 설정
REDIS_ADDR=localhost:6379         # 기본값: localhost:6379
REDIS_DB=0                        # 기본값: 0
REDIS_STREAM=streamHotdeals       # 기본값: streamHotdeals
REDIS_STREAM_COUNT=1              # 기본값: 1
REDIS_STREAM_MAX_LENGTH=500       # 기본값: 500

# 크롤링 설정
CRAWL_INTERVAL_SECONDS=60         # 기본값: 60초

# 아르카 URL 설정
ARCA_URL=https://arca.live

# 로그 레벨 설정
LOG_LEVEL=INFO                    # 기본값: INFO
```

### 서비스 장애 대응

애플리케이션은 개별 서비스 장애에 대해 resilient하게 설계되었습니다:

- **프록시 서비스 실패**: 프록시를 가져올 수 없어도 애플리케이션은 계속 실행되며, 직접 연결을 사용합니다
- **캐시 서비스 실패**: 캐시가 사용 불가능해도 크롤링은 계속 진행됩니다
- **환경변수 미설정**: 모든 환경변수에 대해 합리적인 기본값이 제공됩니다

## 사용법

### 애플리케이션 시작

```bash
go run main.go
```

### 새로운 크롤러 추가

1. `crawler/` 디렉토리에 새 크롤러 파일 생성
2. `CrawlerConfig`를 사용하여 크롤러 설정
3. `crawler/factory.go`에 크롤러 추가

```go
func CreateCrawlers() []Crawler {
    return []Crawler{
        *NewArcaCrawler(),
        *NewYourNewCrawler(), // 새 크롤러 추가
    }
}
```

### 서비스 사용 예시

크롤러에서 서비스 사용:

```go
func (c *Crawler) Crawl() ([]HotDeal, error) {
    // 캐시 확인
    cache := services.GetCache()
    cacheKey := fmt.Sprintf("crawl:%s:last_run", c.Provider)
    
    if cachedData, err := cache.Get(cacheKey); err == nil {
        // 캐시된 데이터 처리
    }
    
    // 크롤링 로직...
    
    // 결과 발행
    if len(deals) > 0 {
        publisher := services.GetPublisher()
        for _, deal := range deals {
            dealJSON, _ := json.Marshal(deal)
            publisher.Publish(deal.Id, dealJSON)
        }
    }
    
    return deals, nil
}
```

## 장점

1. **중앙화된 서비스 관리**: 모든 서비스가 하나의 컨테이너에서 관리됨
2. **의존성 주입 간소화**: 생성자 파라미터 대신 전역 접근 방식 사용
3. **테스트 용이성**: 인터페이스 기반 설계로 목킹 가능
4. **확장성**: 새로운 서비스 추가가 쉬움
5. **안전성**: 서비스 초기화 상태 체크 및 스레드 안전성 보장

## 빌드 & 배포

```bash
# 빌드
go build -o dealcrawler main.go

# 실행
./dealcrawler
```

## 문제 해결

### 일반적인 오류와 해결 방법

1. **"Failed to initialize services"**
   - Redis나 Memcache 서버가 실행 중인지 확인하세요
   - 환경변수의 주소와 포트가 올바른지 확인하세요
   - 네트워크 연결을 확인하세요

2. **"no working proxies available"**
   - 정상적인 동작입니다. 애플리케이션은 프록시 없이도 계속 실행됩니다
   - 프록시 서비스(spys.me)가 일시적으로 사용 불가능할 수 있습니다

3. **"Failed to load .env file"**
   - 정상적인 동작입니다. 환경변수가 설정되지 않은 경우 기본값이 사용됩니다
   - 필요한 경우 `.env.example`을 참고하여 `.env` 파일을 생성하세요

### 로그 레벨 조정

```bash
# 디버그 로그 활성화
LOG_LEVEL=DEBUG go run main.go

# 오류만 표시
LOG_LEVEL=ERROR go run main.go
```

### 개발 모드 실행

서비스 없이 테스트하려면:

```bash
# 최소 설정으로 실행
CRAWL_INTERVAL_SECONDS=10 go run main.go
```

## 의존성

- `github.com/bradfitz/gomemcache` - Memcache 클라이언트
- `github.com/redis/go-redis/v9` - Redis 클라이언트
- `github.com/rs/zerolog` - 구조화된 로깅
- `github.com/joho/godotenv` - 환경변수 로딩
