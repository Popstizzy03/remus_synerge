package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func LoggingMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Create custom response writer to capture status code and size
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     200, // Default status code
			}
			
			// Get client IP
			clientIP := getClientIP(r)
			
			// Log request start
			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Str("ip", clientIP).
				Str("user_agent", r.UserAgent()).
				Str("referer", r.Referer()).
				Int64("content_length", r.ContentLength).
				Msg("Request started")
			
			// Process request
			next.ServeHTTP(rw, r)
			
			// Calculate duration
			duration := time.Since(start)
			
			// Log request completion
			logEvent := logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Str("ip", clientIP).
				Int("status", rw.statusCode).
				Int("size", rw.size).
				Dur("duration", duration).
				Str("user_agent", r.UserAgent())
			
			// Add user info if available from context
			if userID, ok := GetUserIDFromContext(r.Context()); ok {
				logEvent = logEvent.Int("user_id", userID)
			}
			
			if username, ok := GetUsernameFromContext(r.Context()); ok {
				logEvent = logEvent.Str("username", username)
			}
			
			// Log with appropriate level based on status code
			switch {
			case rw.statusCode >= 500:
				logEvent.Msg("Request completed with server error")
			case rw.statusCode >= 400:
				logEvent.Msg("Request completed with client error")
			case rw.statusCode >= 300:
				logEvent.Msg("Request completed with redirect")
			default:
				logEvent.Msg("Request completed successfully")
			}
		})
	}
}