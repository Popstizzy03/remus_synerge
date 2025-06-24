// internal/api/server.go
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"

	"myapp/internal/api/handlers"
	"myapp/internal/api/middleware"
	"myapp/internal/config"
	"myapp/internal/repository"
)

type Server struct {
	router *mux.Router
	server *http.Server
	logger zerolog.Logger
}

func New(cfg *config.Config, db *sql.DB, logger zerolog.Logger) *Server {
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
	userRoutes.HandleFunc("", userHandler.GetUsers).Methods("GET")
	userRoutes.HandleFunc("/{id}", userHandler.GetUser).Methods("GET")
	userRoutes.HandleFunc("", userHandler.CreateUser).Methods("POST")
	userRoutes.HandleFunc("/{id}", userHandler.UpdateUser).Methods("PUT")
	userRoutes.HandleFunc("/{id}", userHandler.DeleteUser).Methods("DELETE")
	
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

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}