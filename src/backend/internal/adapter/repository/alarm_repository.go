package repository

import (
	"context"

	"github.com/g04/backend/internal/domain"
)

var _ domain.AlarmRepository = (*SQLiteAlarmRepository)(nil)

type SQLiteAlarmRepository struct {
	db DBExecutor
}

func NewSQLiteAlarmRepository(db DBExecutor) *SQLiteAlarmRepository {
	return &SQLiteAlarmRepository{db: db}
}

func (r *SQLiteAlarmRepository) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	query := `
		INSERT INTO file_uploads (file_name, file_type, file_hash, status, rows_processed, rows_inserted, error_log)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query,
		upload.FileName,
		upload.FileType,
		upload.FileHash,
		upload.Status,
		upload.RowsProcessed,
		upload.RowsInserted,
		upload.ErrorLog,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	upload.ID = id
	return nil
}

func (r *SQLiteAlarmRepository) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	var completedAtStr *string
	if upload.CompletedAt != nil {
		s := upload.CompletedAt.Format("2006-01-02 15:04:05")
		completedAtStr = &s
	}

	query := `
		UPDATE file_uploads
		SET status = ?, rows_processed = ?, rows_inserted = ?, error_log = ?, completed_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		upload.Status,
		upload.RowsProcessed,
		upload.RowsInserted,
		upload.ErrorLog,
		completedAtStr,
		upload.ID,
	)
	return err
}

func (r *SQLiteAlarmRepository) GetUploadByID(ctx context.Context, id int64) (*domain.FileUpload, error) {
	query := `
		SELECT id, file_name, file_type, file_hash, status, rows_processed, rows_inserted, error_log, created_at, completed_at
		FROM file_uploads
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.FileUpload
	var createdAtStr string
	var completedAtStr *string

	err := row.Scan(
		&u.ID,
		&u.FileName,
		&u.FileType,
		&u.FileHash,
		&u.Status,
		&u.RowsProcessed,
		&u.RowsInserted,
		&u.ErrorLog,
		&createdAtStr,
		&completedAtStr,
	)
	if err != nil {
		return nil, err
	}

	u.CreatedAt = parseTime(createdAtStr)
	if completedAtStr != nil {
		t := parseTime(*completedAtStr)
		u.CompletedAt = &t
	}

	return &u, nil
}

func (r *SQLiteAlarmRepository) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	query := `
		SELECT id, file_name, file_type, file_hash, status, rows_processed, rows_inserted, error_log, created_at, completed_at
		FROM file_uploads
		WHERE file_hash = ?
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, hash)

	var u domain.FileUpload
	var createdAtStr string
	var completedAtStr *string

	err := row.Scan(
		&u.ID,
		&u.FileName,
		&u.FileType,
		&u.FileHash,
		&u.Status,
		&u.RowsProcessed,
		&u.RowsInserted,
		&u.ErrorLog,
		&createdAtStr,
		&completedAtStr,
	)
	if err != nil {
		return nil, err
	}

	u.CreatedAt = parseTime(createdAtStr)
	if completedAtStr != nil {
		t := parseTime(*completedAtStr)
		u.CompletedAt = &t
	}

	return &u, nil
}

func (r *SQLiteAlarmRepository) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	if len(alarms) == 0 {
		return nil
	}

	query := `
		INSERT INTO telemetry_alarms (
			upload_id, tag, station_code, platform, door_number,
			alarm_type, duration_seconds, event_timestamp,
			normalization_timestamp, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range alarms {
		_, err := stmt.ExecContext(ctx,
			a.UploadID,
			a.Tag,
			a.StationCode,
			a.Platform,
			a.DoorNumber,
			a.AlarmType,
			a.DurationSeconds,
			a.EventTimestamp,
			a.NormalizationTimestamp,
			a.Source,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
