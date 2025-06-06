package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

// New initializes and returns a new logger that writes to both stdout and a log file.
func New(logPath string) *log.Logger {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(mw, " ", log.LstdFlags|log.Lshortfile)
	return logger
}

// NewSilent creates a logger that discards all output (for clean console output)
func NewSilent() *log.Logger {
	return log.New(io.Discard, "", 0)
}
