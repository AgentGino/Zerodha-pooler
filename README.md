# Zerodha Connect

A high-performance, Indian stock market data fetcher for the Zerodha Kite API with support for multiple storage backends.

## Features

- 🚀 **Multiple Storage Backends**: DuckDB, SQLite, JSON, CSV
- 📊 **Rate-Limited API Calls**: Respects Zerodha API limits
- 🔄 **Smart Chunking**: Optimizes requests based on data intervals
- 🔐 **Secure Authentication**: Token caching with automatic renewal
- 📝 **Comprehensive Logging**: Detailed operation tracking
- 🛠️ **CLI Interface**: Professional command-line experience
- 📦 **Modular Architecture**: Clean, maintainable codebase

## Installation

```bash
# Clone the repository
git clone https://github.com/your-repo/zerodha-connect
cd zerodha-connect

# Install dependencies
go mod download

# Build the application
go build -o zerodha-connect cmd/fetcher/main.go
```

## Quick Start

1. **Copy the example configuration:**
```bash
cp config.example.yaml config.yaml
```

2. **Edit the configuration with your API credentials:**
```yaml
api_key: "your_api_key_here"
api_secret: "your_api_secret_here"
instruments:
  - "SBIN"
  - "RELIANCE"
from_date: "2024-01-01"
to_date: "2024-01-31"
interval: "minute"
storage_type: "duckdb"
storage_path: "market_data.duckdb"
```

3. **Validate your configuration:**
```bash
./zerodha-connect validate
```

4. **Download instruments list:**
```bash
./zerodha-connect fetch instruments
```

5. **Fetch market data:**
```bash
./zerodha-connect fetch data -f config.yaml
```

## CLI Commands

### Main Commands

#### `fetch` - Fetch Data from Zerodha API

The fetch command has two subcommands:

##### `fetch instruments` - Download Instruments
```bash
# Download instruments using API credentials from config file
./zerodha-connect fetch instruments

# Download instruments using command line flags
./zerodha-connect fetch instruments --api-key YOUR_KEY --api-secret YOUR_SECRET

# Use specific config file
./zerodha-connect fetch instruments --config my-config.yaml
```

Downloads and caches the complete instrument list from Zerodha API to `instruments_cache.json` for symbol validation and token mapping. Can work standalone with just API credentials.

##### `fetch data` - Fetch Historical Data
```bash
# Fetch using config file
./zerodha-connect fetch data -f config.yaml

# Override config with command-line flags
./zerodha-connect fetch data --instruments SBIN,RELIANCE --from 2024-01-01 --to 2024-01-31

# Use different storage backend
./zerodha-connect fetch data --storage-type csv --storage-path ./data/csv

# Skip confirmation prompt
./zerodha-connect fetch data --yes
```

**Flags:**
- `-f, --file`: Config file path (specific to fetch-data)
- `--instruments, -i`: Comma-separated instrument list
- `--from`: Start date (YYYY-MM-DD)
- `--to`: End date (YYYY-MM-DD)
- `--interval`: Data interval (minute, 5minute, day, etc.)
- `--storage-type`: Storage backend (duckdb, sqlite, json, csv)
- `--storage-path`: Path to database file or directory
- `--yes, -y`: Skip confirmation prompt
- `--api-key`: Zerodha API key
- `--api-secret`: Zerodha API secret

#### `validate` - Validate Configuration
```bash
# Validate default config
./zerodha-connect validate

# Validate specific config file
./zerodha-connect validate --config my-config.yaml
```

Performs comprehensive validation:
- Configuration file format
- API connectivity
- Instrument symbols
- Storage backend accessibility
- Date ranges and intervals

#### `storage` - Storage Information
```bash
./zerodha-connect storage
```

Displays detailed information about available storage backends and their use cases.

### Global Flags

- `--config, -c`: Configuration file path (default: config.yaml)
- `--verbose, -v`: Enable verbose logging
- `--help, -h`: Show help
- `--version`: Show version

## Storage Backends

### 🚀 DuckDB (Recommended)
- **Best for**: Analytical queries, time series analysis
- **Format**: Single database file (.duckdb)
- **Pros**: Fast aggregations, SQL queries, columnar storage
- **Example**: 
  ```yaml
  storage_type: "duckdb"
  storage_path: "market_data.duckdb"
  ```

### 💾 SQLite
- **Best for**: Universal compatibility, portability
- **Format**: Single database file (.sqlite)
- **Pros**: Widely supported, portable, SQL queries
- **Example**:
  ```yaml
  storage_type: "sqlite"
  storage_path: "market_data.sqlite"
  ```

### 📄 JSON
- **Best for**: Human-readable data, debugging
- **Format**: One JSON file per instrument
- **Pros**: Human-readable, easy to inspect
- **Example**:
  ```yaml
  storage_type: "json"
  storage_path: "./data/json/"
  ```

