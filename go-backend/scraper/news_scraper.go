package main

import "github.com/gocolly/colly"

type Article struct {
	comapany, url, text string
	bias, confidence    float64
}

func Scrape() {
	// Test
	companiesToScrape := []string{"microsoft", "nvidia"}
	visitedUrls := make(map[string]bool)

	c := colly.NewCollector(
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{Parallelism: 4, DomainGlob: "*"})

	// Setup the scraping object need to now setup the scraping and crawling process of what we will do upon reaching each website and such
}
