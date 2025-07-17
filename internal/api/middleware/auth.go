package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

type JWTClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.StandardClaims
}

type AuthService struct {
	secretKey []byte
	logger    zerolog.Logger
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func NewAuthService(logger zerolog.Logger) *AuthService {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		// Generate a random secret key for development
		// In production, this should be set as an environment variable
		logger.Warn().Msg("JWT_SECRET_KEY not set, generating random key")
		secretKey = generateRandomKey()
	}
	
	return &AuthService{
		secretKey: []byte(secretKey),
		logger:    logger,
	}
}

func generateRandomKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("Failed to generate random key: %v", err))
	}
	return hex.EncodeToString(bytes)
}

func (as *AuthService) GenerateToken(userID int, username, email string) (string, time.Time, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "remus_synerge",
			Subject:   fmt.Sprintf("user_%d", userID),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(as.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}
	
	return tokenString, expirationTime, nil
}

func (as *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return as.secretKey, nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token claims")
}

func (as *AuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

func (as *AuthService) ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func AuthMiddleware(authService *AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				authService.logger.Warn().
					Str("ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Msg("Missing authorization header")
				
				sendUnauthorizedResponse(w, "Missing authorization header")
				return
			}
			
			// Check if the header starts with "Bearer "
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				authService.logger.Warn().
					Str("ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Msg("Invalid authorization header format")
				
				sendUnauthorizedResponse(w, "Invalid authorization header format")
				return
			}
			
			// Extract the token
			tokenString := authHeader[len(bearerPrefix):]
			if tokenString == "" {
				sendUnauthorizedResponse(w, "Missing token")
				return
			}
			
			// Validate the token
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				authService.logger.Warn().
					Err(err).
					Str("ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Msg("Invalid token")
				
				sendUnauthorizedResponse(w, "Invalid token")
				return
			}
			
			// Add user info to request context
			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "username", claims.Username)
			ctx = context.WithValue(ctx, "email", claims.Email)
			
			authService.logger.Debug().
				Int("user_id", claims.UserID).
				Str("username", claims.Username).
				Str("path", r.URL.Path).
				Msg("User authenticated")
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuthMiddleware(authService *AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No auth header, continue without user context
				next.ServeHTTP(w, r)
				return
			}
			
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			
			tokenString := authHeader[len(bearerPrefix):]
			if tokenString == "" {
				next.ServeHTTP(w, r)
				return
			}
			
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				// Invalid token, continue without user context
				next.ServeHTTP(w, r)
				return
			}
			
			// Add user info to request context
			ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "username", claims.Username)
			ctx = context.WithValue(ctx, "email", claims.Email)
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func sendUnauthorizedResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "Unauthorized",
		"message": message,
	})
}

// Helper functions to extract user info from context
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value("user_id").(int)
	return userID, ok
}

func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}

func GetEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value("email").(string)
	return email, ok
}