### 📊 CSV
- **Best for**: Excel compatibility, data analysis
- **Format**: One CSV file per instrument
- **Pros**: Excel/spreadsheet compatible
- **Example**:
  ```yaml
  storage_type: "csv"
  storage_path: "./data/csv/"
  ```

## Configuration

### Complete Configuration Example

```yaml
# API Configuration
api_key: "your_api_key_here"
api_secret: "your_api_secret_here"
access_token: ""  # Auto-populated after first login

# Data Configuration
instruments:
  - "SBIN"
  - "RELIANCE"
  - "TCS"
  - "INFY"
from_date: "2024-01-01"
to_date: "2024-01-31"
interval: "minute"  # minute, 3minute, 5minute, 10minute, 15minute, 30minute, 60minute, day

# Storage Configuration
storage_type: "duckdb"  # duckdb, sqlite, json, csv
storage_path: "market_data.duckdb"

# Logging
log_file: "kite_fetcher.log"
```

### Configuration Validation

The application performs comprehensive validation of your configuration:

#### **Required Fields:**
- `api_key` - Your Zerodha API key
- `api_secret` - Your Zerodha API secret  
- `instruments` - At least one trading symbol
- `from_date` - Start date in YYYY-MM-DD format
- `to_date` - End date in YYYY-MM-DD format
- `interval` - Data interval (minute, 3minute, 5minute, 10minute, 15minute, 30minute, 60minute, day)

#### **Format Validation:**
- **Dates**: Must be in `YYYY-MM-DD` format
- **Date Range**: `from_date` must be before `to_date`
- **Intervals**: Must be one of the supported intervals
- **Storage Types**: Must be `duckdb`, `sqlite`, `json`, or `csv`
- **Instruments**: Non-empty symbols, max 20 characters each

#### **Path Validation:**
- **Storage Paths**: Validates write permissions and creates directories if needed
- **Log Files**: Ensures log directory is writable

#### **Live Validation** (via `validate` command):
- **API Connectivity**: Tests authentication with your credentials
- **Instrument Symbols**: Validates against live instrument list from Zerodha
- **Storage Backend**: Tests actual storage initialization

### Data Intervals

- `minute`: 1-minute candles
- `3minute`: 3-minute candles
- `5minute`: 5-minute candles
- `10minute`: 10-minute candles
- `15minute`: 15-minute candles
- `30minute`: 30-minute candles
- `60minute`: 1-hour candles
- `day`: Daily candles

## API Rate Limits

The application automatically handles Zerodha API rate limits:
- **3 requests per second** maximum
- **60 days** of intraday data per request
- **2000 days** of daily data per request
- Smart chunking based on interval type

## Examples

### Complete Workflow
```bash
# 1. Validate configuration
./zerodha-connect validate

# 2. Download instruments (one-time setup)
./zerodha-connect fetch instruments

# 3. Fetch market data
./zerodha-connect fetch data -f config.yaml
```

### Fetch Multiple Timeframes
```bash
# Fetch minute data
./zerodha-connect fetch data --interval minute --storage-path minute_data.duckdb

# Fetch daily data
./zerodha-connect fetch data --interval day --storage-path daily_data.duckdb
```

### Use Different Storage for Different Purposes
```bash
# DuckDB for analysis
./zerodha-connect fetch data --storage-type duckdb --storage-path analysis.duckdb

# CSV for Excel
./zerodha-connect fetch data --storage-type csv --storage-path ./excel_data/

# JSON for inspection
./zerodha-connect fetch data --storage-type json --storage-path ./debug_data/
```

### Automated Workflows
```bash
# Complete workflow with validation
./zerodha-connect validate && \
./zerodha-connect fetch instruments && \
./zerodha-connect fetch data -f config.yaml --yes
```

## Troubleshooting

### Configuration Validation

**Always run validation first when encountering issues:**
```bash
./zerodha-connect validate
```

This performs comprehensive checks and provides detailed error messages for:
- Missing or invalid configuration fields
- Date format and range issues  
- Invalid intervals or storage types
- API connectivity problems
- Storage permission issues
- Invalid instrument symbols

### Common Issues

1. **Configuration Validation Failed**
   - Check the specific error messages from `validate` command
   - Ensure all required fields are present and correctly formatted
   - Verify date format is YYYY-MM-DD
   - Check that interval and storage_type are valid options

2. **Authentication Failed**
   - Verify API key and secret
   - Check if access token needs renewal
   - Run `./zerodha-connect validate` to test connectivity

3. **Invalid Instruments**
   - Run `./zerodha-connect fetch instruments` to download latest instrument list
   - Use `./zerodha-connect validate` to check instrument symbols
   - Ensure symbols match Zerodha's format exactly

4. **Storage Issues**
   - Check directory permissions for file-based storage
   - Ensure sufficient disk space
   - Verify database file is not locked by another process
   - Run validation to test storage accessibility

### Debug Mode
```bash
./zerodha-connect fetch data -f config.yaml --verbose
```

## License

MIT

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

For issues and questions:
- Create an issue on GitHub
- Check existing documentation
- Use `./zerodha-connect --help` for command help 
