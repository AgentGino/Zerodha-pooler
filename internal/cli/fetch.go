package cli

import (
	"fmt"
	"log"
	"time"

	"zerodha-connect/internal/config"
	"zerodha-connect/internal/kite"
	"zerodha-connect/internal/logger"
	"zerodha-connect/internal/storage"
	"zerodha-connect/internal/ui"

	"github.com/spf13/cobra"
)

var (
	// Fetch data command flags
	instruments    []string
	fromDate       string
	toDate         string
	interval       string
	storageType    string
	storagePath    string
	skipConfirm    bool
	apiKey         string
	apiSecret      string
	dataConfigFile string
)

const (
	MaxCandlesPerRequest  = 22500
	InstrumentsPerRequest = 1
)

// fetchCmd represents the parent fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch data from Zerodha Kite API",
	Long: `Fetch data from Zerodha Kite API.

This command has two subcommands:
- instruments: Download and cache instrument list
- data: Fetch historical market data

Use "zerodha-connect fetch [subcommand] --help" for more information.`,
}

// fetchInstrumentsCmd represents the instruments command
var fetchInstrumentsCmd = &cobra.Command{
	Use:   "instruments",
	Short: "Download and cache instrument list",
	Long: `Download the complete instrument list from Zerodha Kite API and cache it locally.

This command fetches all available instruments and saves them to instruments_cache.json.
The cache is used by other commands to validate instrument symbols and get token mappings.

API credentials can be provided via config file or command line flags.

Examples:
  # Download instruments using API credentials from config file
  zerodha-connect fetch instruments

  # Download instruments using command line flags
  zerodha-connect fetch instruments --api-key YOUR_KEY --api-secret YOUR_SECRET

  # Use specific config file
  zerodha-connect fetch instruments --config my-config.yaml`,
	RunE: runFetchInstruments,
}

// fetchDataCmd represents the data command
var fetchDataCmd = &cobra.Command{
	Use:   "data",
	Short: "Fetch historical market data",
	Long: `Fetch historical market data from Zerodha Kite API.

This command fetches OHLCV (Open, High, Low, Close, Volume) data for specified 
instruments and stores it in your chosen format. The data is fetched in chunks
to respect API rate limits and optimize performance.

Examples:
  # Fetch data using config file
  zerodha-connect fetch data -f config.yaml

  # Fetch specific instruments with CLI flags
  zerodha-connect fetch data --instruments SBIN,RELIANCE --from 2024-01-01 --to 2024-01-31

  # Use different storage backend
  zerodha-connect fetch data --storage-type csv --storage-path ./data/csv

  # Skip confirmation prompt
  zerodha-connect fetch data --yes`,
	RunE: runFetchData,
}

func runFetchInstruments(cmd *cobra.Command, args []string) error {
	var apiKeyToUse, apiSecretToUse string

	// Try to load config file if it exists, otherwise use flags
	if conf, err := config.Load(configFile); err == nil {
		// Config file exists, use those credentials as defaults
		apiKeyToUse = conf.APIKey
		apiSecretToUse = conf.APISecret
	}

	// Override with command line flags if provided
	if apiKey != "" {
		apiKeyToUse = apiKey
	}
	if apiSecret != "" {
		apiSecretToUse = apiSecret
	}

	// Validate we have API credentials
	if apiKeyToUse == "" || apiSecretToUse == "" {
		return fmt.Errorf("API credentials required. Provide them via:\n" +
			"  • Config file (api_key and api_secret fields)\n" +
			"  • Command flags: --api-key and --api-secret")
	}

	// Create minimal config for authentication
	tempConfig := &config.Config{
		APIKey:    apiKeyToUse,
		APISecret: apiSecretToUse,
		LogFile:   "instruments_fetch.log",
	}

	// Initialize silent logger for technical details
	appLogger := logger.NewSilent()
	if verbose {
		// Only use verbose logger if explicitly requested
		appLogger = logger.New(tempConfig.LogFile)
		appLogger.Println("🔧 Verbose mode enabled")
	}

	fmt.Println("📦 Downloading instruments...")

	// Initialize Kite client
	kiteClient := kite.NewClientWithConfigPath(tempConfig, appLogger, configFile)
	if err := kiteClient.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}
	fmt.Println("✅ API authentication successful")

	// Download instruments
	instruments, err := kite.GetInstruments(kiteClient.GetKiteConnectClient(), appLogger)
	if err != nil {
		return fmt.Errorf("failed to get instruments: %v", err)
	}

	fmt.Printf("✅ Successfully downloaded and cached %d instruments\n", len(instruments))
	return nil
}

