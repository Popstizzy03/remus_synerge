// pkg/logger/logger.go
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// New creates a new zerolog.Logger instance.
func New() zerolog.Logger {
	// Check for a development environment variable to set the log level
	logLevel := zerolog.InfoLevel
	if os.Getenv("APP_ENV") == "development" {
		logLevel = zerolog.DebugLevel
	}

	// Use console writer for development for human-readable logs
	if os.Getenv("APP_ENV") == "development" {
		return log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			Level(logLevel).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	// Default to JSON logger for production
	return zerolog.New(os.Stderr).
		Level(logLevel).
		With().
		Timestamp().
		Logger()
}
