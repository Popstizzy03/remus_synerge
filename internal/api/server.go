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

type server struct {
	router *mux.router
	server *http.Server
	logger zerolog.logger
}

func New(cfg *config.Config, db *sql.DB, logger zerolog.Logger) *Server {
	r := mux.NewRouter()

	// Create repositories
	userRepo := repository.NewUserRepository(db)

	// Create handlers
	userHandler := handlers.NewUserHandler(userRepo, logger)

	// Register middleware
}