func runFetchData(cmd *cobra.Command, args []string) error {
	// Determine config file - use -f flag if provided, otherwise global --config
	configPath := configFile
	if dataConfigFile != "" {
		configPath = dataConfigFile
	}

	// Load configuration
	conf, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file '%s': %v", configPath, err)
	}

	// Override config with CLI flags if provided
	if len(instruments) > 0 {
		conf.Instruments = instruments
	}
	if fromDate != "" {
		conf.FromDate = fromDate
	}
	if toDate != "" {
		conf.ToDate = toDate
	}
	if interval != "" {
		conf.Interval = interval
	}
	if storageType != "" {
		conf.StorageType = storageType
	}
	if storagePath != "" {
		conf.StoragePath = storagePath
	}
	if apiKey != "" {
		conf.APIKey = apiKey
	}
	if apiSecret != "" {
		conf.APISecret = apiSecret
	}

	// Perform comprehensive validation
	validation := conf.ValidateComplete()
	if validation.HasErrors() {
		fmt.Println("❌ Configuration validation failed:")
		for _, err := range validation.Errors {
			fmt.Printf("  - %s\n", err.Error())
		}
		return fmt.Errorf("configuration has %d validation error(s)", len(validation.Errors))
	}

	// Initialize silent logger for technical details
	appLogger := logger.NewSilent()
	if verbose {
		// Only use verbose logger if explicitly requested
		appLogger = logger.New(conf.LogFile)
		appLogger.Println("🔧 Verbose mode enabled")
	}

	fmt.Println("🚀 Starting market data fetch...")

	// Services Initialization
	kiteClient := kite.NewClientWithConfigPath(conf, appLogger, configPath)
	if err := kiteClient.Authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}
	fmt.Println("✅ API authentication successful")

	// Determine storage configuration
	storageType := storage.StorageType(conf.StorageType)
	storagePath := conf.StoragePath

	// Backward compatibility with old DuckDB config
	if storagePath == "" && conf.DuckDBPath != "" {
		storagePath = conf.DuckDBPath
		storageType = storage.StorageTypeDuckDB
		if verbose {
			fmt.Println("⚠️  Using deprecated 'duckdb_path' config. Please use 'storage_type' and 'storage_path' instead.")
		}
	}

	// Default storage settings
	if storageType == "" {
		storageType = storage.StorageTypeDuckDB
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

	dbStore, err := storage.NewStore(storageType, storagePath, appLogger)
	if err != nil {
		return fmt.Errorf("failed to initialize %s store: %v", storageType, err)
	}
	defer dbStore.Close()
	if err := dbStore.Init(); err != nil {
		return fmt.Errorf("failed to initialize %s storage: %v", storageType, err)
	}

	fmt.Printf("📦 Using %s storage: %s\n", storageType, storagePath)

	// Instrument Discovery
	fmt.Println("🔍 Loading instruments...")
	instruments, err := kite.GetInstruments(kiteClient.GetKiteConnectClient(), appLogger)
	if err != nil {
		return fmt.Errorf("failed to get instruments: %v", err)
	}
	instrumentTokenMap := make(map[string]int)
	for _, instr := range instruments {
		instrumentTokenMap[instr.Tradingsymbol] = int(instr.InstrumentToken)
	}
	fmt.Printf("✅ Loaded %d instruments\n", len(instruments))

	// Execution Plan - dates are already validated
	from, _ := time.Parse("2006-01-02", conf.FromDate)
	to, _ := time.Parse("2006-01-02", conf.ToDate)

	totalAPICalls, validInstruments := calculateAPICalls(conf, instrumentTokenMap, from, to, appLogger)
	if validInstruments == 0 {
		return fmt.Errorf("no valid instruments found to process")
	}

	// User Confirmation
	if !skipConfirm && !confirmPlan(conf, validInstruments, totalAPICalls) {
		fmt.Println("❌ Operation cancelled by user")
		return nil
	}

	fmt.Printf("📊 Fetching data for %d instruments...\n", validInstruments)

	// Data Fetching Loop
	runFetchingLoop(conf, instrumentTokenMap, kiteClient, dbStore, from, to, appLogger)

	fmt.Println("✅ Market data fetch completed successfully!")
	return nil
}

