// pkg/database/postgres.go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"remus_synerge/internal/config"
)

// NewPostgresClient creates a new PostgreSQL connection pool.
func NewPostgresClient(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Ping the database to ensure the connection is established.
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}
