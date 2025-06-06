package storage

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// CSVStore provides a storage interface for CSV files (one file per instrument).
type CSVStore struct {
	basePath string
	logger   *log.Logger
}

// NewCSVStore creates a new CSV store.
func NewCSVStore(basePath string, logger *log.Logger) (*CSVStore, error) {
	return &CSVStore{basePath: basePath, logger: logger}, nil
}

// Init initializes the storage directory.
func (s *CSVStore) Init() error {
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create CSV storage directory: %v", err)
	}
	s.logger.Printf("✅ CSV storage directory ready: %s", s.basePath)
	return nil
}

// StoreCandles stores candles to a CSV file for the specific instrument.
func (s *CSVStore) StoreCandles(instrumentSymbol string, candles []kiteconnect.HistoricalData) (int, error) {
	fileName := fmt.Sprintf("%s.csv", instrumentSymbol)
	filePath := filepath.Join(s.basePath, fileName)

	// Check if file exists to determine if we need headers
	fileExists := false
	if _, err := os.Stat(filePath); err == nil {
		fileExists = true
	}

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if this is a new file
	if !fileExists {
		header := []string{"instrument", "timestamp", "open", "high", "low", "close", "volume"}
		if err := writer.Write(header); err != nil {
			return 0, fmt.Errorf("failed to write CSV header: %v", err)
		}
	}

	// Write candle data
	var inserted int
	for _, c := range candles {
		record := []string{
			instrumentSymbol,
			c.Date.Time.Format("2006-01-02 15:04:05"),
			strconv.FormatFloat(c.Open, 'f', -1, 64),
			strconv.FormatFloat(c.High, 'f', -1, 64),
			strconv.FormatFloat(c.Low, 'f', -1, 64),
			strconv.FormatFloat(c.Close, 'f', -1, 64),
			strconv.FormatInt(int64(c.Volume), 10),
		}

		if err := writer.Write(record); err != nil {
			s.logger.Printf("      \\_ CSV write error: %v, for candle %+v", err, c)
		} else {
			inserted++
		}
	}

	s.logger.Printf("📄 Stored %d candles to %s", len(candles), fileName)
	return inserted, nil
}

// Close cleanup resources (no-op for CSV).
func (s *CSVStore) Close() error {
	return nil
}
