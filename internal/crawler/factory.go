package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/services/cache"
)

// CreateCrawlers creates all the crawlers based on the configuration
func CreateCrawlers(cfg *config.Config, cacheSvc cache.CacheService) []Crawler {
	return []Crawler{
		NewFMKoreaCrawler(cfg.FMKoreaURL, cacheSvc),
		NewDamoangCrawler(cfg.DamoangURL, cacheSvc),
		NewArcaCrawler(cfg.ArcaURL, cacheSvc),
		NewQuasarCrawler(cfg.QuasarURL, cacheSvc),
		NewCoolandjoyCrawler(cfg.CoolandjoyURL, cacheSvc),
		NewClienCrawler(cfg.ClienURL, cacheSvc),
		NewPpomCrawler(cfg.PpomURL, cacheSvc),
		NewPpomEnCrawler(cfg.PpomEnURL, cacheSvc),
		NewRuliwebCrawler(cfg.RuliwebURL, cacheSvc),
	}
}
