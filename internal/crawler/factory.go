package crawler

import (
	"sjsage522/hotdealworker/config"
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
	crawlers = append(crawlers, NewFMKoreaCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewPpomCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewPpomEnCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewQuasarCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewDamoangCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewArcaCrawler(cfg, cacheSvc))
	crawlers = append(crawlers, NewCoolandjoyCrawler(cfg, cacheSvc))

	return crawlers
}
