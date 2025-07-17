package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
	"remus_synerge/internal/models"
	"remus_synerge/internal/repository"
)

type UserHandler struct {
	userRepo repository.UserRepository
	logger   zerolog.Logger
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type UpdateUserRequest struct {
	Username string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Password string `json:"password,omitempty" validate:"omitempty,min=8"`
}

type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func NewUserHandler(userRepo repository.UserRepository, logger zerolog.Logger) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode request body")
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validateCreateUserRequest(req); err != nil {
		h.logger.Error().Err(err).Msg("Invalid request data")
		h.sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if user already exists
	existingUser, err := h.userRepo.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		h.logger.Error().Str("email", req.Email).Msg("User already exists")
		h.sendErrorResponse(w, http.StatusConflict, "User with this email already exists")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to hash password")
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdUser, err := h.userRepo.CreateUser(ctx, user)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create user")
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	response := UserResponse{
		ID:        createdUser.ID,
		Username:  createdUser.Username,
		Email:     createdUser.Email,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
	}

	h.sendJSONResponse(w, http.StatusCreated, response)
	h.logger.Info().Int("user_id", createdUser.ID).Msg("User created successfully")
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		h.sendErrorResponse(w, http.StatusBadRequest, "Missing user ID")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userRepo.GetUserByID(ctx, id)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", id).Msg("Failed to get user")
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

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		h.sendErrorResponse(w, http.StatusBadRequest, "Missing user ID")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to decode request body")
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing user
	existingUser, err := h.userRepo.GetUserByID(ctx, id)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", id).Msg("Failed to get user")
		h.sendErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	// Update fields if provided
	if req.Username != "" {
		existingUser.Username = req.Username
	}
	if req.Email != "" {
		existingUser.Email = req.Email
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to hash password")
			h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to process password")
			return
		}
		existingUser.Password = string(hashedPassword)
	}

	existingUser.UpdatedAt = time.Now()

	updatedUser, err := h.userRepo.UpdateUser(ctx, existingUser)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", id).Msg("Failed to update user")
		h.sendErrorResponse(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	response := UserResponse{
		ID:        updatedUser.ID,
		Username:  updatedUser.Username,
		Email:     updatedUser.Email,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	}

	h.sendJSONResponse(w, http.StatusOK, response)
	h.logger.Info().Int("user_id", id).Msg("User updated successfully")
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		h.sendErrorResponse(w, http.StatusBadRequest, "Missing user ID")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = h.userRepo.DeleteUser(ctx, id)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", id).Msg("Failed to delete user")
		h.sendErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
	h.logger.Info().Int("user_id", id).Msg("User deleted successfully")
}

func (h *UserHandler) validateCreateUserRequest(req CreateUserRequest) error {
	if req.Username == "" || len(req.Username) < 3 || len(req.Username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	if req.Email == "" || !isValidEmail(req.Email) {
		return fmt.Errorf("valid email is required")
	}
	if req.Password == "" || len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func (h *UserHandler) sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (h *UserHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}

func isValidEmail(email string) bool {
	// Basic email validation - in production, use a proper email validation library
	return len(email) > 0 && 
		   len(email) <= 254 && 
		   strings.Contains(email, "@") && 
		   strings.Contains(email, ".")
}