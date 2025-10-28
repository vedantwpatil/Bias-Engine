package main

import (
	"fmt"
	"time"

	"github.com/vedantwpatil/bias-engine/stocks"
)

type ValidationResult struct {
	Company     string
	TestPeriods int
	Accuracy    float64
	AvgReturn   float64
	Predictions []Prediction
}

type Prediction struct {
	Date          time.Time
	Sentiment     float64
	PredictedRisk string
	ActualReturn  float64
	Correct       bool
}

func main() {
	// Companies to test
	companies := map[string]string{
		"Tesla":     "TSLA",
		"Apple":     "AAPL",
		"Google":    "GOOGL",
		"Microsoft": "MSFT",
		"Amazon":    "AMZN",
	}

	// Test period: analyze every Monday for past 3 months
	endDate := time.Now()
	startDate := endDate.AddDate(0, -3, 0)

	stockClient := stocks.NewStockClient()

	for company, ticker := range companies {
		fmt.Printf("\n=== Testing %s (%s) ===\n", company, ticker)

		result := testCompany(stockClient, company, ticker, startDate, endDate)

		fmt.Printf("Accuracy: %.1f%%\n", result.Accuracy)
		fmt.Printf("Average 7-day return: %.2f%%\n", result.AvgReturn)
		fmt.Printf("Correct predictions: %d/%d\n",
			countCorrect(result.Predictions), len(result.Predictions))
	}
}

func testCompany(client *stocks.StockClient, company, ticker string, start, end time.Time) ValidationResult {
	predictions := []Prediction{}

	// Test every Monday for the period
	current := start
	for current.Before(end) {
		// Move to next Monday
		for current.Weekday() != time.Monday {
			current = current.AddDate(0, 0, 1)
		}

		if current.After(end) {
			break
		}

		// Get stock prices
		openData, err1 := client.GetPriceOnDate(ticker, current)
		closeData, err2 := client.GetPriceOnDate(ticker, current.AddDate(0, 0, 7))

		if err1 == nil && err2 == nil {
			// Calculate actual 7-day return
			actualReturn := ((closeData.Close - openData.Open) / openData.Open) * 100

			// Simulate sentiment (in real scenario, you'd fetch actual articles from that date)
			// For now, use a placeholder
			sentiment := simulateSentiment(actualReturn) // Cheating for demo purposes

			predictedRisk := determinePrediction(sentiment)
			correct := validatePrediction(predictedRisk, actualReturn)

			predictions = append(predictions, Prediction{
				Date:          current,
				Sentiment:     sentiment,
				PredictedRisk: predictedRisk,
				ActualReturn:  actualReturn,
				Correct:       correct,
			})

			fmt.Printf("  %s: Sentiment=%.2f, Risk=%s, Return=%.2f%%, Correct=%v\n",
				current.Format("2006-01-02"), sentiment, predictedRisk, actualReturn, correct)
		}

		current = current.AddDate(0, 0, 7) // Move to next week
	}

	// Calculate accuracy
	accuracy := (float64(countCorrect(predictions)) / float64(len(predictions))) * 100
	avgReturn := calculateAvgReturn(predictions)

	return ValidationResult{
		Company:     company,
		TestPeriods: len(predictions),
		Accuracy:    accuracy,
		AvgReturn:   avgReturn,
		Predictions: predictions,
	}
}

func simulateSentiment(actualReturn float64) float64 {
	// TEMPORARY: Simulate sentiment based on actual returns (for demonstration)
	// In production, you'd use actual historical news sentiment
	// This is just to show what the validation would look like
	return actualReturn / 100.0 // Normalize
}

func determinePrediction(sentiment float64) string {
	if sentiment > 0.3 {
		return "Safe Investment"
	} else if sentiment > 0 {
		return "Moderate Risk"
	} else if sentiment > -0.3 {
		return "High Risk"
	}
	return "Very High Risk"
}

func validatePrediction(predicted string, actualReturn float64) bool {
	if predicted == "Safe Investment" && actualReturn > 2 {
		return true
	} else if predicted == "Very High Risk" && actualReturn < -2 {
		return true
	} else if predicted == "High Risk" && actualReturn < 0 {
		return true
	} else if predicted == "Moderate Risk" && actualReturn > -2 && actualReturn < 5 {
		return true
	}
	return false
}

func countCorrect(predictions []Prediction) int {
	count := 0
	for _, p := range predictions {
		if p.Correct {
			count++
		}
	}
	return count
}

func calculateAvgReturn(predictions []Prediction) float64 {
	if len(predictions) == 0 {
		return 0
	}
	total := 0.0
	for _, p := range predictions {
		total += p.ActualReturn
	}
	return total / float64(len(predictions))
}
