package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// JSONStore provides a storage interface for JSON files (one file per instrument).
type JSONStore struct {
	basePath string
	logger   *log.Logger
}

// NewJSONStore creates a new JSON store.
func NewJSONStore(basePath string, logger *log.Logger) (*JSONStore, error) {
	return &JSONStore{basePath: basePath, logger: logger}, nil
}

// Init initializes the storage directory.
func (s *JSONStore) Init() error {
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create JSON storage directory: %v", err)
	}
	s.logger.Printf("✅ JSON storage directory ready: %s", s.basePath)
	return nil
}

// StoreCandles stores candles to a JSON file for the specific instrument.
func (s *JSONStore) StoreCandles(instrumentSymbol string, candles []kiteconnect.HistoricalData) (int, error) {
	fileName := fmt.Sprintf("%s.json", instrumentSymbol)
	filePath := filepath.Join(s.basePath, fileName)

	// Load existing data if file exists
	var existingData []kiteconnect.HistoricalData
	if data, err := os.ReadFile(filePath); err == nil {
		json.Unmarshal(data, &existingData)
	}

	// Append new candles
	allData := append(existingData, candles...)

	// Write back to file
	jsonData, err := json.MarshalIndent(allData, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal JSON data: %v", err)
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return 0, fmt.Errorf("failed to write JSON file: %v", err)
	}

	s.logger.Printf("📄 Stored %d candles to %s (total: %d)", len(candles), fileName, len(allData))
	return len(candles), nil
}

// Close cleanup resources (no-op for JSON).
func (s *JSONStore) Close() error {
	return nil
}
