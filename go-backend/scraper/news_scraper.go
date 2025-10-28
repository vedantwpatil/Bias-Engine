package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vedantwpatil/bias-engine/models"

	"github.com/gocolly/colly/v2"
)

func ScrapeCompanyNews(company string, maxArticles int) ([]models.Article, error) {
	articles := make([]models.Article, 0)

	c := colly.NewCollector(
		colly.AllowedDomains("news.google.com", "www.google.com"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)

	// Rate limiting
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       2 * time.Second,
		RandomDelay: 1 * time.Second,
	})

	// Extract article links from Google News
	c.OnHTML("article", func(e *colly.HTMLElement) {
		if len(articles) >= maxArticles {
			return
		}

		title := e.ChildText("h3, h4")
		link := e.ChildAttr("a", "href")

		if title != "" && link != "" {
			// Clean up Google News redirect link
			if strings.Contains(link, "./articles") {
				link = "https://news.google.com" + link[1:]
			}

			article := models.Article{
				Title:     title,
				URL:       link,
				Source:    extractSource(e.Text),
				Timestamp: time.Now(),
				Body:      extractPreview(e.Text), // Use preview for demo
			}

			articles = append(articles, article)
			log.Printf("Found article: %s", title)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request error: %v", err)
	})

	searchURL := fmt.Sprintf("https://news.google.com/search?q=%s+stock+news", company)
	log.Printf("Scraping: %s", searchURL)

	err := c.Visit(searchURL)
	if err != nil {
		return nil, err
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles found for company: %s", company)
	}

	return articles, nil
}

func extractSource(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if len(line) > 3 && len(line) < 50 && !strings.Contains(line, "ago") {
			return strings.TrimSpace(line)
		}
	}
	return "Unknown"
}

func extractPreview(text string) string {
	// Take first 200 chars as preview for demo purposes
	// In production, you'd fetch full article content
	if len(text) > 200 {
		return text[:200]
	}
	return text
}
