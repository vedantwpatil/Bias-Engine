package models

import "time"

type BacktestEntry struct {
	Company      string    `json:"company"`
	Date         time.Time `json:"date"`
	SentimentAvg float64   `json:"sentiment_avg"`
	RiskScore    string    `json:"risk_score"`
	ArticleCount int       `json:"article_count"`

	// Stock data (fetched later)
	OpenPrice     float64 `json:"open_price"`
	ClosePrice1D  float64 `json:"close_1d"`
	ClosePrice7D  float64 `json:"close_7d"`
	ClosePrice30D float64 `json:"close_30d"`
	Return1D      float64 `json:"return_1d"`
	Return7D      float64 `json:"return_7d"`
	Return30D     float64 `json:"return_30d"`
}

type BacktestResult struct {
	TotalAnalyses int             `json:"total_analyses"`
	Accuracy1D    float64         `json:"accuracy_1d"`
	Accuracy7D    float64         `json:"accuracy_7d"`
	Accuracy30D   float64         `json:"accuracy_30d"`
	Entries       []BacktestEntry `json:"entries"`
}
