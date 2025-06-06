package storage

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/marcboeker/go-duckdb"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// DuckDBStore provides a storage interface for DuckDB.
type DuckDBStore struct {
	db     *sql.DB
	logger *log.Logger
}

// NewDuckDBStore creates a new DuckDB store.
func NewDuckDBStore(path string, logger *log.Logger) (*DuckDBStore, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("duckdb connection failed: %v", err)
	}
	return &DuckDBStore{db: db, logger: logger}, nil
}

// Init initializes the database schema.
func (s *DuckDBStore) Init() error {
	createTable := `
	CREATE TABLE IF NOT EXISTS ohlcv (
		instrument VARCHAR,
		open DOUBLE,
		high DOUBLE,
		low DOUBLE,
		close DOUBLE,
		timestamp TIMESTAMP,
		volume BIGINT
	);`
	if _, err := s.db.Exec(createTable); err != nil {
		return fmt.Errorf("failed to create DuckDB table: %v", err)
	}
	s.logger.Println("✅ DuckDB table 'ohlcv' is ready.")
	return nil
}

// StoreCandles inserts a slice of candles into the database.
func (s *DuckDBStore) StoreCandles(instrumentSymbol string, candles []kiteconnect.HistoricalData) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("DB transaction error: %v", err)
	}
	defer tx.Rollback() // Rollback on error

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
			c.Date.Time,
			c.Volume,
		)
		if err != nil {
			// Log individual insert error but continue trying to insert others
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
func (s *DuckDBStore) Close() error {
	return s.db.Close()
}
