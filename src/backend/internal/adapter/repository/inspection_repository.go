package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

var _ port.InspectionRepository = (*PostgresInspectionRepository)(nil)

type PostgresInspectionRepository struct {
	db DBExecutor
}

func NewPostgresInspectionRepository(db DBExecutor) *PostgresInspectionRepository {
	return &PostgresInspectionRepository{db: db}
}

func (r *PostgresInspectionRepository) Create(ctx context.Context, ins *domain.Inspection) error {
	if ins.ID == "" {
		ins.ID = uuid.New().String()
	}

	query := `
		INSERT INTO inspections (id, station_code, platform, door_number, scheduled_for, priority, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		ins.ID,
		ins.StationCode,
		ins.Platform,
		ins.DoorNumber,
		ins.ScheduledFor,
		ins.Priority,
		ins.Status,
		ins.Notes,
	)
	return err
}

func (r *PostgresInspectionRepository) UpdateStatus(ctx context.Context, id string, status, notes string, completedAt *time.Time) error {
	query := `
		UPDATE inspections
		SET status = $1, notes = $2, completed_at = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`
	res, err := r.db.ExecContext(ctx, query, status, notes, completedAt, id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return domain.ErrInspectionNotFound
	}
	return nil
}

func (r *PostgresInspectionRepository) GetByID(ctx context.Context, id string) (*domain.Inspection, error) {
	query := `
		SELECT id, station_code, platform, door_number, scheduled_for, priority, status, notes, completed_at, created_at, updated_at
		FROM inspections
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var ins domain.Inspection
	var schedStr, notesStr *string
	var completedAt *time.Time
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&ins.ID,
		&ins.StationCode,
		&ins.Platform,
		&ins.DoorNumber,
		&schedStr,
		&ins.Priority,
		&ins.Status,
		&notesStr,
		&completedAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInspectionNotFound
		}
		return nil, err
	}

	if schedStr != nil {
		ins.ScheduledFor = *schedStr
	}
	if notesStr != nil {
		ins.Notes = *notesStr
	}
	ins.CompletedAt = completedAt
	ins.CreatedAt = createdAt
	ins.UpdatedAt = updatedAt

	return &ins, nil
}

func (r *PostgresInspectionRepository) List(ctx context.Context, limit, offset int, statusFilter string) ([]*domain.Inspection, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var query string
	var args []any

	if statusFilter != "" {
		query = `
			SELECT id, station_code, platform, door_number, scheduled_for, priority, status, notes, completed_at, created_at, updated_at
			FROM inspections
			WHERE status = $1
			ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`
		args = []any{statusFilter, limit, offset}
	} else {
		query = `
			SELECT id, station_code, platform, door_number, scheduled_for, priority, status, notes, completed_at, created_at, updated_at
			FROM inspections
			ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`
		args = []any{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Inspection
	for rows.Next() {
		var ins domain.Inspection
		var schedStr, notesStr *string
		var completedAt *time.Time
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&ins.ID,
			&ins.StationCode,
			&ins.Platform,
			&ins.DoorNumber,
			&schedStr,
			&ins.Priority,
			&ins.Status,
			&notesStr,
			&completedAt,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if schedStr != nil {
			ins.ScheduledFor = *schedStr
		}
		if notesStr != nil {
			ins.Notes = *notesStr
		}
		ins.CompletedAt = completedAt
		ins.CreatedAt = createdAt
		ins.UpdatedAt = updatedAt

		list = append(list, &ins)
	}

	return list, nil
}
