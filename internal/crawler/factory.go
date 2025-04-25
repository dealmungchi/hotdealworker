package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/services/cache"
)

// CreateCrawlers creates all the crawlers based on the configuration
func CreateCrawlers(cfg config.Config, cacheSvc cache.CacheService) []Crawler {
	return []Crawler{
		NewClienCrawler(cfg, cacheSvc),
		NewRuliwebCrawler(cfg, cacheSvc),
		NewFMKoreaCrawler(cfg, cacheSvc),
		NewPpomCrawler(cfg, cacheSvc),
		NewPpomEnCrawler(cfg, cacheSvc),
		NewQuasarCrawler(cfg, cacheSvc),
		NewDamoangCrawler(cfg, cacheSvc),
		NewArcaCrawler(cfg, cacheSvc),
		NewCoolandjoyCrawler(cfg, cacheSvc),
		NewDealbadaCrawler(cfg, cacheSvc),
		NewMissycoupons(cfg, cacheSvc),
		NewMalltail(cfg, cacheSvc),
		NewBbasak(cfg, cacheSvc),
		NewCity(cfg, cacheSvc),
		NewEomisae(cfg, cacheSvc),
		NewZod(cfg, cacheSvc),
	}
}
