package repository

import (
	"context"
	"database/sql"

	"github.com/g04/backend/internal/domain"
)

var _ domain.MaintenanceRepository = (*SQLiteMaintenanceRepository)(nil)

type SQLiteMaintenanceRepository struct {
	db *sql.DB
}

func NewSQLiteMaintenanceRepository(db *sql.DB) *SQLiteMaintenanceRepository {
	return &SQLiteMaintenanceRepository{db: db}
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
