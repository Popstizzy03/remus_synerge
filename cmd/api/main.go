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
	// Initialize logger
	l := logger.New()
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to load configuration")
	}
	
	// Initialize database connection
	db, err := database.NewPostgresClient(cfg.Database)
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()
	
	// Create and start the server
	srv := server.New(cfg, db, l)
	
	// Start server in a goroutine
	go func() {
		l.Info().Str("address", cfg.Server.Address).Msg("Starting server")
		if err := srv.Start(); err != nil {
			l.Fatal().Err(err).Msg("Server failed")
		}
	}()
	
	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	l.Info().Msg("Shutting down server...")
	
	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		l.Fatal().Err(err).Msg("Server forced to shutdown")
	}
	
	l.Info().Msg("Server exiting")
}