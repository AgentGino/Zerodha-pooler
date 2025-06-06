package cli

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"zerodha-connect/internal/config"
	"zerodha-connect/internal/kite"
	"zerodha-connect/internal/storage"

	"github.com/spf13/cobra"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and API connectivity",
	Long: `Validate your configuration file and test API connectivity.

This command performs the following checks:
- Validates configuration file format and required fields
- Tests Zerodha API connectivity and authentication
- Verifies instrument symbols are valid
- Checks storage backend accessibility
- Validates date ranges and intervals

This is useful for troubleshooting issues before running a full fetch operation.

Examples:
  # Validate default config file
  zerodha-connect validate

  # Validate specific config file
  zerodha-connect validate --config my-config.yaml`,
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	fmt.Printf("🔍 Validating: %s\n\n", configFile)

	// Load configuration
	conf, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("❌ Config file error: %v", err)
	}

	// Perform field-by-field validation report
	showFieldValidationReport(conf)

	// Perform comprehensive validation
	validation := conf.ValidateComplete()
	if validation.HasErrors() {
		fmt.Println("\n❌ VALIDATION ERRORS:")
		for _, err := range validation.Errors {
			fmt.Printf("   • %s\n", err.Error())
		}
		return fmt.Errorf("\nPlease fix the above errors and try again")
	}

	// Test components silently
	fmt.Println("🔄 Testing components...")

	// Create a completely silent logger for validation
	tempLogger := log.New(io.Discard, "", 0)

	// Test storage
	if err := testStorage(conf, tempLogger); err != nil {
		fmt.Printf("   ❌ Storage: %v\n", err)
		return fmt.Errorf("storage test failed")
	}
	fmt.Println("   ✅ Storage: Ready")

	// Test API
	if err := testAPI(conf, tempLogger); err != nil {
		fmt.Printf("   ❌ API: %v\n", err)
		return fmt.Errorf("API test failed")
	}
	fmt.Println("   ✅ API: Connected")

	// Test instruments
	validCount, totalCount, err := testInstruments(conf, tempLogger)
	if err != nil {
		fmt.Printf("   ❌ Instruments: %v\n", err)
		return fmt.Errorf("instrument test failed")
	}
	fmt.Printf("   ✅ Instruments: %d/%d valid\n", validCount, totalCount)

	// Show execution estimate
	showExecutionEstimate(conf)

	fmt.Println("\n🎉 All validations passed!")
	fmt.Println("   Ready to fetch data")
	return nil
}

func showFieldValidationReport(conf *config.Config) {
	fmt.Println("📋 Configuration Check:")

	// Check if we have a valid date range first
	var dateRangeValid bool
	var dateRangeError string
	if isValidDate(conf.FromDate) && isValidDate(conf.ToDate) {
		from, _ := time.Parse("2006-01-02", conf.FromDate)
		to, _ := time.Parse("2006-01-02", conf.ToDate)
		dateRangeValid = from.Before(to)
		if !dateRangeValid {
			dateRangeError = "from_date must be before to_date"
		}
	}

	// Required fields - show date fields as invalid if range is invalid
	checkField("API Key", conf.APIKey != "", conf.APIKey != "")
	checkField("API Secret", conf.APISecret != "", conf.APISecret != "")
	checkField("Instruments", len(conf.Instruments) > 0, fmt.Sprintf("%d symbols", len(conf.Instruments)))

	// Date validation with range check
	fromDateValid := isValidDate(conf.FromDate) && (dateRangeValid || conf.ToDate == "")
	toDateValid := isValidDate(conf.ToDate) && (dateRangeValid || conf.FromDate == "")

	checkFieldWithNote("From Date", fromDateValid, conf.FromDate, dateRangeError)
	checkFieldWithNote("To Date", toDateValid, conf.ToDate, dateRangeError)
	checkField("Interval", isValidInterval(conf.Interval), conf.Interval)

	// Optional fields
	storageType := conf.StorageType
	if storageType == "" {
		storageType = "duckdb (default)"
	}
	checkField("Storage Type", isValidStorageType(conf.StorageType), storageType)

	// Date range check
	if isValidDate(conf.FromDate) && isValidDate(conf.ToDate) {
		from, _ := time.Parse("2006-01-02", conf.FromDate)
		to, _ := time.Parse("2006-01-02", conf.ToDate)
		days := int(to.Sub(from).Hours() / 24)
		checkField("Date Range", from.Before(to), fmt.Sprintf("%d days", days))
	}
}

