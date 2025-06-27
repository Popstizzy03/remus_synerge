// internal/repository/users.go
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"remus_synerge/internal/models"
)

// UserRepository defines the interface for user data operations.

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) (int64, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id int64) error
}

// userRepo is the implementation of UserRepository.

type userRepo struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

// CreateUser adds a new user to the database.
func (r *userRepo) CreateUser(ctx context.Context, user *models.User) (int64, error) {
	query := `INSERT INTO users (username, email, password, created_at, updated_at)
			   VALUES ($1, $2, $3, $4, $5) RETURNING id`
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	var id int64
	err := r.db.QueryRow(ctx, query, user.Username, user.Email, user.Password, user.CreatedAt, user.UpdatedAt).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetUserByID retrieves a user by their ID.
func (r *userRepo) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE id = $1`
	user := &models.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by their email.
func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = $1`
	user := &models.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser updates a user's information.
func (r *userRepo) UpdateUser(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET username = $1, email = $2, updated_at = $3 WHERE id = $4`
	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, query, user.Username, user.Email, user.UpdatedAt, user.ID)
	return err
}

// DeleteUser removes a user from the database.
func (r *userRepo) DeleteUser(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
