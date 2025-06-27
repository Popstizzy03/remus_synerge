// internal/config/config.go
package config

import (
	"os"
	"strconv"
)

type Config struct {
		Server 	 ServerConfig
		Database DatabaseConfig
}

type ServerConfig struct {
		Address string
		Port 	int
}

type DatabaseConfig struct {
		Host     string
		Port 	 int 
		User 	 string 
		Password string
		Name 	 string 
		SSLMode  string 
}

// Load Configuration from environment variables
func Load() (*Config, error) {
		port, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
		dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))

		return &Config{
				Server: ServerConfig{
						Address: getEnv("SERVER_ADDRESS", "0.0.0.0"),
						Port: port,
				},
				Database: DatabaseConfig{
						Host: 		getEnv("DB_HOST", "localhost"),
						Port: 		dbPort,
						User: 		getEnv("DB_USER", "postgres"),
						Password: 	getEnv("DB_PASSWORD", "postgres"),
						Name: 		getEnv("DB_NAME", "remus_synerge"),
						SSLMode: 	getEnv("DB_SSLMODE", "disable"),
				},
		}, nil 
}

// Helper function to get environment variables with default values
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
			return value
	}
	return defaultValue
}