func checkField(name string, isValid bool, value interface{}) {
	checkFieldWithNote(name, isValid, value, "")
}

func checkFieldWithNote(name string, isValid bool, value interface{}, note string) {
	status := "❌"
	if isValid {
		status = "✅"
	}

	// Format value display
	var displayValue string
	switch v := value.(type) {
	case string:
		if strings.Contains(name, "API") && v != "" && !strings.Contains(v, "(") {
			// Mask API credentials
			displayValue = "****" + v[len(v)-4:]
		} else {
			displayValue = v
		}
	case bool:
		if v {
			displayValue = "present"
		} else {
			displayValue = "missing"
		}
	default:
		displayValue = fmt.Sprintf("%v", v)
	}

	// Add note if there's a cross-field validation issue
	if !isValid && note != "" && displayValue != "missing" && displayValue != "" {
		displayValue = displayValue + " ⚠️"
	}

	fmt.Printf("   %s %-15s %s\n", status, name+":", displayValue)
}

func testStorage(conf *config.Config, logger *log.Logger) error {
	storageType := storage.StorageType(conf.StorageType)
	if storageType == "" {
		storageType = storage.StorageTypeDuckDB
	}
	storagePath := conf.StoragePath
	if storagePath == "" && conf.DuckDBPath != "" {
		storagePath = conf.DuckDBPath
	}
	if storagePath == "" {
		switch storageType {
		case storage.StorageTypeJSON:
			storagePath = "data/json"
		case storage.StorageTypeCSV:
			storagePath = "data/csv"
		case storage.StorageTypeSQLite:
			storagePath = "market_data.sqlite"
		default:
			storagePath = "market_data.duckdb"
		}
	}

	store, err := storage.NewStore(storageType, storagePath, logger)
	if err != nil {
		return fmt.Errorf("initialization failed")
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		return fmt.Errorf("setup failed")
	}
	return nil
}

func testAPI(conf *config.Config, logger *log.Logger) error {
	kiteClient := kite.NewClient(conf, logger)
	if err := kiteClient.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed")
	}
	return nil
}

func testInstruments(conf *config.Config, logger *log.Logger) (int, int, error) {
	kiteClient := kite.NewClient(conf, logger)
	if err := kiteClient.Authenticate(); err != nil {
		return 0, 0, err
	}

	instruments, err := kite.GetInstruments(kiteClient.GetKiteConnectClient(), logger)
	if err != nil {
		return 0, 0, fmt.Errorf("API fetch failed")
	}

	// Validate requested instruments
	instrumentTokenMap := make(map[string]int)
	for _, instr := range instruments {
		instrumentTokenMap[instr.Tradingsymbol] = int(instr.InstrumentToken)
	}

	validCount := 0
	for _, symbol := range conf.Instruments {
		if _, exists := instrumentTokenMap[symbol]; exists {
			validCount++
		}
	}

	if validCount == 0 {
		return 0, len(conf.Instruments), fmt.Errorf("no valid symbols found")
	}

	return validCount, len(conf.Instruments), nil
}

func showExecutionEstimate(conf *config.Config) {
	from, _ := time.Parse("2006-01-02", conf.FromDate)
	to, _ := time.Parse("2006-01-02", conf.ToDate)

	// Rough estimate based on valid instruments
	chunks := kite.GenerateDateChunks(from, to, conf.Interval)
	totalAPICalls := len(chunks) * len(conf.Instruments) // Approximate

	estimatedTimeSeconds := float64(totalAPICalls) / float64(kite.RateLimitRequestsPerSecond)
	estimatedMinutes := int(estimatedTimeSeconds / 60)

	fmt.Println("\n⏱️  Execution Estimate:")
	fmt.Printf("   📊 API Calls: ~%d\n", totalAPICalls)
	if estimatedMinutes > 0 {
		fmt.Printf("   ⏳ Time: ~%d minutes\n", estimatedMinutes)
	} else {
		fmt.Printf("   ⏳ Time: ~%d seconds\n", int(estimatedTimeSeconds))
	}
}

func isValidDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func isValidInterval(interval string) bool {
	validIntervals := []string{"minute", "3minute", "5minute", "10minute", "15minute", "30minute", "60minute", "day"}
	for _, valid := range validIntervals {
		if interval == valid {
			return true
		}
	}
	return false
}

func isValidStorageType(storageType string) bool {
	if storageType == "" {
		return true // Default is valid
	}
	validTypes := []string{"duckdb", "sqlite", "json", "csv"}
	for _, valid := range validTypes {
		if storageType == valid {
			return true
		}
	}
	return false
}
