# HotDeal Worker

핫딜 정보를 크롤링하여 Redis Stream으로 발행하는 Go 워커 애플리케이션입니다.

## 특징

- 다양한 핫딜 사이트 지원 (16개 사이트)
- 병렬 크롤링
- Memcache를 이용한 Rate Limiting
- Redis Stream을 통한 실시간 데이터 발행
- ChromeDB 지원 (JavaScript 렌더링이 필요한 사이트)
- 로깅 (zerolog)
- Graceful Shutdown
- 환경 변수 기반 설정

## 아키텍처

### 디렉토리 구조

```
.
├── config/             # 설정 관리
├── internal/
│   └── crawler/       # 크롤러 구현
├── pkg/
│   └── errors/        # 커스텀 에러 타입
├── services/
│   ├── cache/         # Memcache 서비스
│   ├── publisher/     # Redis 발행 서비스
│   └── worker/        # 워커 서비스
├── helpers/           # 유틸리티 함수
└── logger/            # 로깅 서비스
```

### 주요 컴포넌트

- **Worker**: 크롤링 주기 관리 및 조정
- **Crawler**: 각 사이트별 크롤링 로직
- **Publisher**: Redis Stream으로 데이터 발행
- **Cache**: Rate Limiting을 위한 캐시

## 설치 및 실행

### 사전 요구사항

- Go 1.24.1 이상
- Redis
- Memcache
- ChromeDB (선택사항)

### 환경 변수

```bash
# Redis 설정
REDIS_ADDR=localhost:6379
REDIS_DB=0
REDIS_STREAM=streamHotdeals
REDIS_STREAM_COUNT=1
REDIS_STREAM_MAX_LENGTH=500

# Memcache 설정
MEMCACHE_ADDR=localhost:11211

# 크롤링 설정
CRAWL_INTERVAL_SECONDS=60
USE_CHROME_DB=false
CHROME_DB_ADDR=http://localhost:3000

# 환경 설정
HOTDEAL_ENVIRONMENT=development
LOG_LEVEL=debug

# 크롤러 활성화 (기본값: true)
CRAWLER_FMKOREA_ENABLED=true
CRAWLER_DAMOANG_ENABLED=true
# ... 기타 크롤러 설정
```

### 실행

```bash
# 개발 환경
go run main.go

# 프로덕션 환경
go build -o hotdealworker
./hotdealworker

# Docker
docker-compose up
```

## 지원 사이트

- FM Korea
- 다모앙
- 아카라이브
- 퀘이사존
- 쿨앤조이
- 클리앙
- 뽐뿌
- 루리웹
- 딜바다
- 미씨쿠폰
- 몰테일
- 빠삭
- 시티
- 어미새
- ZOD

## 개발

### 테스트

```bash
# 단위 테스트
go test ./...

# 통합 테스트 (Redis/Memcache 필요)
go test -v ./integration_test.go
```

### 새로운 크롤러 추가

1. `internal/crawler/` 디렉토리에 새로운 크롤러 파일 생성
2. `UnifiedCrawler`를 사용하여 구현
3. `factory.go`에 크롤러 생성자 추가
4. 환경 변수에 URL 및 활성화 설정 추가

## 모니터링
- 크롤링 주기별 성능 (소요 시간, 수집된 딜 수)
- 크롤러별 성공/실패 상태
- Rate Limiting 발생
- Redis 발행 상태

