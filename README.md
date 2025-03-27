# HotDeal Worker

핫딜 정보를 크롤링하여 Redis에 publish 하는 워커 프로그램입니다.

## Features

- 핫딜 사이트 동시 크롤링
- Rate limiting 방지 및 처리 (Memcached 캐시)
- JSON to base64encoded 데이터 발행 (Redis pub/sub)
- 환경 변수 설정
- 로깅/에러 처리

## Support Sites

- FMKorea (FM코리아) - **doing**
- Damoang (다모아)
- Arca Live (아카라이브)
- Quasar Zone (퀘이사존)
- Coolandjoy (쿨앤조이)
- Clien (클리앙)
- Ppomppu (뽐뿌)
- Ppomppu English (뽐뿌 영문)
- Ruliweb (루리웹)

## Environments

| 변수 | 설명 | 기본값 |
|------|------|--------|
| REDIS_ADDR | Redis 서버 주소 | localhost:6379 |
| REDIS_DB | Redis DB 번호 | 0 |
| REDIS_CHANNEL | Redis 발행 채널 | hotdeals |
| MEMCACHE_ADDR | Memcached 서버 주소 | localhost:11211 |
| CRAWL_INTERVAL_SECONDS | 크롤링 간격 (초) | 60 |
| FMKOREA_URL | FM코리아 크롤링 URL | http://www.fmkorea.com/hotdeal |
| DAMOANG_URL | 다모아 크롤링 URL | https://damoang.net/economy |
| ARCA_URL | 아카라이브 크롤링 URL | https://arca.live/b/hotdeal |
| QUASAR_URL | 퀘이사존 크롤링 URL | https://quasarzone.com/bbs/qb_saleinfo |
| COOLANDJOY_URL | 쿨앤조이 크롤링 URL | https://coolenjoy.net/bbs/jirum |
| CLIEN_URL | 클리앙 크롤링 URL | https://www.clien.net/service/board/jirum |
| PPOM_URL | 뽐뿌 크롤링 URL | https://www.ppomppu.co.kr/zboard/zboard.php?id=ppomppu |
| PPOMEN_URL | 뽐뿌 영문 크롤링 URL | https://www.ppomppu.co.kr/zboard/zboard.php?id=ppomppu4 |
| RULIWEB_URL | 루리웹 크롤링 URL | https://bbs.ruliweb.com/market/board/1020?view=thumbnail&page=1 |

## Install

### Basic

```bash
# 레포지토리 클론
git clone https://github.com/yourusername/hotdealworker.git
cd hotdealworker

# 의존성 설치
go mod download

# 빌드
go build -o hotdealworker

# 실행
./hotdealworker
```

### Docker

```bash
# Docker Compose로 실행
docker compose up -d
```

## Message Structure

Redis에 publish 하는 메시지는 다음과 같은 구조의 JSON 배열을 Base64로 인코딩한 형태입니다:
- Json Data worked by each crawler
```json
[
  {
    "title": "상품명",
    "link": "상품 링크",
    "price": "가격",
    "thumbnail": "썸네일 이미지 URL",
    "posted_at": "게시 일시"
  },
  ...
]
```

## Tests

```bash
# 모든 테스트 실행
make test

# 단위 테스트만 실행
make unit-test

# 통합 테스트만 실행
make integration-test
```

## Modules

- `config/`: 애플리케이션 설정 관련 코드
- `services/`: 서비스 계층 (캐시, 발행자, 워커)
- `internal/crawler/`: 크롤러 인터페이스 및 구현체
- `helpers/`: 유틸리티 함수 (HTTP, 로깅 등)

## License

MIT License