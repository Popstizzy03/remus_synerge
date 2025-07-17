package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type RateLimiter struct {
	mu          sync.RWMutex
	requests    map[string][]time.Time
	maxRequests int
	window      time.Duration
	logger      zerolog.Logger
}

func NewRateLimiter(maxRequests int, window time.Duration, logger zerolog.Logger) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
		logger:      logger,
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.requests {
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = validTimes
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	times, exists := rl.requests[ip]
	
	if !exists {
		rl.requests[ip] = []time.Time{now}
		return true
	}
	
	// Remove old entries
	var validTimes []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			validTimes = append(validTimes, t)
		}
	}
	
	if len(validTimes) >= rl.maxRequests {
		return false
	}
	
	rl.requests[ip] = append(validTimes, now)
	return true
}

func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			
			if !limiter.Allow(ip) {
				limiter.logger.Warn().
					Str("ip", ip).
					Str("path", r.URL.Path).
					Msg("Rate limit exceeded")
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Rate limit exceeded","message":"Too many requests"}`))
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeadersMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			
			// Remove server info
			w.Header().Set("Server", "")
			
			next.ServeHTTP(w, r)
		})
	}
}

func CORSMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow specific origins in production
			origin := r.Header.Get("Origin")
			if origin != "" {
				// In production, replace with your actual domains
				allowedOrigins := []string{
					"http://localhost:3000",
					"https://yourdomain.com",
				}
				
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}
			
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

func RecoveryMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error().
						Interface("error", err).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("ip", getClientIP(r)).
						Msg("Panic recovered")
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"Internal server error","message":"An unexpected error occurred"}`))
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

func RequestValidationMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate request size
			if r.ContentLength > 1024*1024 { // 1MB limit
				logger.Warn().
					Int64("content_length", r.ContentLength).
					Str("ip", getClientIP(r)).
					Msg("Request too large")
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte(`{"error":"Request too large","message":"Request body exceeds maximum size"}`))
				return
			}
			
			// Validate content type for POST/PUT requests
			if r.Method == "POST" || r.Method == "PUT" {
				contentType := r.Header.Get("Content-Type")
				if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
					logger.Warn().
						Str("content_type", contentType).
						Str("ip", getClientIP(r)).
						Msg("Invalid content type")
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnsupportedMediaType)
					w.Write([]byte(`{"error":"Unsupported media type","message":"Content-Type must be application/json"}`))
					return
				}
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

func TimeoutMiddleware(timeout time.Duration, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			r = r.WithContext(ctx)
			
			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(w, r)
			}()
			
			select {
			case <-done:
				// Request completed successfully
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					logger.Warn().
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("ip", getClientIP(r)).
						Dur("timeout", timeout).
						Msg("Request timeout")
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusRequestTimeout)
					w.Write([]byte(`{"error":"Request timeout","message":"Request took too long to process"}`))
				}
			}
		})
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	
	return ip
}