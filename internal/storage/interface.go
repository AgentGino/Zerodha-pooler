package storage

import (
	"log"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// Store defines the interface for all storage implementations.
type Store interface {
	// Init initializes the storage (creates tables, directories, etc.)
	Init() error

	// StoreCandles stores historical data for an instrument
	StoreCandles(instrumentSymbol string, candles []kiteconnect.HistoricalData) (int, error)

	// Close cleanup resources
	Close() error
}

// StorageType represents the different storage types available.
type StorageType string

const (
	StorageTypeDuckDB StorageType = "duckdb"
	StorageTypeSQLite StorageType = "sqlite"
	StorageTypeJSON   StorageType = "json"
	StorageTypeCSV    StorageType = "csv"
)

// NewStore creates a new storage instance based on the specified type.
func NewStore(storageType StorageType, path string, logger *log.Logger) (Store, error) {
	switch storageType {
	case StorageTypeDuckDB:
		return NewDuckDBStore(path, logger)
	case StorageTypeSQLite:
		return NewSQLiteStore(path, logger)
	case StorageTypeJSON:
		return NewJSONStore(path, logger)
	case StorageTypeCSV:
		return NewCSVStore(path, logger)
	default:
		return NewDuckDBStore(path, logger) // Default fallback
	}
}
