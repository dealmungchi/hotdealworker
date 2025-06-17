package internal

import (
	"github.com/dealmungchi/dealcrawler/services/cache"
	"github.com/dealmungchi/dealcrawler/services/proxy"
	"github.com/dealmungchi/dealcrawler/services/publisher"
)

// Dependencies holds all service dependencies
type Dependencies struct {
	Cache     cache.CacheService
	Publisher publisher.Publisher
	Proxy     proxy.ProxyManager
}
