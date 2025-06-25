// internal/api/middleware/logging.go
package middleware

import (
		"net/http"
		"time"

		"github.com/rs/zerlog"
)

func LoggingMiddleware(logger zerlog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a custom response writer to capture the status code
		})
	}
}