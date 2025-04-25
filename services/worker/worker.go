package worker

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"time"

	"sjsage522/hotdealworker/internal/crawler"
	"sjsage522/hotdealworker/logger"
	"sjsage522/hotdealworker/services/publisher"
)

// Worker handles the crawling and publishing process
type Worker struct {
	ctx           context.Context
	crawlers      []crawler.Crawler
	publisher     publisher.Publisher
	crawlInterval time.Duration
}

// NewWorker creates a new worker
func NewWorker(
	ctx context.Context,
	crawlers []crawler.Crawler,
	pub publisher.Publisher,
	crawlInterval time.Duration,
) *Worker {
	return &Worker{
		ctx:           ctx,
		crawlers:      crawlers,
		publisher:     pub,
		crawlInterval: crawlInterval,
	}
}

// Start starts the worker process
func (w *Worker) Start() {
	for {
		start := time.Now()
		w.runCrawlers()
		elapsed := time.Since(start)
		logger.Info("크롤링 소요 시간: %s", elapsed)
		time.Sleep(w.crawlInterval)
	}
}

// runCrawlers runs all the crawlers in parallel and then trims the streams
func (w *Worker) runCrawlers() {
	var wg sync.WaitGroup
	for _, c := range w.crawlers {
		wg.Add(1)
		go func(c crawler.Crawler) {
			defer wg.Done()
			w.crawlAndPublish(c)
		}(c)
	}
	wg.Wait()

	// Trim all streams after crawling
	if err := w.publisher.TrimStreams(); err != nil {
		logger.Error("StreamTrimming", err)
	}
}

// crawlAndPublish crawls deals from a crawler and publishes them
func (w *Worker) crawlAndPublish(c crawler.Crawler) {
	crawlerName := c.GetName()
	if crawlerName == "" {
		crawlerName = reflect.TypeOf(c).Elem().Name()
	}

	deals, err := c.FetchDeals()
	if err != nil {
		logger.Error(crawlerName, err)
		return
	}

	for _, deal := range deals {
		dealData, err := json.Marshal(deal)
		if err != nil {
			logger.Error(crawlerName, err)
			return
		}

		if err := w.publisher.Publish(c.GetProvider(), dealData); err != nil {
			logger.Error(crawlerName, err)
		}
	}

	w.newMethod(deals, crawlerName)
}

func (w *Worker) newMethod(deals []crawler.HotDeal, crawlerName string) {
	if os.Getenv("HOTDEAL_ENVIRONMENT") != "production" {
		for i, deal := range deals[:5] {
			var loggableDeal map[string]interface{}
			dealData, err := json.MarshalIndent(deal, "", "  ")
			if err != nil {
				logger.Error(crawlerName, err)
				continue
			}
			if err := json.Unmarshal(dealData, &loggableDeal); err != nil {
				logger.Error(crawlerName, err)
				continue
			}
			if _, exists := loggableDeal["thumbnail"]; exists {
				loggableDeal["thumbnail"] = "OK"
			}
			loggableDealData, err := json.MarshalIndent(loggableDeal, "", "  ")
			if err != nil {
				logger.Error(crawlerName, err)
				continue
			}
			logger.Debug("크롤링 데이터 %d: %s", i+1, string(loggableDealData))
		}
	}
}
