package storage

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/vedantwpatil/bias-engine/models"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(filepath string) (*Database, error) {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, err
	}

	// Create table
	createTable := `
    CREATE TABLE IF NOT EXISTS analyses (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        company TEXT NOT NULL,
        date DATETIME NOT NULL,
        sentiment_avg REAL,
        risk_score TEXT,
        article_count INTEGER,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_company_date ON analyses(company, date);
    `

	_, err = db.Exec(createTable)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) SaveAnalysisWithDate(company string, sentimentAvg float64, riskScore string, articleCount int, analysisDate time.Time) error {
	query := `INSERT INTO analyses (company, date, sentiment_avg, risk_score, article_count) 
              VALUES (?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query, company, analysisDate, sentimentAvg, riskScore, articleCount)
	return err
}

func (d *Database) GetHistoricalAnalyses(company string, startDate, endDate time.Time) ([]models.BacktestEntry, error) {
	query := `SELECT company, date, sentiment_avg, risk_score, article_count 
              FROM analyses 
              WHERE company = ? AND date BETWEEN ? AND ?
              ORDER BY date ASC`

	rows, err := d.db.Query(query, company, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.BacktestEntry
	for rows.Next() {
		var entry models.BacktestEntry
		err := rows.Scan(&entry.Company, &entry.Date, &entry.SentimentAvg, &entry.RiskScore, &entry.ArticleCount)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
