// cmd/api/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myapp/internal/api/server"
	"myapp/internal/config"
	"myapp/pkg/database"
	"myapp/pkg/logger"
)

func main() {
	// Internalize logger
	l := logger.New()

	// Load Configuration
	cfg, err := config.Load()
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize database connection
	
	
}