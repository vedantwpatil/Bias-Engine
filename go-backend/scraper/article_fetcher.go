package scraper

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func FetchFullArticle(url string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract main content (heuristic approach)
	var content strings.Builder

	// Try common article selectors
	selectors := []string{
		"article p",
		".article-body p",
		".story-body p",
		"[itemprop='articleBody'] p",
		"main p",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if len(text) > 50 { // Filter out short paragraphs
				content.WriteString(text)
				content.WriteString(" ")
			}
		})

		if content.Len() > 500 {
			break // Found good content
		}
	}

	fullText := content.String()

	// Limit to 2000 chars (FinBERT limit is 512 tokens ~2048 chars)
	if len(fullText) > 2000 {
		fullText = fullText[:2000]
	}

	if len(fullText) < 100 {
		return "", fmt.Errorf("insufficient content extracted")
	}

	return fullText, nil
}
