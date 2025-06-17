package crawler

func CreateCrawlers() []Crawler {
	return []Crawler{
		*NewArcaCrawler(),
	}
}
