package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.MaintenanceRepository = (*SQLiteMaintenanceRepository)(nil)

type SQLiteMaintenanceRepository struct {
	db *sql.DB
}

func NewSQLiteMaintenanceRepository(db *sql.DB) *SQLiteMaintenanceRepository {
	return &SQLiteMaintenanceRepository{db: db}
}

func (r *SQLiteMaintenanceRepository) Create(ctx context.Context, event *domain.MaintenanceEvent) error {
	query := `
		INSERT INTO maintenance_events (
			sap_order_number, order_type, station_code,
			installation_location, start_timestamp, end_timestamp,
			duration_hours, description, symptom, cause, action_taken, source, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	event.CreatedAt = now
	event.Source = "manual_form"

	res, err := r.db.ExecContext(
		ctx,
		query,
		event.SAPOrderNumber,
		event.OrderType,
		event.StationCode,
		event.InstallationLocation,
		event.StartTimestamp,
		event.EndTimestamp,
		event.DurationHours,
		event.Description,
		event.Symptom,
		event.Cause,
		event.ActionTaken,
		event.Source,
		now.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	event.ID = id
	return nil
}

func (r *SQLiteMaintenanceRepository) List(ctx context.Context, limit, offset int) ([]*domain.MaintenanceEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, IFNULL(upload_id, 0), IFNULL(sap_order_number, ''), order_type,
		       IFNULL(station_code, ''), IFNULL(installation_location, ''),
		       start_timestamp, IFNULL(end_timestamp, ''), duration_hours,
		       IFNULL(description, ''), IFNULL(symptom, ''), IFNULL(cause, ''),
		       IFNULL(action_taken, ''), source, created_at
		FROM maintenance_events
		ORDER BY id DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.MaintenanceEvent
	for rows.Next() {
		var e domain.MaintenanceEvent
		var createdAtStr string

		err := rows.Scan(
			&e.ID,
			&e.UploadID,
			&e.SAPOrderNumber,
			&e.OrderType,
			&e.StationCode,
			&e.InstallationLocation,
			&e.StartTimestamp,
			&e.EndTimestamp,
			&e.DurationHours,
			&e.Description,
			&e.Symptom,
			&e.Cause,
			&e.ActionTaken,
			&e.Source,
			&createdAtStr,
		)
		if err != nil {
			return nil, err
		}

		if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			e.CreatedAt = t
		}
		events = append(events, &e)
	}

	return events, rows.Err()
}

func (r *SQLiteMaintenanceRepository) BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO maintenance_events (
			upload_id, sap_order_number, order_type, station_code,
			installation_location, start_timestamp, end_timestamp,
			duration_hours, description, symptom, cause, action_taken, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		_, err := stmt.ExecContext(
			ctx,
			e.UploadID,
			e.SAPOrderNumber,
			e.OrderType,
			e.StationCode,
			e.InstallationLocation,
			e.StartTimestamp,
			e.EndTimestamp,
			e.DurationHours,
			e.Description,
			e.Symptom,
			e.Cause,
			e.ActionTaken,
			e.Source,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
