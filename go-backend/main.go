package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/vedantwpatil/bias-engine/models"
	"github.com/vedantwpatil/bias-engine/scraper"
)

var funcMap = template.FuncMap{
	"mul": func(a, b float64) float64 {
		return a * b
	},
	"add": func(a, b int) int {
		return a + b
	},
}

var sourceCredibility = map[string]float64{
	"Reuters":             1.0,
	"Bloomberg":           1.0,
	"Wall Street Journal": 1.0,
	"Financial Times":     0.95,
	"CNBC":                0.9,
	"MarketWatch":         0.85,
	"Seeking Alpha":       0.75,
	"Motley Fool":         0.7,
	"Yahoo Finance":       0.75,
	"Investor's Business": 0.8,
	"Barron's":            0.9,
	"Forbes":              0.85,
	"Investing.com":       0.7,
}

var (
	templates     = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	nlpServiceURL = getEnv("NLP_SERVICE_URL", "http://localhost:8000")
)

type ArticleWeight struct {
	Recency     float64
	Credibility float64
}

type CompanyAnalysis struct {
	Company       string
	Articles      []models.Article
	AvgSentiment  float64
	RiskScore     string
	TotalArticles int
	LastUpdated   time.Time
}

func calculateImpactScore(article models.Article) float64 {
	credibility, exists := sourceCredibility[article.Source]
	if !exists {
		credibility = 0.5 // Unknown sources get medium weight
	}

	// Recency
	hoursOld := time.Since(article.Timestamp).Hours()
	recencyWeight := 1.0 / (1.0 + hoursOld/24.0) // Decays over days

	// Sentiment Strength (how far from neutral)
	sentimentStrength := article.Sentiment.Positive - article.Sentiment.Negative
	sentimentMagnitude := math.Abs(sentimentStrength)

	// Confidence (how certain the model is)
	confidence := 1.0
	if article.Sentiment.Positive > article.Sentiment.Negative && article.Sentiment.Positive > article.Sentiment.Neutral {
		confidence = article.Sentiment.Positive
	} else if article.Sentiment.Negative > article.Sentiment.Positive && article.Sentiment.Negative > article.Sentiment.Neutral {
		confidence = article.Sentiment.Negative
	} else {
		confidence = article.Sentiment.Neutral
	}

	// Combined impact score
	// Weight distribution: 30% credibility, 20% recency, 30% magnitude, 20% confidence
	impactScore := (credibility * 0.3) +
		(recencyWeight * 0.2) +
		(sentimentMagnitude * 0.3) +
		(confidence * 0.2)

	return impactScore
}

func sortArticlesByImpact(articles []models.Article) []models.Article {
	// Calculate impact scores
	for i := range articles {
		articles[i].ImpactScore = calculateImpactScore(articles[i])
	}

	// Sort by impact score (highest first)
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].ImpactScore > articles[j].ImpactScore
	})

	return articles
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/analyze", analyzeHandler)
	http.HandleFunc("/api/company", apiCompanyHandler)

	log.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func calculateWeightedSentiment(articles []models.Article) float64 {
	if len(articles) == 0 {
		return 0
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for _, article := range articles {
		// Use impact score as weight
		weight := article.ImpactScore

		// Sentiment score
		score := article.Sentiment.Positive - article.Sentiment.Negative

		weightedSum += score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
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

	// Scrape articles
	articles, err := scraper.ScrapeCompanyNews(company, 100)
	if err != nil {
		log.Printf("Scraping error: %v", err)
		http.Error(w, "Failed to scrape articles", http.StatusInternalServerError)
		return
	}

	log.Printf("Scraped %d articles in %v", len(articles), time.Since(startTime))

	// Analyze articles concurrently
	articles = analyzeSentimentConcurrent(articles, 10)

	// Sort by impact (most influential first)
	articles = sortArticlesByImpact(articles)

	log.Printf("Total processing time: %v", time.Since(startTime))

	// Calculate risk score (now using weighted sentiment)
	analysis := CompanyAnalysis{
		Company:       company,
		Articles:      articles, // Now sorted by impact
		AvgSentiment:  calculateWeightedSentiment(articles),
		RiskScore:     determineRiskWeighted(articles),
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
				// Try to fetch full article first
				fullText, err := scraper.FetchFullArticle(articles[idx].URL)
				if err != nil {
					log.Printf("Failed to fetch full article %d: %v, using preview", idx, err)
					fullText = articles[idx].Body
				} else {
					log.Printf("Fetched full article %d (%d chars)", idx, len(fullText))
					articles[idx].Body = fullText
				}

				sentiment, err := analyzeSentiment(fullText)
				if err != nil {
					log.Printf("Analysis error for article %d: %v", idx, err)
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

	for i := range articles {
		articlesChan <- i
	}
	close(articlesChan)

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

func determineRiskWeighted(articles []models.Article) string {
	avgSentiment := calculateWeightedSentiment(articles)

	// Calculate consensus using impact-weighted variance
	consensus := calculateConsensus(articles)

	// Low consensus = high uncertainty = risky
	biasMultiplier := consensus

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

func calculateConsensus(articles []models.Article) float64 {
	if len(articles) == 0 {
		return 0
	}

	mean := calculateWeightedSentiment(articles)
	variance := 0.0
	totalWeight := 0.0

	for _, article := range articles {
		score := article.Sentiment.Positive - article.Sentiment.Negative
		weight := article.ImpactScore
		diff := score - mean
		variance += weight * diff * diff
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0
	}

	stdDev := math.Sqrt(variance / totalWeight)

	// Normalize: high stdDev = low consensus
	consensus := 1.0 / (1.0 + stdDev)

	return consensus
}
