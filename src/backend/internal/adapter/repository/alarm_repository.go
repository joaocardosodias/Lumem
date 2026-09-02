package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.AlarmRepository = (*SQLiteAlarmRepository)(nil)

type SQLiteAlarmRepository struct {
	db *sql.DB
}

func NewSQLiteAlarmRepository(db *sql.DB) *SQLiteAlarmRepository {
	return &SQLiteAlarmRepository{db: db}
}

func (r *SQLiteAlarmRepository) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	query := `
		INSERT INTO file_uploads (file_name, file_type, rows_processed, rows_inserted, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	upload.CreatedAt = now
	upload.Status = "processing"

	res, err := r.db.ExecContext(
		ctx,
		query,
		upload.FileName,
		upload.FileType,
		upload.RowsProcessed,
		upload.RowsInserted,
		upload.Status,
		now.Format("2006-01-02 15:04:05"),
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
	query := `
		UPDATE file_uploads
		SET rows_processed = ?, rows_inserted = ?, status = ?, error_log = ?, completed_at = ?
		WHERE id = ?
	`
	var completedAtStr *string
	if upload.CompletedAt != nil {
		s := upload.CompletedAt.Format("2006-01-02 15:04:05")
		completedAtStr = &s
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		upload.RowsProcessed,
		upload.RowsInserted,
		upload.Status,
		upload.ErrorLog,
		completedAtStr,
		upload.ID,
	)
	return err
}

func parseTime(raw string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (r *SQLiteAlarmRepository) GetUploadByID(ctx context.Context, id int64) (*domain.FileUpload, error) {
	query := `
		SELECT id, file_name, file_type, rows_processed, rows_inserted, status, IFNULL(error_log, ''), created_at, completed_at
		FROM file_uploads
		WHERE id = ?
		LIMIT 1
	`
	var upload domain.FileUpload
	var createdAtStr string
	var completedAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&upload.ID,
		&upload.FileName,
		&upload.FileType,
		&upload.RowsProcessed,
		&upload.RowsInserted,
		&upload.Status,
		&upload.ErrorLog,
		&createdAtStr,
		&completedAtStr,
	)
	if err != nil {
		return nil, err
	}

	upload.CreatedAt = parseTime(createdAtStr)

	if completedAtStr.Valid {
		t := parseTime(completedAtStr.String)
		if !t.IsZero() {
			upload.CompletedAt = &t
		}
	}

	return &upload, nil
}

func (r *SQLiteAlarmRepository) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	if len(alarms) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO telemetry_alarms (
			upload_id, tag, station_code, platform, door_number,
			alarm_type, duration_seconds, event_timestamp, normalization_timestamp, source
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range alarms {
		_, err := stmt.ExecContext(
			ctx,
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

	return tx.Commit()
}
