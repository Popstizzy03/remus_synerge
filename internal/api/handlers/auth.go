package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"remus_synerge/internal/api/middleware"
	"remus_synerge/internal/repository"
)

type AuthHandler struct {
	userRepo    repository.UserRepository
	authService *middleware.AuthService
	logger      zerolog.Logger
}

func NewAuthHandler(userRepo repository.UserRepository, authService *middleware.AuthService, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		authService: authService,
		logger:      logger,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req middleware.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode login request")
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		h.logger.Error().Msg("Empty email or password")
		h.sendErrorResponse(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Get user by email
	user, err := h.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("User not found")
		h.sendErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Compare password
	if err := h.authService.ComparePassword(user.Password, req.Password); err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("Invalid password")
		h.sendErrorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.authService.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate token")
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := middleware.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: middleware.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
	}

	h.sendJSONResponse(w, http.StatusOK, response)
	h.logger.Info().
		Int("user_id", user.ID).
		Str("username", user.Username).
		Str("email", user.Email).
		Msg("User logged in successfully")
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get user info from context (set by auth middleware)
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.sendErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	username, _ := middleware.GetUsernameFromContext(r.Context())
	email, _ := middleware.GetEmailFromContext(r.Context())

	// Generate new token
	token, expiresAt, err := h.authService.GenerateToken(userID, username, email)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate refresh token")
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := middleware.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: middleware.UserInfo{
			ID:       userID,
			Username: username,
			Email:    email,
		},
	}

	h.sendJSONResponse(w, http.StatusOK, response)
	h.logger.Info().
		Int("user_id", userID).
		Str("username", username).
		Msg("Token refreshed successfully")
}

func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user info from context (set by auth middleware)
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.sendErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	user, err := h.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", userID).Msg("Failed to get user profile")
		h.sendErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	response := UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	h.sendJSONResponse(w, http.StatusOK, response)
}

func (h *AuthHandler) sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}