func calculateAPICalls(conf *config.Config, tokenMap map[string]int, from, to time.Time, logger *log.Logger) (int, int) {
	totalAPICalls := 0
	validInstruments := 0

	if verbose {
		logger.Println("📊 Calculating API calls needed...")
	}

	var invalidInstruments []string
	for _, instrumentSymbol := range conf.Instruments {
		if _, ok := tokenMap[instrumentSymbol]; !ok {
			invalidInstruments = append(invalidInstruments, instrumentSymbol)
			if verbose {
				logger.Printf("⚠️  %s not found in instrument list. Will skip.", instrumentSymbol)
			}
			continue
		}
		validInstruments++
		chunks := kite.GenerateDateChunks(from, to, conf.Interval)
		totalAPICalls += len(chunks)
		if verbose {
			logger.Printf("  \\_ %s: %d chunks needed", instrumentSymbol, len(chunks))
		}
	}

	if len(invalidInstruments) > 0 && !verbose {
		fmt.Printf("⚠️  %d invalid instruments will be skipped\n", len(invalidInstruments))
	}

	return totalAPICalls, validInstruments
}

func confirmPlan(conf *config.Config, validInstruments, totalAPICalls int) bool {
	estimatedTimeSeconds := float64(totalAPICalls) / float64(kite.RateLimitRequestsPerSecond)
	estimatedMinutes := int(estimatedTimeSeconds / 60)
	estimatedRemainingSeconds := int(estimatedTimeSeconds) % 60

	var chunkExplanation, chunkSizeInfo string
	if kite.IsDailyOrLarger(conf.Interval) {
		chunkSizeInfo = fmt.Sprintf("%d days per chunk", kite.DailyChunkDays)
		chunkExplanation = "Daily+ intervals: Zerodha allows multiple years per request"
	} else {
		chunkSizeInfo = fmt.Sprintf("%d days per chunk", kite.IntradayMaxDays)
		chunkExplanation = fmt.Sprintf("Intraday intervals: Zerodha limit is %d days per request (~%d candles max)",
			kite.IntradayMaxDays, MaxCandlesPerRequest)
	}

	plan := ui.FetchPlan{
		ValidInstruments:          validInstruments,
		FromDate:                  conf.FromDate,
		ToDate:                    conf.ToDate,
		Interval:                  conf.Interval,
		RateLimitPerSecond:        kite.RateLimitRequestsPerSecond,
		ChunkExplanation:          chunkExplanation,
		ChunkSizeInfo:             chunkSizeInfo,
		InstrumentsPerRequest:     InstrumentsPerRequest,
		TotalAPICalls:             totalAPICalls,
		EstimatedMinutes:          estimatedMinutes,
		EstimatedRemainingSeconds: estimatedRemainingSeconds,
	}
	return ui.ConfirmExecution(plan)
}

