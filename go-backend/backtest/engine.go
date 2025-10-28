package backtest

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/vedantwpatil/bias-engine/models"
	"github.com/vedantwpatil/bias-engine/stocks"
	"github.com/vedantwpatil/bias-engine/storage"
)

type BacktestEngine struct {
	db          *storage.Database
	stockClient *stocks.StockClient
}

func NewBacktestEngine(db *storage.Database) *BacktestEngine {
	return &BacktestEngine{
		db:          db,
		stockClient: stocks.NewStockClient(),
	}
}

func (b *BacktestEngine) RunBacktest(company, ticker string, startDate, endDate time.Time) (models.BacktestResult, error) {
	log.Printf("Running backtest for %s (%s) from %s to %s",
		company, ticker, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// Get historical analyses from database
	entries, err := b.db.GetHistoricalAnalyses(company, startDate, endDate)
	if err != nil {
		return models.BacktestResult{}, err
	}

	if len(entries) == 0 {
		return models.BacktestResult{}, fmt.Errorf("no historical data found for %s", company)
	}

	log.Printf("Found %d historical analyses", len(entries))

	// Fetch stock prices for all dates at once (more efficient)
	priceStart := startDate.AddDate(0, 0, -5) // Get a bit earlier for open prices
	priceEnd := endDate.AddDate(0, 0, 35)     // Get 35 days ahead for future returns

	allPrices, err := b.stockClient.GetHistoricalData(ticker, priceStart, priceEnd)
	if err != nil {
		return models.BacktestResult{}, fmt.Errorf("failed to fetch stock data: %v", err)
	}

	log.Printf("Fetched %d days of price data", len(allPrices))

	// Create price lookup map
	priceMap := make(map[string]stocks.PriceData)
	for _, price := range allPrices {
		priceMap[price.Date] = price
	}

	// Calculate returns for each analysis
	for i := range entries {
		analysisDate := entries[i].Date.Format("2006-01-02")

		// Get opening price on analysis date
		if openData, exists := priceMap[analysisDate]; exists {
			entries[i].OpenPrice = openData.Open

			// Calculate future returns
			entries[i].ClosePrice1D = b.getClosePriceNDaysLater(priceMap, entries[i].Date, 1)
			entries[i].ClosePrice7D = b.getClosePriceNDaysLater(priceMap, entries[i].Date, 7)
			entries[i].ClosePrice30D = b.getClosePriceNDaysLater(priceMap, entries[i].Date, 30)

			if entries[i].OpenPrice > 0 {
				entries[i].Return1D = ((entries[i].ClosePrice1D - entries[i].OpenPrice) / entries[i].OpenPrice) * 100
				entries[i].Return7D = ((entries[i].ClosePrice7D - entries[i].OpenPrice) / entries[i].OpenPrice) * 100
				entries[i].Return30D = ((entries[i].ClosePrice30D - entries[i].OpenPrice) / entries[i].OpenPrice) * 100
			}

			log.Printf("Analysis %d: Date=%s, Sentiment=%.2f, Risk=%s, 7D Return=%.2f%%",
				i+1, analysisDate, entries[i].SentimentAvg, entries[i].RiskScore, entries[i].Return7D)
		}
	}

	// Calculate accuracy metrics
	result := models.BacktestResult{
		TotalAnalyses: len(entries),
		Entries:       entries,
		Accuracy1D:    b.calculateAccuracy(entries, "1D"),
		Accuracy7D:    b.calculateAccuracy(entries, "7D"),
		Accuracy30D:   b.calculateAccuracy(entries, "30D"),
	}

	log.Printf("Backtest complete: 1D=%.1f%%, 7D=%.1f%%, 30D=%.1f%%",
		result.Accuracy1D, result.Accuracy7D, result.Accuracy30D)

	return result, nil
}

func (b *BacktestEngine) getClosePriceNDaysLater(priceMap map[string]stocks.PriceData, date time.Time, days int) float64 {
	// Try to find price exactly N days later
	targetDate := date.AddDate(0, 0, days)

	// Try up to 5 business days later (in case of weekends/holidays)
	for offset := 0; offset < 5; offset++ {
		checkDate := targetDate.AddDate(0, 0, offset).Format("2006-01-02")
		if price, exists := priceMap[checkDate]; exists {
			return price.Close
		}
	}

	return 0 // No data found
}

func (b *BacktestEngine) calculateAccuracy(entries []models.BacktestEntry, horizon string) float64 {
	correct := 0
	total := 0

	for _, entry := range entries {
		var actualReturn float64

		switch horizon {
		case "1D":
			actualReturn = entry.Return1D
		case "7D":
			actualReturn = entry.Return7D
		case "30D":
			actualReturn = entry.Return30D
		}

		if actualReturn == 0 {
			continue // Skip if no data
		}

		// Check prediction accuracy
		predicted := entry.RiskScore

		// Scoring logic
		if predicted == "Safe Investment" && actualReturn > 2 {
			correct++
		} else if predicted == "Very High Risk" && actualReturn < -2 {
			correct++
		} else if predicted == "High Risk" && actualReturn < 0 {
			correct++
		} else if predicted == "Moderate Risk" && math.Abs(actualReturn) < 5 {
			correct++
		}

		total++
	}

	if total == 0 {
		return 0
	}

	return (float64(correct) / float64(total)) * 100
}
