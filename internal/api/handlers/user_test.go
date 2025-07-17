package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"remus_synerge/internal/models"
)

// Mock repository for testing
type mockUserRepository struct {
	users map[int]*models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[int]*models.User),
	}
}

func (m *mockUserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if user.Email == "existing@example.com" {
		return nil, errors.New("user already exists")
	}
	
	user.ID = len(m.users) + 1
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if _, exists := m.users[user.ID]; !exists {
		return nil, errors.New("user not found")
	}
	m.users[user.ID] = user
	return user, nil
}

func (m *mockUserRepository) DeleteUser(ctx context.Context, id int) error {
	if _, exists := m.users[id]; !exists {
		return errors.New("user not found")
	}
	delete(m.users, id)
	return nil
}

func TestUserHandler_CreateUser(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	mockRepo := newMockUserRepository()
	handler := NewUserHandler(mockRepo, logger)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "valid user creation",
			requestBody: CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "invalid email",
			requestBody: CreateUserRequest{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "short password",
			requestBody: CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "short",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "missing username",
			requestBody: CreateUserRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			rr := httptest.NewRecorder()
			handler.CreateUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError {
				var errorResp ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("expected error response, got: %s", rr.Body.String())
				}
			} else {
				var userResp UserResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &userResp); err != nil {
					t.Errorf("expected user response, got: %s", rr.Body.String())
				}
			}
		})
	}
}

func TestUserHandler_GetUser(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	mockRepo := newMockUserRepository()
	handler := NewUserHandler(mockRepo, logger)

	// Create a test user
	testUser := &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockRepo.users[1] = testUser

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid user ID",
			userID:         "1",
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "invalid user ID",
			userID:         "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "non-numeric user ID",
			userID:         "abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.userID})
			
			rr := httptest.NewRecorder()
			handler.GetUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError {
				var errorResp ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("expected error response, got: %s", rr.Body.String())
				}
			} else {
				var userResp UserResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &userResp); err != nil {
					t.Errorf("expected user response, got: %s", rr.Body.String())
				}
				
				if userResp.ID != testUser.ID {
					t.Errorf("expected user ID %d, got %d", testUser.ID, userResp.ID)
				}
			}
		})
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	mockRepo := newMockUserRepository()
	handler := NewUserHandler(mockRepo, logger)

	// Create a test user
	testUser := &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockRepo.users[1] = testUser

	tests := []struct {
		name           string
		userID         string
		requestBody    interface{}
		expectedStatus int
		expectedError  bool
	}{
		{
			name:   "valid user update",
			userID: "1",
			requestBody: UpdateUserRequest{
				Username: "updateduser",
				Email:    "updated@example.com",
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:   "user not found",
			userID: "999",
			requestBody: UpdateUserRequest{
				Username: "updateduser",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "invalid user ID",
			userID:         "abc",
			requestBody:    UpdateUserRequest{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = mux.SetURLVars(req, map[string]string{"id": tt.userID})
			
			rr := httptest.NewRecorder()
			handler.UpdateUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError {
				var errorResp ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("expected error response, got: %s", rr.Body.String())
				}
			} else {
				var userResp UserResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &userResp); err != nil {
					t.Errorf("expected user response, got: %s", rr.Body.String())
				}
			}
		})
	}
}

func TestUserHandler_DeleteUser(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	mockRepo := newMockUserRepository()
	handler := NewUserHandler(mockRepo, logger)

	// Create a test user
	testUser := &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockRepo.users[1] = testUser

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid user deletion",
			userID:         "1",
			expectedStatus: http.StatusNoContent,
			expectedError:  false,
		},
		{
			name:           "user not found",
			userID:         "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "invalid user ID",
			userID:         "abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.userID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.userID})
			
			rr := httptest.NewRecorder()
			handler.DeleteUser(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedError {
				var errorResp ErrorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
					t.Errorf("expected error response, got: %s", rr.Body.String())
				}
			}
		})
	}
}