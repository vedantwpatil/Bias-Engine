package backtest

import (
	"log"
	"time"

	"github.com/vedantwpatil/bias-engine/models"
	"github.com/vedantwpatil/bias-engine/scraper"
)

// SimulateHistoricalAnalysis runs your analyzer on a past date and stores the result
func (b *BacktestEngine) SimulateHistoricalAnalysis(company string, date time.Time) error {
	log.Printf("Simulating analysis for %s on %s", company, date.Format("2006-01-02"))

	// Scrape articles (these will be current articles, but we'll pretend they're from the past date)
	articles, err := scraper.ScrapeCompanyNews(company, 100)
	if err != nil {
		return err
	}

	// Calculate sentiment (using your existing logic)
	totalSentiment := 0.0
	for _, article := range articles {
		score := article.Sentiment.Positive - article.Sentiment.Negative
		totalSentiment += score
	}
	avgSentiment := totalSentiment / float64(len(articles))

	// Determine risk score (simplified version)
	var riskScore string
	if avgSentiment > 0.3 {
		riskScore = "Safe Investment"
	} else if avgSentiment > 0 {
		riskScore = "Moderate Risk"
	} else if avgSentiment > -0.3 {
		riskScore = "High Risk"
	} else {
		riskScore = "Very High Risk"
	}

	// Save with the PAST date
	entry := models.BacktestEntry{
		Company:      company,
		Date:         date,
		SentimentAvg: avgSentiment,
		RiskScore:    riskScore,
		ArticleCount: len(articles),
	}

	// Save to database with backdated timestamp
	return b.db.SaveAnalysisWithDate(company, avgSentiment, riskScore, len(articles), date)
}
