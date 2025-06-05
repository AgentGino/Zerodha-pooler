package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey      string   `yaml:"api_key"`
	APISecret   string   `yaml:"api_secret"`
	AccessToken string   `yaml:"access_token"`
	Instruments []string `yaml:"instruments"`
	FromDate    string   `yaml:"from_date"`
	ToDate      string   `yaml:"to_date"`
	Interval    string   `yaml:"interval"`
	DuckDBPath  string   `yaml:"duckdb_path"`
	LogFile     string   `yaml:"log_file"`
}

func readConfig(path string) (*Config, error) {
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

func saveConfig(path string, conf *Config) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func authenticateKite(conf *Config, logger *log.Logger) error {
	// Check if we already have a valid access token
	if conf.AccessToken != "" {
		logger.Println("✅ Access token found in config. Proceeding...")
		return nil
	}

	logger.Println("🔐 No access token found. Starting authentication flow...")

	if conf.APIKey == "" || conf.APISecret == "" {
		return fmt.Errorf("API key and API secret are required for authentication")
	}

	// Create Kite instance
	kc := kiteconnect.New(conf.APIKey)

	// Generate login URL
	loginURL := kc.GetLoginURL()
	logger.Printf("🌐 Opening browser for Zerodha login: %s", loginURL)

	// Open browser
	if err := openBrowser(loginURL); err != nil {
		logger.Printf("⚠️  Failed to open browser automatically: %v", err)
		logger.Printf("Please manually open this URL in your browser: %s", loginURL)
	}

	// Wait for user to provide request token
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🔑 AUTHENTICATION REQUIRED")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("1. Login to Zerodha in the browser that just opened")
	fmt.Println("2. After successful login, you'll be redirected to a URL")
	fmt.Println("3. Copy the 'request_token' parameter from the redirected URL")
	fmt.Println("4. Paste it below and press Enter")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Print("Enter request token: ")

	reader := bufio.NewReader(os.Stdin)
	requestToken, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read request token: %v", err)
	}
	requestToken = strings.TrimSpace(requestToken)

	if requestToken == "" {
		return fmt.Errorf("request token cannot be empty")
	}

	logger.Printf("🔄 Exchanging request token for access token...")

	// Generate session (access token)
	data, err := kc.GenerateSession(requestToken, conf.APISecret)
	if err != nil {
		return fmt.Errorf("failed to generate session: %v", err)
	}

	// Save access token to config
	conf.AccessToken = data.AccessToken
	if err := saveConfig("config.yaml", conf); err != nil {
		return fmt.Errorf("failed to save access token to config: %v", err)
	}

	logger.Println("✅ Authentication successful! Access token saved to config.yaml")
	logger.Printf("🎯 Access token: %s", data.AccessToken)

	return nil
}

func parseIntervalMinutes(interval string) int {
	switch interval {
	case "minute":
		return 1
	case "3minute":
		return 3
	case "5minute":
		return 5
	case "10minute":
		return 10
	case "15minute":
		return 15
	case "30minute":
		return 30
	case "60minute", "hour":
		return 60
	case "day":
		return 1440
	default:
		return 1 // default to 1 minute if unknown
	}
}

func generateDateChunks(from, to time.Time, interval string) [][2]time.Time {
	intervalMinutes := parseIntervalMinutes(interval)
	var chunkSize time.Duration

	if intervalMinutes >= 1440 { // Daily or larger
		chunkSize = 365 * 24 * time.Hour // 1 year chunks for daily data
	} else {
		// Calculate safe chunk size to stay under 10k candles
		candlesPerDay := 1440 / intervalMinutes
		maxDays := 10000 / candlesPerDay
		if maxDays < 1 {
			maxDays = 1
		}
		chunkSize = time.Duration(maxDays*24) * time.Hour
	}

	var chunks [][2]time.Time
	currentStart := from

	for currentStart.Before(to) {
		currentEnd := currentStart.Add(chunkSize)
		if currentEnd.After(to) {
			currentEnd = to
		}
		chunks = append(chunks, [2]time.Time{currentStart, currentEnd})
		currentStart = currentEnd.Add(time.Second) // Avoid overlap
	}

	return chunks
}

func setupLogger(logPath string) *log.Logger {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(mw, " ", log.LstdFlags|log.Lshortfile)
	return logger
}

