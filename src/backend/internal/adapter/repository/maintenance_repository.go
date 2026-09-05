package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

var _ port.MaintenanceRepository = (*PostgresMaintenanceRepository)(nil)

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

	const chunkSize = 50
	for i := 0; i < len(events); i += chunkSize {
		end := i + chunkSize
		if end > len(events) {
			end = len(events)
		}
		chunk := events[i:end]

		var sb strings.Builder
		sb.WriteString(`
			INSERT INTO maintenance_events (
				id, upload_id, sap_order_number, order_type, installation_location,
				station_code, start_timestamp, end_timestamp, duration_hours,
				description, symptom, cause, action_taken, source
			) VALUES 
		`)

		vals := make([]any, 0, len(chunk)*14)
		for j, e := range chunk {
			if e.ID == "" {
				e.ID = uuid.New().String()
			}
			if j > 0 {
				sb.WriteString(", ")
			}
			idx := j * 14
			fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				idx+1, idx+2, idx+3, idx+4, idx+5, idx+6, idx+7, idx+8, idx+9, idx+10, idx+11, idx+12, idx+13, idx+14)

			vals = append(vals,
				e.ID, e.UploadID, e.SAPOrderNumber, e.OrderType, e.InstallationLocation,
				e.StationCode, e.StartTimestamp, e.EndTimestamp, e.DurationHours,
				e.Description, e.Symptom, e.Cause, e.ActionTaken, e.Source,
			)
		}

		if _, err := r.db.ExecContext(ctx, sb.String(), vals...); err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresMaintenanceRepository) DeleteByUploadID(ctx context.Context, uploadID string) error {
	query := `DELETE FROM maintenance_events WHERE upload_id = $1`
	_, err := r.db.ExecContext(ctx, query, uploadID)
	return err
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
