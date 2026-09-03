package repository

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.InspectionRepository = (*SQLiteInspectionRepository)(nil)

type SQLiteInspectionRepository struct {
	db DBExecutor
}

func NewSQLiteInspectionRepository(db DBExecutor) *SQLiteInspectionRepository {
	return &SQLiteInspectionRepository{db: db}
}

func (r *SQLiteInspectionRepository) Create(ctx context.Context, ins *domain.Inspection) error {
	query := `
		INSERT INTO inspections (station_code, platform, door_number, scheduled_for, priority, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query,
		ins.StationCode,
		ins.Platform,
		ins.DoorNumber,
		ins.ScheduledFor,
		ins.Priority,
		ins.Status,
		ins.Notes,
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
	var compStr *string
	if completedAt != nil {
		s := completedAt.Format("2006-01-02 15:04:05")
		compStr = &s
	}

	query := `
		UPDATE inspections
		SET status = ?, notes = ?, completed_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, status, notes, compStr, id)
	return err
}

func (r *SQLiteInspectionRepository) GetByID(ctx context.Context, id int64) (*domain.Inspection, error) {
	query := `
		SELECT id, station_code, platform, door_number, scheduled_for, priority, status, notes, completed_at, created_at, updated_at
		FROM inspections
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var ins domain.Inspection
	var schedStr, notesStr, compStr, createStr, updateStr *string

	err := row.Scan(
		&ins.ID,
		&ins.StationCode,
		&ins.Platform,
		&ins.DoorNumber,
		&schedStr,
		&ins.Priority,
		&ins.Status,
		&notesStr,
		&compStr,
		&createStr,
		&updateStr,
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
	if compStr != nil {
		t := parseTime(*compStr)
		ins.CompletedAt = &t
	}
	if createStr != nil {
		ins.CreatedAt = parseTime(*createStr)
	}
	if updateStr != nil {
		ins.UpdatedAt = parseTime(*updateStr)
	}

	return &ins, nil
}

func (r *SQLiteInspectionRepository) List(ctx context.Context, limit, offset int, statusFilter string) ([]*domain.Inspection, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, station_code, platform, door_number, scheduled_for, priority, status, notes, completed_at, created_at, updated_at
		FROM inspections
	`
	var args []any

	if statusFilter != "" {
		query += ` WHERE status = ?`
		args = append(args, statusFilter)
	}

	query += ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Inspection
	for rows.Next() {
		var ins domain.Inspection
		var schedStr, notesStr, compStr, createStr, updateStr *string

		err := rows.Scan(
			&ins.ID,
			&ins.StationCode,
			&ins.Platform,
			&ins.DoorNumber,
			&schedStr,
			&ins.Priority,
			&ins.Status,
			&notesStr,
			&compStr,
			&createStr,
			&updateStr,
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
		if compStr != nil {
			t := parseTime(*compStr)
			ins.CompletedAt = &t
		}
		if createStr != nil {
			ins.CreatedAt = parseTime(*createStr)
		}
		if updateStr != nil {
			ins.UpdatedAt = parseTime(*updateStr)
		}

		list = append(list, &ins)
	}

	return list, nil
}
