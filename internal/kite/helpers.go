package kite

import (
	"time"
)

const (
	// IntradayMaxDays is the max days per request for minute/intraday data.
	IntradayMaxDays = 60
	// DailyChunkDays is the chunk size for daily+ data (5 years).
	DailyChunkDays = 2000
)

// IsDailyOrLarger checks if the given interval is for daily data or larger.
func IsDailyOrLarger(interval string) bool {
	return parseIntervalMinutes(interval) >= 1440
}

func parseIntervalMinutes(interval string) int {
	intervalMap := map[string]int{
		"minute":   1,
		"3minute":  3,
		"5minute":  5,
		"10minute": 10,
		"15minute": 15,
		"30minute": 30,
		"60minute": 60,
		"hour":     60,
		"day":      1440,
	}

	if minutes, exists := intervalMap[interval]; exists {
		return minutes
	}
	return 1 // default to 1 minute if unknown
}

// GenerateDateChunks creates time chunks for API requests based on the interval.
func GenerateDateChunks(from, to time.Time, interval string) [][2]time.Time {
	var chunkSize time.Duration

	if IsDailyOrLarger(interval) { // Daily or larger
		chunkSize = DailyChunkDays * 24 * time.Hour
	} else {
		// For intraday intervals
		chunkSize = IntradayMaxDays * 24 * time.Hour
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
