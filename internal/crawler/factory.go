package crawler

import (
	"sjsage522/hotdealworker/config"
	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/services/cache"
)

// CreateCrawlers creates enabled crawlers based on the configuration
func CreateCrawlers(cfg *config.Config, cacheSvc cache.CacheService) []Crawler {
	log := logger.Default.WithField("component", "crawler_factory")

	crawlers := []Crawler{}

	// Map of crawler constructors
	crawlerConstructors := map[string]func(*config.Config, cache.CacheService) Crawler{
		"clien":        func(c *config.Config, cs cache.CacheService) Crawler { return NewClienCrawler(*c, cs) },
		"ruliweb":      func(c *config.Config, cs cache.CacheService) Crawler { return NewRuliwebCrawler(*c, cs) },
		"fmkorea":      func(c *config.Config, cs cache.CacheService) Crawler { return NewFMKoreaCrawler(*c, cs) },
		"ppom":         func(c *config.Config, cs cache.CacheService) Crawler { return NewPpomCrawler(*c, cs) },
		"ppomen":       func(c *config.Config, cs cache.CacheService) Crawler { return NewPpomEnCrawler(*c, cs) },
		"quasar":       func(c *config.Config, cs cache.CacheService) Crawler { return NewQuasarCrawler(*c, cs) },
		"damoang":      func(c *config.Config, cs cache.CacheService) Crawler { return NewDamoangCrawler(*c, cs) },
		"arca":         func(c *config.Config, cs cache.CacheService) Crawler { return NewArcaCrawler(*c, cs) },
		"coolandjoy":   func(c *config.Config, cs cache.CacheService) Crawler { return NewCoolandjoyCrawler(*c, cs) },
		"dealbada":     func(c *config.Config, cs cache.CacheService) Crawler { return NewDealbadaCrawler(*c, cs) },
		"missycoupons": func(c *config.Config, cs cache.CacheService) Crawler { return NewMissycoupons(*c, cs) },
		"malltail":     func(c *config.Config, cs cache.CacheService) Crawler { return NewMalltail(*c, cs) },
		"bbasak":       func(c *config.Config, cs cache.CacheService) Crawler { return NewBbasak(*c, cs) },
		"city":         func(c *config.Config, cs cache.CacheService) Crawler { return NewCity(*c, cs) },
		"eomisae":      func(c *config.Config, cs cache.CacheService) Crawler { return NewEomisae(*c, cs) },
		"zod":          func(c *config.Config, cs cache.CacheService) Crawler { return NewZod(*c, cs) },
	}

	// Create crawlers based on configuration
	for name, crawlerCfg := range cfg.Crawlers {
		if !crawlerCfg.Enabled {
			log.Debug().
				Str("crawler", name).
				Msg("Crawler disabled")
			continue
		}

		constructor, exists := crawlerConstructors[name]
		if !exists {
			log.Warn().
				Str("crawler", name).
				Msg("Unknown crawler type")
			continue
		}

		crawler := constructor(cfg, cacheSvc)
		crawlers = append(crawlers, crawler)

		log.Info().
			Str("crawler", name).
			Str("url", crawlerCfg.URL).
			Msg("Crawler created")
	}

	return crawlers
}
