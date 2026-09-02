package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.UserRepository = (*SQLiteUserRepository)(nil)

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE email = ?
		LIMIT 1
	`

	var user domain.User
	var createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
		user.CreatedAt = t
	} else if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		user.CreatedAt = t
	}

	if t, err := time.Parse("2006-01-02 15:04:05", updatedAtStr); err == nil {
		user.UpdatedAt = t
	} else if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		user.UpdatedAt = t
	}

	return &user, nil
}

func (r *SQLiteUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE id = ?
		LIMIT 1
	`

	var user domain.User
	var createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
		user.CreatedAt = t
	} else if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		user.CreatedAt = t
	}

	if t, err := time.Parse("2006-01-02 15:04:05", updatedAtStr); err == nil {
		user.UpdatedAt = t
	} else if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		user.UpdatedAt = t
	}

	return &user, nil
}

func (r *SQLiteUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.IsActive,
		user.CreatedAt.Format("2006-01-02 15:04:05"),
		user.UpdatedAt.Format("2006-01-02 15:04:05"),
	)

	return err
}
