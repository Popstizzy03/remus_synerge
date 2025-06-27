// internal/api/server.go
package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"remus_synerge/internal/api/handlers"
	"remus_synerge/internal/api/middleware"
	"remus_synerge/internal/config"
	"remus_synerge/internal/repository"
)

// Server holds the dependencies for the HTTP server.

type Server struct {
	router *mux.Router
	server *http.Server
	logger zerolog.Logger
}

// New creates and configures a new server.
func New(cfg *config.Config, db *pgxpool.Pool, logger zerolog.Logger) *Server {
	r := mux.NewRouter()

	// Create repositories
	userRepo := repository.NewUserRepository(db)

	// Create handlers
	userHandler := handlers.NewUserHandler(userRepo, logger)

	// Register middleware
	r.Use(middleware.LoggingMiddleware(logger))

	// Register routes
	r.HandleFunc("/health", handlers.HealthCheck).Methods("GET")

	// User routes
	userRoutes := r.PathPrefix("/users").Subrouter()
	userRoutes.HandleFunc("", userHandler.CreateUser).Methods("POST")
	userRoutes.HandleFunc("/{id:[0-9]+}", userHandler.GetUser).Methods("GET")
	userRoutes.HandleFunc("/{id:[0-9]+}", userHandler.UpdateUser).Methods("PUT")
	userRoutes.HandleFunc("/{id:[0-9]+}", userHandler.DeleteUser).Methods("DELETE")

	// Apply auth middleware to protected routes
	userRoutes.Use(middleware.AuthMiddleware)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		router: r,
		server: srv,
		logger: logger,
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	s.logger.Info().Msgf("Server listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
