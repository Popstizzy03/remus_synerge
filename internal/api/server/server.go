package server

import (
	"context"
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

type Server struct {
	router      *mux.Router
	server      *http.Server
	logger      zerolog.Logger
	metrics     *middleware.Metrics
	authService *middleware.AuthService
}

func New(cfg *config.Config, db *pgxpool.Pool, logger zerolog.Logger) *Server {
	// Initialize metrics
	metrics := middleware.NewMetrics(logger)
	
	// Initialize authentication service
	authService := middleware.NewAuthService(logger)
	
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	
	// Initialize handlers
	userHandler := handlers.NewUserHandler(userRepo, logger)
	authHandler := handlers.NewAuthHandler(userRepo, authService, logger)
	
	// Create router
	r := mux.NewRouter()
	
	// Initialize rate limiter (100 requests per minute)
	rateLimiter := middleware.NewRateLimiter(100, time.Minute, logger)
	
	// Global middleware (applied to all routes)
	r.Use(middleware.RecoveryMiddleware(logger))
	r.Use(middleware.SecurityHeadersMiddleware(logger))
	r.Use(middleware.CORSMiddleware(logger))
	r.Use(middleware.LoggingMiddleware(logger))
	r.Use(middleware.MetricsMiddleware(metrics))
	r.Use(middleware.RateLimitMiddleware(rateLimiter))
	r.Use(middleware.RequestValidationMiddleware(logger))
	r.Use(middleware.TimeoutMiddleware(30*time.Second, logger))
	
	// Public routes (no authentication required)
	publicRouter := r.PathPrefix("/api/v1").Subrouter()
	publicRouter.HandleFunc("/health", middleware.HealthCheckHandler(metrics)).Methods("GET")
	publicRouter.HandleFunc("/metrics", middleware.MetricsHandler(metrics)).Methods("GET")
	publicRouter.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	publicRouter.HandleFunc("/users", userHandler.CreateUser).Methods("POST") // User registration
	
	// Protected routes (authentication required)
	protectedRouter := r.PathPrefix("/api/v1").Subrouter()
	protectedRouter.Use(middleware.AuthMiddleware(authService))
	
	// Auth routes
	protectedRouter.HandleFunc("/auth/refresh", authHandler.RefreshToken).Methods("POST")
	protectedRouter.HandleFunc("/auth/profile", authHandler.GetProfile).Methods("GET")
	
	// User routes
	protectedRouter.HandleFunc("/users/{id:[0-9]+}", userHandler.GetUser).Methods("GET")
	protectedRouter.HandleFunc("/users/{id:[0-9]+}", userHandler.UpdateUser).Methods("PUT")
	protectedRouter.HandleFunc("/users/{id:[0-9]+}", userHandler.DeleteUser).Methods("DELETE")
	
	// Static file serving
	staticDir := "/static/"
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("./static/"))))
	
	// Create HTTP server with enhanced configuration
	addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
	
	// Start metrics logging goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			metrics.LogMetrics()
		}
	}()
	
	return &Server{
		router:      r,
		server:      srv,
		logger:      logger,
		metrics:     metrics,
		authService: authService,
	}
}

func (s *Server) Start() error {
	s.logger.Info().Msgf("Server starting on %s", s.server.Addr)
	s.logger.Info().Msg("Available endpoints:")
	s.logger.Info().Msg("  Public:")
	s.logger.Info().Msg("    GET  /api/v1/health")
	s.logger.Info().Msg("    GET  /api/v1/metrics")
	s.logger.Info().Msg("    POST /api/v1/auth/login")
	s.logger.Info().Msg("    POST /api/v1/users (registration)")
	s.logger.Info().Msg("  Protected:")
	s.logger.Info().Msg("    POST /api/v1/auth/refresh")
	s.logger.Info().Msg("    GET  /api/v1/auth/profile")
	s.logger.Info().Msg("    GET  /api/v1/users/{id}")
	s.logger.Info().Msg("    PUT  /api/v1/users/{id}")
	s.logger.Info().Msg("    DELETE /api/v1/users/{id}")
	
	// Try to enable HTTPS if TLS cert and key are available
	if tlsCert := s.server.TLSConfig; tlsCert != nil {
		s.logger.Info().Msg("Starting HTTPS server")
		return s.server.ListenAndServeTLS("", "")
	}
	
	s.logger.Info().Msg("Starting HTTP server")
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down server...")
	s.metrics.LogMetrics() // Log final metrics
	return s.server.Shutdown(ctx)
}

func (s *Server) GetMetrics() *middleware.Metrics {
	return s.metrics
}