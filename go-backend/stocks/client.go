package stocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const StockServiceURL = "http://localhost:8001"

type PriceData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume int64   `json:"volume"`
}

type StockClient struct {
	httpClient *http.Client
}

func NewStockClient() *StockClient {
	return &StockClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *StockClient) GetHistoricalData(ticker string, startDate, endDate time.Time) ([]PriceData, error) {
	payload := map[string]string{
		"ticker":     ticker,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}

	jsonData, _ := json.Marshal(payload)

	resp, err := c.httpClient.Post(
		StockServiceURL+"/stock_data",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("stock service returned %d", resp.StatusCode)
	}

	var prices []PriceData
	if err := json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, err
	}

	return prices, nil
}

func (c *StockClient) GetPriceOnDate(ticker string, date time.Time) (*PriceData, error) {
	payload := map[string]string{
		"ticker": ticker,
		"date":   date.Format("2006-01-02"),
	}

	jsonData, _ := json.Marshal(payload)

	resp, err := c.httpClient.Post(
		StockServiceURL+"/stock_price",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("no price data found for %s on %s", ticker, date.Format("2006-01-02"))
	}

	var price PriceData
	if err := json.NewDecoder(resp.Body).Decode(&price); err != nil {
		return nil, err
	}

	return &price, nil
}
