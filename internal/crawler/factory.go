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
	
	// Use ChromeDB for FMKorea if configured
	if cfg.UseChromeDB {
		// Create an FMKorea crawler with ChromeDB
		fmConfig := CrawlerConfig{
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
		crawlers = append(crawlers, NewChromeDBCrawler(fmConfig, cacheSvc, cfg.ChromeDBAddr))
	} else {
		crawlers = append(crawlers, NewFMKoreaCrawler(cfg, cacheSvc))
	}
	
	crawlers = append(crawlers, NewPpomCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewPpomEnCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewQuasarCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewDamoangCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewArcaCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewCoolandjoyCrawler(cfg, cacheSvc))

	return crawlers
}