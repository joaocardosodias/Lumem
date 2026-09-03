package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.UserRepository = (*PostgresUserRepository)(nil)

type PostgresUserRepository struct {
	db DBExecutor
}

func NewPostgresUserRepository(db DBExecutor) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, email)

	var u domain.User
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.IsActive,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt

	return &u, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.User
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&u.Role,
		&u.IsActive,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt

	return &u, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}
