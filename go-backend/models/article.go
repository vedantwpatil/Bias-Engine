package models

import "time"

type Article struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Source    string    `json:"source"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
	Sentiment Sentiment `json:"sentiment"`
}

type Sentiment struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}
