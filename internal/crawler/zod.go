package crawler

import (
	"regexp"
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func NewZod(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	priceHandler := func(s *goquery.Selection) string {
		// 가격 정보가 있는 요소 선택
		priceSel := s.Find("div.app-list-meta.zod-board--deal-meta.tw-mt-1 span")
		if priceSel.Length() == 0 {
			return ""
		}

		// 원본 텍스트 가져오기
		priceText := strings.TrimSpace(priceSel.Text())

		// "가격:" 이후의 텍스트 추출
		priceIndex := strings.Index(priceText, "가격:")
		if priceIndex != -1 {
			// "가격:" 이후부터 "배송비:" 이전까지의 텍스트 추출
			priceStart := priceIndex + len("가격:")
			priceEnd := strings.Index(priceText[priceStart:], "배송비:")

			var priceValue string
			if priceEnd != -1 {
				// "배송비:" 텍스트가 있는 경우
				priceValue = strings.TrimSpace(priceText[priceStart : priceStart+priceEnd])
			} else {
				// "배송비:" 텍스트가 없는 경우
				priceValue = strings.TrimSpace(priceText[priceStart:])
			}

			// 숫자와 쉼표만 포함된 가격 추출
			priceRegex := regexp.MustCompile(`[0-9,]+`)
			priceMatch := priceRegex.FindString(priceValue)

			if priceMatch != "" {
				return priceMatch + "원"
			}
			return priceValue
		}

		// "가격:" 텍스트가 없는 경우, 숫자와 쉼표, 그리고 '원'으로 끝나는 부분 추출
		priceRegex := regexp.MustCompile(`[0-9,]+원`)
		priceMatch := priceRegex.FindString(priceText)
		if priceMatch != "" {
			return priceMatch
		}

		// 숫자만 추출
		numRegex := regexp.MustCompile(`[0-9,]+`)
		numMatch := numRegex.FindString(priceText)
		if numMatch != "" {
			return numMatch + "원"
		}

		return priceText
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.ZodURL + "/deal",
		CacheKey:     "zod_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.ZodURL,
		Provider:     "Zod",
		UseChrome:    true,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:      "ul.app-board-template-list.zod-board-list--deal li",
			Title:         "a.tw-flex-1 div.tw-flex-1 div.app-list-title.tw-flex-wrap span.tw-mr-1.app-list-title-item",
			Link:          "a.tw-flex-1",
			Thumbnail:     "a.tw-flex-1 div.app-thumbnail img",
			Category:      "span.zod-board--deal-meta-category",
			ClassFilter:   "notice zod-board-list-deal-ended",
			PriceHandlers: []ElementHandler{priceHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "/", 4)
		},
	}, cacheSvc)
}
