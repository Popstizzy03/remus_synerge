// pkg/database/postgres.go
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"remus_synerge/internal/config"
)

// NewPostgresClient creates a new PostgreSQL connection pool.
func NewPostgresClient(cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool
	config.MaxConns = int32(cfg.MaxConnections)
	config.MaxConnIdleTime = time.Duration(cfg.MaxIdleTime) * time.Second
	config.MaxConnLifetime = time.Duration(cfg.MaxLifetime) * time.Second
	config.HealthCheckPeriod = 30 * time.Second
	
	// Set connection timeouts
	config.ConnConfig.ConnectTimeout = 10 * time.Second
	config.ConnConfig.RuntimeParams["application_name"] = "remus_synerge"

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Ping the database to ensure the connection is established.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}
