package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/vedantwpatil/bias-engine/models"
	"github.com/vedantwpatil/bias-engine/scraper"
)

var funcMap = template.FuncMap{
	"mul": func(a, b float64) float64 {
		return a * b
	},
}

var (
	templates     = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	nlpServiceURL = getEnv("NLP_SERVICE_URL", "http://localhost:8000")
)

type CompanyAnalysis struct {
	Company       string
	Articles      []models.Article
	AvgSentiment  float64
	RiskScore     string
	TotalArticles int
	LastUpdated   time.Time
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/analyze", analyzeHandler)
	http.HandleFunc("/api/company", apiCompanyHandler)

	log.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "index.html", nil)
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	company := r.FormValue("company")
	if company == "" {
		http.Error(w, "Company name required", http.StatusBadRequest)
		return
	}

	startTime := time.Now()

	// Scrape articles - increase to 100
	articles, err := scraper.ScrapeCompanyNews(company, 100)
	if err != nil {
		log.Printf("Scraping error: %v", err)
		http.Error(w, "Failed to scrape articles", http.StatusInternalServerError)
		return
	}

	log.Printf("Scraped %d articles in %v", len(articles), time.Since(startTime))

	// Analyze articles concurrently
	articles = analyzeSentimentConcurrent(articles, 10) // 10 concurrent workers

	log.Printf("Total processing time: %v", time.Since(startTime))

	// Calculate risk score
	analysis := CompanyAnalysis{
		Company:       company,
		Articles:      articles,
		AvgSentiment:  calculateAvgSentiment(articles),
		RiskScore:     determineRisk(articles),
		TotalArticles: len(articles),
		LastUpdated:   time.Now(),
	}

	templates.ExecuteTemplate(w, "company.html", analysis)
}

// Concurrent sentiment analysis with worker pool
func analyzeSentimentConcurrent(articles []models.Article, numWorkers int) []models.Article {
	var wg sync.WaitGroup
	articlesChan := make(chan int, len(articles))

	// Launch workers
	for range numWorkers {
		wg.Go(func() {
			for idx := range articlesChan {
				sentiment, err := analyzeSentiment(articles[idx].Body)
				if err != nil {
					log.Printf("Analysis error for article %d: %v", idx, err)
					// Set neutral sentiment on error
					articles[idx].Sentiment = models.Sentiment{
						Positive: 0.33,
						Neutral:  0.34,
						Negative: 0.33,
					}
					continue
				}
				articles[idx].Sentiment = sentiment
			}
		})
	}

	// Send work to workers
	for i := range articles {
		articlesChan <- i
	}
	close(articlesChan)

	// Wait for all workers to finish
	wg.Wait()

	return articles
}

func analyzeSentiment(text string) (models.Sentiment, error) {
	if text == "" {
		return models.Sentiment{}, fmt.Errorf("empty text")
	}

	payload := map[string]string{"text": text}
	jsonData, _ := json.Marshal(payload)

	client := &http.Client{
		Timeout: 15 * time.Second, // Increased timeout
	}

	resp, err := client.Post(
		nlpServiceURL+"/analyze",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return models.Sentiment{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return models.Sentiment{}, fmt.Errorf("status %d", resp.StatusCode)
	}

	var sentiment models.Sentiment
	if err := json.NewDecoder(resp.Body).Decode(&sentiment); err != nil {
		return models.Sentiment{}, err
	}

	return sentiment, nil
}

func apiCompanyHandler(w http.ResponseWriter, r *http.Request) {
	company := r.URL.Query().Get("name")
	if company == "" {
		http.Error(w, "Company parameter required", http.StatusBadRequest)
		return
	}

	articles, err := scraper.ScrapeCompanyNews(company, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

func calculateAvgSentiment(articles []models.Article) float64 {
	if len(articles) == 0 {
		return 0
	}

	total := 0.0
	for _, article := range articles {
		score := article.Sentiment.Positive - article.Sentiment.Negative
		total += score
	}

	return total / float64(len(articles))
}

func determineRisk(articles []models.Article) string {
	avgSentiment := calculateAvgSentiment(articles)

	avgNeutral := 0.0
	for _, article := range articles {
		avgNeutral += article.Sentiment.Neutral
	}
	avgNeutral /= float64(len(articles))

	biasMultiplier := 1.0 - avgNeutral
	adjustedScore := avgSentiment * biasMultiplier

	if adjustedScore > 0.3 {
		return "Safe Investment"
	} else if adjustedScore > 0 {
		return "Moderate Risk"
	} else if adjustedScore > -0.3 {
		return "High Risk"
	}
	return "Very High Risk"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
