package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/vedantwpatil/bias-engine/models"
	"github.com/vedantwpatil/bias-engine/scraper"
)

// Define custom template functions
var funcMap = template.FuncMap{
	"mul": func(a, b float64) float64 {
		return a * b
	},
}

// Parse templates with custom functions
var templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

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

	// Scrape articles
	articles, err := scraper.ScrapeCompanyNews(company, 10)
	if err != nil {
		log.Printf("Scraping error: %v", err)
		http.Error(w, "Failed to scrape articles", http.StatusInternalServerError)
		return
	}

	// Analyze each article
	for i := range articles {
		sentiment, err := analyzeSentiment(articles[i].Body)
		if err != nil {
			log.Printf("Analysis error: %v", err)
			continue
		}
		articles[i].Sentiment = sentiment
	}

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

func apiCompanyHandler(w http.ResponseWriter, r *http.Request) {
	company := r.URL.Query().Get("name")
	if company == "" {
		http.Error(w, "Company parameter required", http.StatusBadRequest)
		return
	}

	articles, err := scraper.ScrapeCompanyNews(company, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

func analyzeSentiment(text string) (models.Sentiment, error) {
	payload := map[string]string{"text": text}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(
		"http://localhost:5000/analyze",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return models.Sentiment{}, err
	}
	defer resp.Body.Close()

	var sentiment models.Sentiment
	if err := json.NewDecoder(resp.Body).Decode(&sentiment); err != nil {
		return models.Sentiment{}, err
	}

	return sentiment, nil
}

func calculateAvgSentiment(articles []models.Article) float64 {
	if len(articles) == 0 {
		return 0
	}

	total := 0.0
	for _, article := range articles {
		// Calculate weighted sentiment (positive - negative)
		score := article.Sentiment.Positive - article.Sentiment.Negative
		total += score
	}

	return total / float64(len(articles))
}

func determineRisk(articles []models.Article) string {
	avgSentiment := calculateAvgSentiment(articles)

	// Calculate bias (uncertainty) from neutral scores
	avgNeutral := 0.0
	for _, article := range articles {
		avgNeutral += article.Sentiment.Neutral
	}
	avgNeutral /= float64(len(articles))

	// High neutral = high bias/uncertainty = risky
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
