package crawler

import "os"

func NewArcaCrawler() *Crawler {
	url := os.Getenv("ARCA_URL")

	return NewCrawler(
		CrawlerConfig{
			DealURL:   url + "/b/hotdeal",
			URL:       url,
			Provider:  "Arca",
			UseChrome: false,
			Selectors: Selectors{
				DealList: Selector{
					tag: "div.list-table.hybrid div.vrow.hybrid",
				},
				Title: Selector{
					tag: "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
				},
				Link: Selector{
					tag: "div.vrow-inner div.vrow-top.deal a.title.hybrid-title",
				},
				Thumbnail: Selector{
					tag: "a.title.preview-image div.vrow-preview img",
				},
				PostedAt: Selector{
					tag: "span.col-time time",
				},
				Price: Selector{
					tag: "",
				},
			},
		},
	)
}
