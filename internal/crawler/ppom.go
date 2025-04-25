package crawler

import (
	"regexp"
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// NewPpomCrawler creates a Ppomppu crawler
func NewPpomCrawler(cfg config.Config, cacheSvc cache.CacheService) *UnifiedCrawler {
	priceHandler := func(element *goquery.Selection) string {
		// titile을 가져와서
		// [G마켓](10%캐시적립)온더바디 코튼풋 발을씻자 풋샴푸 쿨링 385ml 2개+레몬리필 500ml 2개 (15,720원/무료)
		// 이렇게 되어있는데 15,720원/무료 이렇게 뽑아내기

		priceText := element.Find("div.baseList-cover a").Text()
		re := regexp.MustCompile(`\d+,\d+원`)
		match := re.FindString(priceText)
		if match == "" {
			re = regexp.MustCompile(`\d+`)
			match = re.FindString(priceText)
			match += "원"
		}
		return strings.TrimSpace(match)
	}

	return NewUnifiedCrawler(CrawlerConfig{
		URL:          cfg.PpomURL + "/zboard/zboard.php?id=ppomppu",
		CacheKey:     "ppom_rate_limited",
		BlockTime:    500,
		BaseURL:      cfg.PpomURL + "/zboard/",
		Provider:     "Ppom",
		UseChrome:    false,
		ChromeDBAddr: cfg.ChromeDBAddr,
		Selectors: Selectors{
			DealList:      "tr.baseList.bbs_new1",
			Title:         "div.baseList-cover a.baseList-title",
			Link:          "div.baseList-cover a.baseList-title",
			Thumbnail:     "a.baseList-thumb img",
			PostedAt:      "time.baseList-time",
			PriceRegex:    `\(([0-9,]+원)\)$`,
			PriceHandlers: []ElementHandler{priceHandler},
		},
		IDExtractor: func(link string) (string, error) {
			return helpers.GetSplitPart(link, "no=", 1)
		},
	}, cacheSvc)
}
