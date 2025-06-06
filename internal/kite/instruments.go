package kite

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

const instrumentCacheFile = "instrument_cache.json"

// InstrumentCache represents a simplified instrument structure for caching
type InstrumentCache struct {
	InstrumentToken int     `json:"instrument_token"`
	ExchangeToken   int     `json:"exchange_token"`
	Tradingsymbol   string  `json:"tradingsymbol"`
	Name            string  `json:"name"`
	LastPrice       float64 `json:"last_price"`
	Expiry          string  `json:"expiry,omitempty"` // Store as string to avoid time parsing issues
	StrikePrice     float64 `json:"strike_price"`
	TickSize        float64 `json:"tick_size"`
	LotSize         float64 `json:"lot_size"`
	InstrumentType  string  `json:"instrument_type"`
	Segment         string  `json:"segment"`
	Exchange        string  `json:"exchange"`
}

// GetInstruments fetches the list of instruments, using a local cache if available.
func GetInstruments(kc *kiteconnect.Client, logger *log.Logger) ([]kiteconnect.Instrument, error) {
	var instrumentsList []kiteconnect.Instrument

	// Try to load instruments from cache
	cachedData, err := os.ReadFile(instrumentCacheFile)
	if err == nil {
		// Try to unmarshal as simplified cache format first
		var cachedInstruments []InstrumentCache
		if unmarshalErr := json.Unmarshal(cachedData, &cachedInstruments); unmarshalErr == nil && len(cachedInstruments) > 0 {
			// Convert cached instruments to kiteconnect.Instrument format
			instrumentsList = make([]kiteconnect.Instrument, len(cachedInstruments))
			for i, cached := range cachedInstruments {
				instrumentsList[i] = kiteconnect.Instrument{
					InstrumentToken: cached.InstrumentToken,
					ExchangeToken:   cached.ExchangeToken,
					Tradingsymbol:   cached.Tradingsymbol,
					Name:            cached.Name,
					LastPrice:       cached.LastPrice,
					StrikePrice:     cached.StrikePrice,
					TickSize:        cached.TickSize,
					LotSize:         cached.LotSize,
					InstrumentType:  cached.InstrumentType,
					Segment:         cached.Segment,
					Exchange:        cached.Exchange,
					// Skip Expiry field to avoid time parsing issues
				}
			}
			logger.Printf("Successfully loaded %d instruments from cache: %s", len(instrumentsList), instrumentCacheFile)
			return instrumentsList, nil
		} else {
			// Try the old format (direct kiteconnect.Instrument unmarshal)
			if unmarshalErr := json.Unmarshal(cachedData, &instrumentsList); unmarshalErr == nil && len(instrumentsList) > 0 {
				logger.Printf("Successfully loaded %d instruments from legacy cache: %s", len(instrumentsList), instrumentCacheFile)
				return instrumentsList, nil
			} else if unmarshalErr != nil {
				logger.Printf("Error unmarshaling cached instruments from %s: %v. Will fetch from API.", instrumentCacheFile, unmarshalErr)
			} else { // len == 0
				logger.Printf("Instrument cache %s is empty. Will fetch from API.", instrumentCacheFile)
			}
		}
	} else {
		if !os.IsNotExist(err) {
			logger.Printf("Error reading cache file %s: %v. Will fetch from API.", instrumentCacheFile, err)
		} else {
			logger.Printf("Cache file %s not found. Will fetch from API.", instrumentCacheFile)
		}
	}

	logger.Println("Fetching instrument list from API...")
	apiInstruments, fetchErr := kc.GetInstruments()
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch instruments from API: %v", fetchErr)
	}
	logger.Printf("Successfully fetched %d instruments from API.", len(apiInstruments))

	// Convert to simplified cache format to avoid time parsing issues
	cachedInstruments := make([]InstrumentCache, len(apiInstruments))
	for i, instr := range apiInstruments {
		expiryStr := ""
		if !instr.Expiry.Time.IsZero() {
			expiryStr = instr.Expiry.Time.Format("2006-01-02")
		}

		cachedInstruments[i] = InstrumentCache{
			InstrumentToken: instr.InstrumentToken,
			ExchangeToken:   instr.ExchangeToken,
			Tradingsymbol:   instr.Tradingsymbol,
			Name:            instr.Name,
			LastPrice:       instr.LastPrice,
			Expiry:          expiryStr,
			StrikePrice:     instr.StrikePrice,
			TickSize:        instr.TickSize,
			LotSize:         instr.LotSize,
			InstrumentType:  instr.InstrumentType,
			Segment:         instr.Segment,
			Exchange:        instr.Exchange,
		}
	}

	// Save simplified format to cache for next time
	jsonData, marshalErr := json.MarshalIndent(cachedInstruments, "", "  ")
	if marshalErr != nil {
		logger.Printf("Warning: Failed to marshal instruments for caching: %v", marshalErr)
	} else {
		if writeErr := os.WriteFile(instrumentCacheFile, jsonData, 0644); writeErr != nil {
			logger.Printf("Warning: Failed to write instrument cache to %s: %v", instrumentCacheFile, writeErr)
		} else {
			logger.Printf("Successfully saved %d instruments to cache: %s", len(apiInstruments), instrumentCacheFile)
		}
	}
	return apiInstruments, nil
}
