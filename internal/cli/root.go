package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	appName        = "zerodha-connect"
	appDescription = "A high-performance market data fetcher for Zerodha Kite API"
	version        = "2.0.0"
)

var (
	configFile string
	verbose    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   appName,
	Short: "Fetch market data from Zerodha Kite API",
	Long: fmt.Sprintf(`%s - %s

A modular, enterprise-grade application for fetching historical market data 
from the Zerodha Kite API with support for multiple storage backends:

• DuckDB - Fast analytical database (default)
• SQLite - Universal database format  
• JSON - Human-readable format (one file per instrument)
• CSV - Excel-compatible format (one file per instrument)

Features:
- Rate-limited API calls respecting Zerodha limits
- Configurable date chunking for optimal performance
- Authentication flow with token caching
- Comprehensive logging and error handling
- Multiple storage options for different use cases`, appName, appDescription),
	Version: version,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	// Add subcommands
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(storageCmd)
	rootCmd.AddCommand(profileCmd)
}
