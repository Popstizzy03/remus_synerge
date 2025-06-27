// internal/api/handlers/handlers.go
package handlers

import "net/http"

// HealthCheck returns a 200 OK status.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
