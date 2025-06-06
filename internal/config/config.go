package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all the configuration for the application.
type Config struct {
	APIKey       string   `yaml:"api_key"`
	APISecret    string   `yaml:"api_secret"`
	RequestToken string   `yaml:"request_token"`
	Instruments  []string `yaml:"instruments"`
	FromDate     string   `yaml:"from_date"`
	ToDate       string   `yaml:"to_date"`
	Interval     string   `yaml:"interval"`
	StorageType  string   `yaml:"storage_type"` // "duckdb", "sqlite", "json", "csv"
	StoragePath  string   `yaml:"storage_path"` // Path to database file or directory for files
	LogFile      string   `yaml:"log_file"`

	// Deprecated: Use StoragePath instead
	DuckDBPath string `yaml:"duckdb_path,omitempty"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("%s (%s): %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult holds all validation errors
type ValidationResult struct {
	Errors []ValidationError
}

func (r *ValidationResult) AddError(field, value, message string) {
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	})
}

func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r *ValidationResult) Error() string {
	if !r.HasErrors() {
		return ""
	}

	var messages []string
	for _, err := range r.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// Load reads the configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

// Save writes the configuration to a YAML file.
func Save(path string, conf *Config) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ValidateBasic performs basic validation of required fields and formats
func (c *Config) ValidateBasic() *ValidationResult {
	result := &ValidationResult{}

	// Required fields
	if c.APIKey == "" {
		result.AddError("api_key", "", "is required")
	}
	if c.APISecret == "" {
		result.AddError("api_secret", "", "is required")
	}
	if len(c.Instruments) == 0 {
		result.AddError("instruments", "", "at least one instrument must be specified")
	}
	if c.FromDate == "" {
		result.AddError("from_date", "", "is required")
	}
	if c.ToDate == "" {
		result.AddError("to_date", "", "is required")
	}
	if c.Interval == "" {
		result.AddError("interval", "", "is required")
	}

	// Date format validation
	if c.FromDate != "" {
		if _, err := time.Parse("2006-01-02", c.FromDate); err != nil {
			result.AddError("from_date", c.FromDate, "must be in YYYY-MM-DD format")
		}
	}
	if c.ToDate != "" {
		if _, err := time.Parse("2006-01-02", c.ToDate); err != nil {
			result.AddError("to_date", c.ToDate, "must be in YYYY-MM-DD format")
		}
	}

	// Date range validation
	if c.FromDate != "" && c.ToDate != "" {
		from, err1 := time.Parse("2006-01-02", c.FromDate)
		to, err2 := time.Parse("2006-01-02", c.ToDate)
		if err1 == nil && err2 == nil {
			if !from.Before(to) {
				result.AddError("date_range", fmt.Sprintf("%s to %s", c.FromDate, c.ToDate), "from_date must be before to_date")
			}
		}
	}

	// Interval validation
	validIntervals := []string{"minute", "3minute", "5minute", "10minute", "15minute", "30minute", "60minute", "day"}
	if c.Interval != "" {
		intervalValid := false
		for _, valid := range validIntervals {
			if c.Interval == valid {
				intervalValid = true
				break
			}
		}
		if !intervalValid {
			result.AddError("interval", c.Interval, fmt.Sprintf("must be one of: %s", strings.Join(validIntervals, ", ")))
		}
	}

	// Storage type validation
	if c.StorageType != "" {
		validStorageTypes := []string{"duckdb", "sqlite", "json", "csv"}
		storageTypeValid := false
		for _, valid := range validStorageTypes {
			if c.StorageType == valid {
				storageTypeValid = true
				break
			}
		}
		if !storageTypeValid {
			result.AddError("storage_type", c.StorageType, fmt.Sprintf("must be one of: %s", strings.Join(validStorageTypes, ", ")))
		}
	}

	// Instrument validation (basic format check)
	for _, instrument := range c.Instruments {
		if strings.TrimSpace(instrument) == "" {
			result.AddError("instruments", instrument, "empty instrument symbol found")
		}
		if len(instrument) > 20 {
			result.AddError("instruments", instrument, "instrument symbol too long (max 20 characters)")
		}
	}

	return result
}

// ValidateStorage performs storage-specific validation
func (c *Config) ValidateStorage() *ValidationResult {
	result := &ValidationResult{}

	// Determine storage type and path
	storageType := c.StorageType
	storagePath := c.StoragePath

	// Handle backward compatibility
	if storagePath == "" && c.DuckDBPath != "" {
		storagePath = c.DuckDBPath
		if storageType == "" {
			storageType = "duckdb"
		}
	}

	// Set defaults if not specified
	if storageType == "" {
		storageType = "duckdb"
	}

	// Validate storage path
	if storagePath != "" {
		switch storageType {
		case "duckdb", "sqlite":
			// For database files, check if parent directory exists or can be created
			dir := filepath.Dir(storagePath)
			if dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					result.AddError("storage_path", storagePath, fmt.Sprintf("cannot create directory: %v", err))
				}
			}
			// Check if we can write to the file
			if _, err := os.Stat(storagePath); err == nil {
				// File exists, check if writable
				if f, err := os.OpenFile(storagePath, os.O_WRONLY, 0); err != nil {
					result.AddError("storage_path", storagePath, "file exists but is not writable")
				} else {
					f.Close()
				}
			}

		case "json", "csv":
			// For file-based storage, ensure it's a directory
			if err := os.MkdirAll(storagePath, 0755); err != nil {
				result.AddError("storage_path", storagePath, fmt.Sprintf("cannot create directory: %v", err))
			}
		}
	}

	// Validate log file path
	if c.LogFile != "" {
		logDir := filepath.Dir(c.LogFile)
		if logDir != "." {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				result.AddError("log_file", c.LogFile, fmt.Sprintf("cannot create log directory: %v", err))
			}
		}
	}

	return result
}

// ValidateComplete performs comprehensive validation including basic and storage validation
func (c *Config) ValidateComplete() *ValidationResult {
	result := &ValidationResult{}

	// Combine basic validation
	basicResult := c.ValidateBasic()
	result.Errors = append(result.Errors, basicResult.Errors...)

	// Add storage validation if basic validation passed
	if !basicResult.HasErrors() {
		storageResult := c.ValidateStorage()
		result.Errors = append(result.Errors, storageResult.Errors...)
	}

	return result
}
