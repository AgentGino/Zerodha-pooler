package storage

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// SQLiteStore provides a storage interface for SQLite.
type SQLiteStore struct {
	db     *sql.DB
	logger *log.Logger
}

// NewSQLiteStore creates a new SQLite store.
func NewSQLiteStore(path string, logger *log.Logger) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite connection failed: %v", err)
	}
	return &SQLiteStore{db: db, logger: logger}, nil
}

// Init initializes the database schema.
func (s *SQLiteStore) Init() error {
	createTable := `
	CREATE TABLE IF NOT EXISTS ohlcv (
		instrument TEXT,
		open REAL,
		high REAL,
		low REAL,
		close REAL,
		timestamp TEXT,
		volume INTEGER
	);`
	if _, err := s.db.Exec(createTable); err != nil {
		return fmt.Errorf("failed to create SQLite table: %v", err)
	}
	s.logger.Println("✅ SQLite table 'ohlcv' is ready.")
	return nil
}

// StoreCandles inserts a slice of candles into the database.
func (s *SQLiteStore) StoreCandles(instrumentSymbol string, candles []kiteconnect.HistoricalData) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("DB transaction error: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO ohlcv VALUES (?,?,?,?,?,?,?)")
	if err != nil {
		return 0, fmt.Errorf("DB prepare error: %v", err)
	}
	defer stmt.Close()

	var inserted int
	for _, c := range candles {
		_, err := stmt.Exec(
			instrumentSymbol,
			c.Open,
			c.High,
			c.Low,
			c.Close,
			c.Date.Time.Format("2006-01-02 15:04:05"),
			c.Volume,
		)
		if err != nil {
			s.logger.Printf("      \\_ Insert error: %v, for candle %+v", err, c)
		} else {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit error: %v", err)
	}
	return inserted, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
