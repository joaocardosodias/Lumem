package repository

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
)

var _ domain.MaintenanceRepository = (*PostgresMaintenanceRepository)(nil)

type PostgresMaintenanceRepository struct {
	db DBExecutor
}

func NewPostgresMaintenanceRepository(db DBExecutor) *PostgresMaintenanceRepository {
	return &PostgresMaintenanceRepository{db: db}
}

func (r *PostgresMaintenanceRepository) BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error {
	if len(events) == 0 {
		return nil
	}

	query := `
		INSERT INTO maintenance_events (
			id, upload_id, sap_order_number, order_type, installation_location,
			station_code, start_timestamp, end_timestamp, duration_hours,
			description, symptom, cause, action_taken, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		if e.ID == "" {
			e.ID = uuid.New().String()
		}
		_, err := stmt.ExecContext(ctx,
			e.ID,
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

func (r *PostgresMaintenanceRepository) Create(ctx context.Context, event *domain.MaintenanceEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	query := `
		INSERT INTO maintenance_events (
			id, sap_order_number, order_type, installation_location,
			station_code, start_timestamp, end_timestamp, duration_hours,
			description, symptom, cause, action_taken, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.ID,
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
	return err
}

func (r *PostgresMaintenanceRepository) List(ctx context.Context, limit, offset int) ([]*domain.MaintenanceEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, COALESCE(upload_id, ''), COALESCE(sap_order_number, ''), order_type, COALESCE(installation_location, ''),
		       COALESCE(station_code, ''), start_timestamp, COALESCE(end_timestamp, ''), duration_hours,
		       COALESCE(description, ''), COALESCE(symptom, ''), COALESCE(cause, ''), COALESCE(action_taken, ''),
		       source, created_at
		FROM maintenance_events
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.MaintenanceEvent
	for rows.Next() {
		var e domain.MaintenanceEvent
		var createdAt time.Time

		err := rows.Scan(
			&e.ID,
			&e.UploadID,
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
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		e.CreatedAt = createdAt
		list = append(list, &e)
	}

	return list, nil
}
