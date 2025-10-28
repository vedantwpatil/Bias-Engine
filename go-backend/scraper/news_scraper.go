package scraper

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vedantwpatil/bias-engine/models"
)

type RSSFeed struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			PubDate     string `xml:"pubDate"`
			Description string `xml:"description"`
			Source      struct {
				Text string `xml:",chardata"`
			} `xml:"source"`
		} `xml:"item"`
	} `xml:"channel"`
}

func ScrapeCompanyNews(company string, maxArticles int) ([]models.Article, error) {
	articles := make([]models.Article, 0)

	// Use Google News RSS feed instead
	rssURL := fmt.Sprintf("https://news.google.com/rss/search?q=%s+stock&hl=en-US&gl=US&ceid=US:en", company)

	log.Printf("Fetching RSS feed: %s", rssURL)

	resp, err := http.Get(rssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("RSS feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse RSS: %v", err)
	}

	for i, item := range feed.Channel.Items {
		if i >= maxArticles {
			break
		}

		article := models.Article{
			Title:     cleanTitle(item.Title),
			URL:       item.Link,
			Source:    extractSource(item.Title),
			Body:      stripHTML(item.Description),
			Timestamp: time.Now(),
		}

		articles = append(articles, article)
		log.Printf("Found article: %s", article.Title)
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles found for company: %s", company)
	}

	return articles, nil
}

func cleanTitle(title string) string {
	if idx := strings.LastIndex(title, " - "); idx > 0 {
		return strings.TrimSpace(title[:idx])
	}
	return title
}

func extractSource(title string) string {
	if idx := strings.LastIndex(title, " - "); idx >= 0 && idx < len(title)-3 {
		source := strings.TrimSpace(title[idx+3:])
		if source != "" {
			return source
		}
	}
	return "Google News"
}

func stripHTML(s string) string {
	// Simple HTML tag removal (or use a library like bluemonday)
	result := ""
	inTag := false
	for _, char := range s {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			result += string(char)
		}
	}
	return result
}
