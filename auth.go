// internal/api/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

// AuthMiddleware protects routes by verifying JWT tokens.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // No "Bearer " prefix
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		// For now, we'll just check if the token is not empty
		// In a real application, you would validate the token here
		if tokenString == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// In a real application, you would parse and validate the token
		// and extract the user ID to add to the request context.
		// For example:
		// claims := &Claims{}
		// token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 	return []byte("your-secret-key"), nil
		// })
		// if err != nil || !token.Valid {
		// 	http.Error(w, "Invalid token", http.StatusUnauthorized)
		// 	return
		// }
		//
		// ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		// next.ServeHTTP(w, r.WithContext(ctx))

		next.ServeHTTP(w, r)
	})
}