func main() {
	conf, err := readConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}
	logger := setupLogger(conf.LogFile)
	logger.Println("🚀 Starting Kite Fetcher. May the API rate limits be ever in your favor.")

	// Handle authentication
	if err := authenticateKite(conf, logger); err != nil {
		logger.Fatalf("Authentication failed: %v", err)
	}

	// Connect to DuckDB
	db, err := sql.Open("duckdb", conf.DuckDBPath)
	if err != nil {
		logger.Fatalf("DuckDB connection failed: %v", err)
	}
	defer db.Close()

	// Create table if not exists
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
	if _, err := db.Exec(createTable); err != nil {
		logger.Fatalf("Failed to create DuckDB table: %v", err)
	}

	// Set up Zerodha client
	kc := kiteconnect.New(conf.APIKey)
	kc.SetAccessToken(conf.AccessToken)

	var instrumentsList []kiteconnect.Instrument
	const instrumentCacheFile = "instrument_cache.json"

	// Try to load instruments from cache
	cachedData, err := os.ReadFile(instrumentCacheFile)
	if err == nil {
		if unmarshalErr := json.Unmarshal(cachedData, &instrumentsList); unmarshalErr == nil && len(instrumentsList) > 0 {
			logger.Printf("Successfully loaded %d instruments from cache: %s", len(instrumentsList), instrumentCacheFile)
		} else {
			if unmarshalErr != nil {
				logger.Printf("Error unmarshaling cached instruments from %s: %v. Will fetch from API.", instrumentCacheFile, unmarshalErr)
			} else { // len == 0
				logger.Printf("Instrument cache %s is empty. Will fetch from API.", instrumentCacheFile)
			}
			instrumentsList = nil // Ensure it's nil to trigger API fetch
		}
	} else {
		logger.Printf("Cache file %s not found or error reading: %v. Will fetch from API.", instrumentCacheFile, err)
	}

	if instrumentsList == nil {
		logger.Println("Fetching instrument list from API...")
		apiInstruments, fetchErr := kc.GetInstruments()
		if fetchErr != nil {
			logger.Fatalf("Failed to fetch instruments from API: %v", fetchErr)
		}
		instrumentsList = apiInstruments
		logger.Printf("Successfully fetched %d instruments from API.", len(instrumentsList))

		// Save to cache for next time
		jsonData, marshalErr := json.MarshalIndent(instrumentsList, "", "  ")
		if marshalErr != nil {
			logger.Printf("Warning: Failed to marshal instruments for caching: %v", marshalErr)
		} else {
			if writeErr := os.WriteFile(instrumentCacheFile, jsonData, 0644); writeErr != nil {
				logger.Printf("Warning: Failed to write instrument cache to %s: %v", instrumentCacheFile, writeErr)
			} else {
				logger.Printf("Successfully saved %d instruments to cache: %s", len(instrumentsList), instrumentCacheFile)
			}
		}
	}

	// Create a map for quick lookup
	instrumentTokenMap := make(map[string]int)
	for _, instr := range instrumentsList {
		instrumentTokenMap[instr.Tradingsymbol] = instr.InstrumentToken
	}
	logger.Printf("Successfully fetched and mapped %d instruments.", len(instrumentsList))

	// Set up rate limiter (now accounts for multiple requests per instrument)
	limiter := rate.NewLimiter(3, 1) // 3 req/sec, burst 1

	from, _ := time.Parse("2006-01-02", conf.FromDate)
	to, _ := time.Parse("2006-01-02", conf.ToDate)

	totalInstruments := len(conf.Instruments)
	for i, instrumentSymbol := range conf.Instruments {
		token, ok := instrumentTokenMap[instrumentSymbol]
		if !ok {
			logger.Printf("[%d/%d] %s not found. Skipping.", i+1, totalInstruments, instrumentSymbol)
			continue
		}

		logger.Printf("[%d/%d] %s - Processing", i+1, totalInstruments, instrumentSymbol)
		chunks := generateDateChunks(from, to, conf.Interval)

		var totalInserted int
		for chunkIdx, chunk := range chunks {
			chunkFrom, chunkTo := chunk[0], chunk[1]

			// Rate limit per chunk request
			if err := limiter.Wait(context.Background()); err != nil {
				logger.Printf("  \\_ Rate limit error (chunk %d): %v", chunkIdx+1, err)
				continue
			}

			logger.Printf("  \\_ Chunk %d/%d: %s to %s", chunkIdx+1, len(chunks),
				chunkFrom.Format("2006-01-02"),
				chunkTo.Format("2006-01-02"))

			candles, err := kc.GetHistoricalData(token, conf.Interval, chunkFrom, chunkTo, false, false)
			if err != nil {
				logger.Printf("    \\_ API error: %v", err)
				continue
			}

			logger.Printf("    \\_ API returned %d candles", len(candles))
			if len(candles) > 0 {
				logger.Printf("    \\_ Date range: %s to %s",
					candles[0].Date.Time.Format("2006-01-02 15:04:05"),
					candles[len(candles)-1].Date.Time.Format("2006-01-02 15:04:05"))
			}

			if len(candles) == 0 {
				logger.Printf("    \\_ No data for this chunk (likely non-trading days)")
				continue
			}

			// Database insertion per chunk
			tx, err := db.Begin()
			if err != nil {
				logger.Printf("    \\_ DB transaction error: %v", err)
				continue
			}

			stmt, err := tx.Prepare("INSERT INTO ohlcv VALUES (?,?,?,?,?,?,?)")
			if err != nil {
				tx.Rollback()
				logger.Printf("    \\_ DB prepare error: %v", err)
				continue
			}

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
					logger.Printf("      \\_ Insert error: %v", err)
				} else {
					inserted++
				}
			}

			stmt.Close()
			if err := tx.Commit(); err != nil {
				logger.Printf("    \\_ Commit error: %v", err)
			} else {
				logger.Printf("    \\_ Inserted %d candles", inserted)
				totalInserted += inserted
			}
		}
		logger.Printf("  \\_ Total inserted for %s: %d candles", instrumentSymbol, totalInserted)
	}
	logger.Println("✅ All data fetched and stored. May your backtests be profitable!")
}
