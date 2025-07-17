package middleware

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Metrics struct {
	mu                   sync.RWMutex
	RequestCount         map[string]int64
	RequestDuration      map[string][]time.Duration
	StatusCodeCount      map[int]int64
	ActiveConnections    int64
	TotalConnections     int64
	ErrorCount           int64
	AverageResponseTime  time.Duration
	StartTime            time.Time
	logger               zerolog.Logger
}

func NewMetrics(logger zerolog.Logger) *Metrics {
	return &Metrics{
		RequestCount:     make(map[string]int64),
		RequestDuration:  make(map[string][]time.Duration),
		StatusCodeCount:  make(map[int]int64),
		StartTime:        time.Now(),
		logger:           logger,
	}
}

func (m *Metrics) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Record request count by method and path
	key := method + " " + path
	m.RequestCount[key]++
	
	// Record request duration
	m.RequestDuration[key] = append(m.RequestDuration[key], duration)
	
	// Keep only last 100 durations for each endpoint
	if len(m.RequestDuration[key]) > 100 {
		m.RequestDuration[key] = m.RequestDuration[key][len(m.RequestDuration[key])-100:]
	}
	
	// Record status code count
	m.StatusCodeCount[statusCode]++
	
	// Record errors (4xx and 5xx)
	if statusCode >= 400 {
		m.ErrorCount++
	}
	
	// Update total connections
	m.TotalConnections++
	
	// Calculate average response time
	var totalDuration time.Duration
	var totalRequests int64
	for _, durations := range m.RequestDuration {
		for _, d := range durations {
			totalDuration += d
			totalRequests++
		}
	}
	if totalRequests > 0 {
		m.AverageResponseTime = totalDuration / time.Duration(totalRequests)
	}
}

func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	uptime := time.Since(m.StartTime)
	
	// Calculate endpoint stats
	endpointStats := make(map[string]interface{})
	for endpoint, durations := range m.RequestDuration {
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := total / time.Duration(len(durations))
			
			endpointStats[endpoint] = map[string]interface{}{
				"count":           m.RequestCount[endpoint],
				"average_duration": avg.String(),
				"last_duration":   durations[len(durations)-1].String(),
			}
		}
	}
	
	return map[string]interface{}{
		"uptime":                uptime.String(),
		"total_requests":        m.TotalConnections,
		"active_connections":    m.ActiveConnections,
		"error_count":          m.ErrorCount,
		"average_response_time": m.AverageResponseTime.String(),
		"status_codes":         m.StatusCodeCount,
		"endpoints":            endpointStats,
		"timestamp":            time.Now().Unix(),
	}
}

func (m *Metrics) LogMetrics() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	m.logger.Info().
		Int64("total_requests", m.TotalConnections).
		Int64("active_connections", m.ActiveConnections).
		Int64("error_count", m.ErrorCount).
		Str("average_response_time", m.AverageResponseTime.String()).
		Str("uptime", time.Since(m.StartTime).String()).
		Msg("Server metrics")
}

func MetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Increment active connections
			metrics.mu.Lock()
			metrics.ActiveConnections++
			metrics.mu.Unlock()
			
			// Create custom response writer to capture status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
			}
			
			// Process request
			next.ServeHTTP(rw, r)
			
			// Calculate duration
			duration := time.Since(start)
			
			// Record metrics
			metrics.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration)
			
			// Decrement active connections
			metrics.mu.Lock()
			metrics.ActiveConnections--
			metrics.mu.Unlock()
		})
	}
}

// Health check handler that includes metrics
func HealthCheckHandler(metrics *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// Simple health check response
		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"uptime":    time.Since(metrics.StartTime).String(),
		}
		
		// Include basic metrics
		if r.URL.Query().Get("metrics") == "true" {
			response["metrics"] = metrics.GetMetrics()
		}
		
		// Write response
		if err := json.NewEncoder(w).Encode(response); err != nil {
			metrics.logger.Error().Err(err).Msg("Failed to encode health check response")
		}
	}
}

// Metrics endpoint handler
func MetricsHandler(metrics *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		metricsData := metrics.GetMetrics()
		
		if err := json.NewEncoder(w).Encode(metricsData); err != nil {
			metrics.logger.Error().Err(err).Msg("Failed to encode metrics response")
		}
	}
}