package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.InspectionRepository = (*SQLiteInspectionRepository)(nil)

type SQLiteInspectionRepository struct {
	db *sql.DB
}

func NewSQLiteInspectionRepository(db *sql.DB) *SQLiteInspectionRepository {
	return &SQLiteInspectionRepository{db: db}
}

func (r *SQLiteInspectionRepository) Create(ctx context.Context, ins *domain.Inspection) error {
	query := `
		INSERT INTO inspections (
			prediction_id, station_code, platform, door_number,
			status, priority, assigned_to, notes, scheduled_for, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	ins.CreatedAt = now
	ins.UpdatedAt = now
	if ins.Status == "" {
		ins.Status = "pending"
	}
	if ins.Priority == "" {
		ins.Priority = "high"
	}

	res, err := r.db.ExecContext(
		ctx,
		query,
		ins.PredictionID,
		ins.StationCode,
		ins.Platform,
		ins.DoorNumber,
		ins.Status,
		ins.Priority,
		ins.AssignedTo,
		ins.Notes,
		ins.ScheduledFor,
		now.Format("2006-01-02 15:04:05"),
		now.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	ins.ID = id
	return nil
}

func (r *SQLiteInspectionRepository) UpdateStatus(ctx context.Context, id int64, status, notes string, completedAt *time.Time) error {
	query := `
		UPDATE inspections
		SET status = ?, notes = CASE WHEN ? != '' THEN ? ELSE notes END,
		    completed_at = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now()
	var completedAtStr *string
	if completedAt != nil {
		s := completedAt.Format("2006-01-02 15:04:05")
		completedAtStr = &s
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		status,
		notes,
		notes,
		completedAtStr,
		now.Format("2006-01-02 15:04:05"),
		id,
	)
	return err
}

func (r *SQLiteInspectionRepository) GetByID(ctx context.Context, id int64) (*domain.Inspection, error) {
	query := `
		SELECT id, prediction_id, station_code, platform, door_number,
		       status, priority, assigned_to, IFNULL(notes, ''), IFNULL(scheduled_for, ''),
		       completed_at, created_at, updated_at
		FROM inspections
		WHERE id = ?
		LIMIT 1
	`
	var ins domain.Inspection
	var completedAtStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ins.ID,
		&ins.PredictionID,
		&ins.StationCode,
		&ins.Platform,
		&ins.DoorNumber,
		&ins.Status,
		&ins.Priority,
		&ins.AssignedTo,
		&ins.Notes,
		&ins.ScheduledFor,
		&completedAtStr,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	ins.CreatedAt = parseTime(createdAtStr)
	ins.UpdatedAt = parseTime(updatedAtStr)
	if completedAtStr.Valid {
		t := parseTime(completedAtStr.String)
		if !t.IsZero() {
			ins.CompletedAt = &t
		}
	}

	return &ins, nil
}

func (r *SQLiteInspectionRepository) List(ctx context.Context, limit, offset int, statusFilter string) ([]*domain.Inspection, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, prediction_id, station_code, platform, door_number,
		       status, priority, assigned_to, IFNULL(notes, ''), IFNULL(scheduled_for, ''),
		       completed_at, created_at, updated_at
		FROM inspections
		WHERE (? = '' OR status = ?)
		ORDER BY id DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, statusFilter, statusFilter, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Inspection
	for rows.Next() {
		var ins domain.Inspection
		var completedAtStr sql.NullString
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&ins.ID,
			&ins.PredictionID,
			&ins.StationCode,
			&ins.Platform,
			&ins.DoorNumber,
			&ins.Status,
			&ins.Priority,
			&ins.AssignedTo,
			&ins.Notes,
			&ins.ScheduledFor,
			&completedAtStr,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		ins.CreatedAt = parseTime(createdAtStr)
		ins.UpdatedAt = parseTime(updatedAtStr)
		if completedAtStr.Valid {
			t := parseTime(completedAtStr.String)
			if !t.IsZero() {
				ins.CompletedAt = &t
			}
		}

		list = append(list, &ins)
	}

	return list, rows.Err()
}
