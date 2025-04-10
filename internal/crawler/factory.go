package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/helpers"
	"sjsage522/hotdealworker/services/cache"
)

// CreateCrawlers creates all the crawlers based on the configuration
func CreateCrawlers(cfg config.Config, cacheSvc cache.CacheService) []Crawler {
	// Use only configurable crawlers
	crawlers := createConfigurableCrawlers(cfg, cacheSvc)
	return crawlers
}

// createConfigurableCrawlers creates crawlers with the configurable approach
func createConfigurableCrawlers(cfg config.Config, cacheSvc cache.CacheService) []Crawler {
	// Create crawlers from individual crawler files
	var crawlers []Crawler

	// Add each crawler
	crawlers = append(crawlers, NewClienCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewRuliwebCrawler(cfg, cacheSvc))

	// FMKorea는 rate limiting 문제로 ChromeDB 사용
	if cfg.UseChromeDB {
		fmkoreaConfig := CrawlerConfig{
			URL:       cfg.FMKoreaURL + "/hotdeal",
			CacheKey:  "fmkorea_rate_limited",
			BlockTime: 300,
			BaseURL:   cfg.FMKoreaURL,
			Provider:  "FMKorea",
			Selectors: Selectors{
				DealList:   "ul li.li",
				Title:      "h3.title a",
				Link:       "h3.title a",
				Thumbnail:  "a img.thumb",
				PostedAt:   "div span.regdate",
				PriceRegex: `\(([0-9,]+원)\)$`,
			},
			ElementTransformers: ElementTransformers{
				RemoveElements: []ElementRemoval{
					{Selector: "span", ApplyToPath: "title"},
				},
			},
			IDExtractor: func(link string) (string, error) {
				return helpers.GetSplitPart(link, "/", 3)
			},
		}
		crawlers = append(crawlers, NewChromeDBCrawler(fmkoreaConfig, cacheSvc, cfg.ChromeDBAddr))
	} else {
		crawlers = append(crawlers, NewFMKoreaCrawler(cfg, cacheSvc))
	}

	// 다른 사이트에 ChromeDB 필요시 아래 코드 참고
	/*
		if cfg.UseChromeDB {
			// Create a crawler with ChromeDB
			siteConfig := CrawlerConfig{
				URL:       "your_url_here",
				CacheKey:  "site_rate_limited",
				BlockTime: 300,
				BaseURL:   "base_url_here",
				Provider:  "SiteName",
				Selectors: Selectors{
					DealList:   "selector_for_deals",
					Title:      "selector_for_title",
					Link:       "selector_for_link",
					Thumbnail:  "selector_for_thumbnail",
					PostedAt:   "selector_for_posted_time",
					PriceRegex: `your_regex_here`,
				},
				IDExtractor: func(link string) (string, error) {
					return helpers.GetSplitPart(link, "/", 3)
				},
			}
			crawlers = append(crawlers, NewChromeDBCrawler(siteConfig, cacheSvc, cfg.ChromeDBAddr))
		}
	*/

	crawlers = append(crawlers, NewPpomCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewPpomEnCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewQuasarCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewDamoangCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewArcaCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewCoolandjoyCrawler(cfg, cacheSvc))

	return crawlers
}
