// internal/api/middleware/logging.go
package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// responseWriter is a custom response writer to capture the status code.

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// LoggingMiddleware logs incoming requests.
func LoggingMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				logger.Info().
					Str("method", r.Method).
					Str("url", r.URL.String()).
					Int("status", rw.status).
					Dur("duration", time.Since(start)).
					Msg("Incoming request")
			}()

			next.ServeHTTP(rw, r)
		})
	}
}
