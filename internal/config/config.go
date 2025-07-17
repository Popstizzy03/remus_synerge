// internal/config/config.go
package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Address           string
	Port              int
	ReadTimeout       int
	WriteTimeout      int
	IdleTimeout       int
	MaxHeaderBytes    int
	TLSCertFile       string
	TLSKeyFile        string
	EnableHTTPS       bool
	EnableMetrics     bool
	StaticDir         string
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConnections  int
	MaxIdleTime     int
	MaxLifetime     int
}

type SecurityConfig struct {
	JWTSecret         string
	JWTExpiration     int
	RateLimitRequests int
	RateLimitWindow   int
	EnableRateLimit   bool
	EnableCORS        bool
	TrustedOrigins    []string
}

// Load Configuration from environment variables
func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	readTimeout, _ := strconv.Atoi(getEnv("READ_TIMEOUT", "15"))
	writeTimeout, _ := strconv.Atoi(getEnv("WRITE_TIMEOUT", "15"))
	idleTimeout, _ := strconv.Atoi(getEnv("IDLE_TIMEOUT", "60"))
	maxHeaderBytes, _ := strconv.Atoi(getEnv("MAX_HEADER_BYTES", "1048576"))
	
	maxConnections, _ := strconv.Atoi(getEnv("DB_MAX_CONNECTIONS", "25"))
	maxIdleTime, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_TIME", "300"))
	maxLifetime, _ := strconv.Atoi(getEnv("DB_MAX_LIFETIME", "1800"))
	
	jwtExpiration, _ := strconv.Atoi(getEnv("JWT_EXPIRATION", "86400"))
	rateLimitRequests, _ := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "100"))
	rateLimitWindow, _ := strconv.Atoi(getEnv("RATE_LIMIT_WINDOW", "60"))
	
	enableHTTPS := getEnv("ENABLE_HTTPS", "false") == "true"
	enableMetrics := getEnv("ENABLE_METRICS", "true") == "true"
	enableRateLimit := getEnv("ENABLE_RATE_LIMIT", "true") == "true"
	enableCORS := getEnv("ENABLE_CORS", "true") == "true"
	
	// Parse trusted origins
	trustedOrigins := parseTrustedOrigins(getEnv("TRUSTED_ORIGINS", "http://localhost:3000"))

	return &Config{
		Server: ServerConfig{
			Address:        getEnv("SERVER_ADDRESS", "0.0.0.0"),
			Port:           port,
			ReadTimeout:    readTimeout,
			WriteTimeout:   writeTimeout,
			IdleTimeout:    idleTimeout,
			MaxHeaderBytes: maxHeaderBytes,
			TLSCertFile:    getEnv("TLS_CERT_FILE", ""),
			TLSKeyFile:     getEnv("TLS_KEY_FILE", ""),
			EnableHTTPS:    enableHTTPS,
			EnableMetrics:  enableMetrics,
			StaticDir:      getEnv("STATIC_DIR", "./static"),
		},
		Database: DatabaseConfig{
			Host:           getEnv("DB_HOST", "localhost"),
			Port:           dbPort,
			User:           getEnv("DB_USER", "postgres"),
			Password:       getEnv("DB_PASSWORD", "postgres"),
			Name:           getEnv("DB_NAME", "remus_synerge"),
			SSLMode:        getEnv("DB_SSLMODE", "disable"),
			MaxConnections: maxConnections,
			MaxIdleTime:    maxIdleTime,
			MaxLifetime:    maxLifetime,
		},
		Security: SecurityConfig{
			JWTSecret:         getEnv("JWT_SECRET_KEY", ""),
			JWTExpiration:     jwtExpiration,
			RateLimitRequests: rateLimitRequests,
			RateLimitWindow:   rateLimitWindow,
			EnableRateLimit:   enableRateLimit,
			EnableCORS:        enableCORS,
			TrustedOrigins:    trustedOrigins,
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

// Helper function to parse trusted origins from comma-separated string
func parseTrustedOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}
	
	var result []string
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	
	return result
}