func runFetchingLoop(conf *config.Config, tokenMap map[string]int, client *kite.Client, store storage.Store, from, to time.Time, logger *log.Logger) {
	totalInstruments := len(conf.Instruments)
	processedInstruments := 0
	totalCandles := 0

	for i, instrumentSymbol := range conf.Instruments {
		token, ok := tokenMap[instrumentSymbol]
		if !ok {
			continue // Already logged in calculation step
		}

		processedInstruments++

		if verbose {
			fmt.Printf("📈 [%d/%d] Processing %s...\n", processedInstruments, len(conf.Instruments), instrumentSymbol)
			logger.Printf("[%d/%d] %s - Processing", i+1, totalInstruments, instrumentSymbol)
		} else {
			// Show progress every 10% or for the last instrument
			progress := (processedInstruments * 100) / len(conf.Instruments)
			interval := len(conf.Instruments) / 10
			if interval < 1 {
				interval = 1
			}
			if processedInstruments%interval == 0 || processedInstruments == len(conf.Instruments) {
				fmt.Printf("📊 Progress: %d%% (%d/%d instruments)\n", progress, processedInstruments, len(conf.Instruments))
			}
		}

		chunks := kite.GenerateDateChunks(from, to, conf.Interval)
		var totalInserted int

		for chunkIdx, chunk := range chunks {
			chunkFrom, chunkTo := chunk[0], chunk[1]

			if verbose {
				logger.Printf("  \\_ Chunk %d/%d: %s to %s", chunkIdx+1, len(chunks),
					chunkFrom.Format("2006-01-02"), chunkTo.Format("2006-01-02"))
			}

			candles, err := client.GetHistoricalData(token, conf.Interval, chunkFrom, chunkTo)
			if err != nil {
				if verbose {
					logger.Printf("    \\_ API error: %v", err)
					fmt.Printf("   ⚠️  API error for %s chunk %d/%d\n", instrumentSymbol, chunkIdx+1, len(chunks))
				}
				continue
			}

			if len(candles) == 0 {
				if verbose {
					logger.Printf("    \\_ No data for this chunk (likely non-trading days)")
				}
				continue
			}

			if verbose {
				logger.Printf("    \\_ API returned %d candles from %s to %s",
					len(candles),
					candles[0].Date.Time.Format("2006-01-02 15:04:05"),
					candles[len(candles)-1].Date.Time.Format("2006-01-02 15:04:05"))
			}

			inserted, err := store.StoreCandles(instrumentSymbol, candles)
			if err != nil {
				if verbose {
					logger.Printf("    \\_ DB store error: %v", err)
					fmt.Printf("   ⚠️  Storage error for %s chunk %d/%d\n", instrumentSymbol, chunkIdx+1, len(chunks))
				}
			} else {
				if verbose {
					logger.Printf("    \\_ Inserted %d candles", inserted)
				}
				totalInserted += inserted
			}
		}

		if verbose {
			logger.Printf("  \\_ Total inserted for %s: %d candles", instrumentSymbol, totalInserted)
			fmt.Printf("   ✅ Saved %d candles for %s\n", totalInserted, instrumentSymbol)
		}
		totalCandles += totalInserted
	}

	fmt.Printf("🎯 Completed: %d candles saved for %d instruments\n", totalCandles, processedInstruments)
}

func init() {
	// Add subcommands to fetch
	fetchCmd.AddCommand(fetchInstrumentsCmd)
	fetchCmd.AddCommand(fetchDataCmd)

	// Fetch instruments command flags
	fetchInstrumentsCmd.Flags().StringVar(&apiKey, "api-key", "", "Zerodha API key")
	fetchInstrumentsCmd.Flags().StringVar(&apiSecret, "api-secret", "", "Zerodha API secret")

	// Fetch data command flags
	fetchDataCmd.Flags().StringVarP(&dataConfigFile, "file", "f", "", "config file path")
	fetchDataCmd.Flags().StringSliceVarP(&instruments, "instruments", "i", []string{}, "comma-separated list of instruments (e.g. SBIN,RELIANCE)")
	fetchDataCmd.Flags().StringVarP(&fromDate, "from", "", "", "start date (YYYY-MM-DD)")
	fetchDataCmd.Flags().StringVarP(&toDate, "to", "", "", "end date (YYYY-MM-DD)")
	fetchDataCmd.Flags().StringVar(&interval, "interval", "", "data interval (minute, 5minute, day, etc.)")
	fetchDataCmd.Flags().StringVar(&storageType, "storage-type", "", "storage type (duckdb, sqlite, json, csv)")
	fetchDataCmd.Flags().StringVar(&storagePath, "storage-path", "", "storage path (file or directory)")
	fetchDataCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "skip confirmation prompt")
	fetchDataCmd.Flags().StringVar(&apiKey, "api-key", "", "Zerodha API key")
	fetchDataCmd.Flags().StringVar(&apiSecret, "api-secret", "", "Zerodha API secret")
}
