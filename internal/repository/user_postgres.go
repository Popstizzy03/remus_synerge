package repository

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"remus_synerge/internal/models"
)

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	query := `INSERT INTO users (username, email, password, created_at, updated_at)
			   VALUES ($1, $2, $3, $4, $5) RETURNING id`
	
	var id int
	err := r.db.QueryRow(ctx, query, user.Username, user.Email, user.Password, user.CreatedAt, user.UpdatedAt).Scan(&id)
	if err != nil {
		return nil, err
	}
	
	user.ID = id
	return user, nil
}

func (r *userRepo) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE id = $1`
	user := &models.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = $1`
	user := &models.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	query := `UPDATE users SET username = $1, email = $2, password = $3, updated_at = $4 WHERE id = $5`
	
	_, err := r.db.Exec(ctx, query, user.Username, user.Email, user.Password, user.UpdatedAt, user.ID)
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

func (r *userRepo) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}