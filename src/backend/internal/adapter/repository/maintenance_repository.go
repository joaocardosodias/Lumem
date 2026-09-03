package repository

import (
	"context"

	"github.com/g04/backend/internal/domain"
)

var _ domain.MaintenanceRepository = (*SQLiteMaintenanceRepository)(nil)

type SQLiteMaintenanceRepository struct {
	db DBExecutor
}

func NewSQLiteMaintenanceRepository(db DBExecutor) *SQLiteMaintenanceRepository {
	return &SQLiteMaintenanceRepository{db: db}
}

func (r *SQLiteMaintenanceRepository) BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error {
	if len(events) == 0 {
		return nil
	}

	query := `
		INSERT INTO maintenance_events (
			upload_id, sap_order_number, order_type, installation_location,
			station_code, start_timestamp, end_timestamp, duration_hours,
			description, symptom, cause, action_taken, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		_, err := stmt.ExecContext(ctx,
			e.UploadID,
			e.SAPOrderNumber,
			e.OrderType,
			e.InstallationLocation,
			e.StationCode,
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

	return nil
}

func (r *SQLiteMaintenanceRepository) Create(ctx context.Context, event *domain.MaintenanceEvent) error {
	query := `
		INSERT INTO maintenance_events (
			sap_order_number, order_type, installation_location,
			station_code, start_timestamp, end_timestamp, duration_hours,
			description, symptom, cause, action_taken, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query,
		event.SAPOrderNumber,
		event.OrderType,
		event.InstallationLocation,
		event.StationCode,
		event.StartTimestamp,
		event.EndTimestamp,
		event.DurationHours,
		event.Description,
		event.Symptom,
		event.Cause,
		event.ActionTaken,
		event.Source,
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
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, upload_id, sap_order_number, order_type, installation_location,
		       station_code, start_timestamp, end_timestamp, duration_hours,
		       description, symptom, cause, action_taken, source, created_at
		FROM maintenance_events
		ORDER BY id DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.MaintenanceEvent
	for rows.Next() {
		var e domain.MaintenanceEvent
		var uploadID *int64
		var createdAtStr string

		err := rows.Scan(
			&e.ID,
			&uploadID,
			&e.SAPOrderNumber,
			&e.OrderType,
			&e.InstallationLocation,
			&e.StationCode,
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

		if uploadID != nil {
			e.UploadID = *uploadID
		}
		e.CreatedAt = parseTime(createdAtStr)
		list = append(list, &e)
	}

	return list, nil
}
