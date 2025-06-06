package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// storageCmd represents the storage command
var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Information about storage backends",
	Long: `Display information about available storage backends and their use cases.

This command helps you choose the right storage backend for your needs.`,
	RunE: runStorage,
}

func runStorage(cmd *cobra.Command, args []string) error {
	fmt.Println("📦 Available Storage Backends")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Println("\n🚀 DuckDB (default)")
	fmt.Println("  - Best for: Analytical queries, time series analysis")
	fmt.Println("  - Format: Single database file (.duckdb)")
	fmt.Println("  - Pros: Fast aggregations, SQL queries, columnar storage")
	fmt.Println("  - Cons: Requires DuckDB to query")
	fmt.Println("  - Example: storage_type: \"duckdb\", storage_path: \"market_data.duckdb\"")

	fmt.Println("\n💾 SQLite")
	fmt.Println("  - Best for: Universal compatibility, small to medium datasets")
	fmt.Println("  - Format: Single database file (.sqlite)")
	fmt.Println("  - Pros: Widely supported, portable, SQL queries")
	fmt.Println("  - Cons: Slower for large analytical workloads")
	fmt.Println("  - Example: storage_type: \"sqlite\", storage_path: \"market_data.sqlite\"")

	fmt.Println("\n📄 JSON")
	fmt.Println("  - Best for: Human-readable data, debugging, small datasets")
	fmt.Println("  - Format: One JSON file per instrument")
	fmt.Println("  - Pros: Human-readable, easy to inspect, no database required")
	fmt.Println("  - Cons: Large file sizes, slower queries")
	fmt.Println("  - Example: storage_type: \"json\", storage_path: \"./data/json/\"")

	fmt.Println("\n📊 CSV")
	fmt.Println("  - Best for: Excel compatibility, data analysis tools")
	fmt.Println("  - Format: One CSV file per instrument")
	fmt.Println("  - Pros: Excel/spreadsheet compatible, widely supported")
	fmt.Println("  - Cons: No data types, larger files, manual schema")
	fmt.Println("  - Example: storage_type: \"csv\", storage_path: \"./data/csv/\"")

	fmt.Println("\n💡 Recommendations:")
	fmt.Println("  - For backtesting/analysis: DuckDB")
	fmt.Println("  - For universal compatibility: SQLite")
	fmt.Println("  - For Excel analysis: CSV")
	fmt.Println("  - For inspection/debugging: JSON")

	return nil
}
