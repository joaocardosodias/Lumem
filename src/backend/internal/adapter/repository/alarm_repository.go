package repository

import (
	"context"
	"database/sql"
	"errors"

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
		INSERT INTO file_uploads (file_name, file_type, file_hash, status, rows_processed, rows_inserted)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query, upload.FileName, upload.FileType, upload.FileHash, upload.Status, upload.RowsProcessed, upload.RowsInserted)
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
		SET status = ?, rows_processed = ?, rows_inserted = ?, error_log = ?, completed_at = ?
		WHERE id = ?
	`
	var completedAtStr *string
	if upload.CompletedAt != nil {
		s := upload.CompletedAt.Format("2006-01-02 15:04:05")
		completedAtStr = &s
	}

	_, err := r.db.ExecContext(ctx, query, upload.Status, upload.RowsProcessed, upload.RowsInserted, upload.ErrorLog, completedAtStr, upload.ID)
	return err
}

func (r *SQLiteAlarmRepository) GetUploadByID(ctx context.Context, id int64) (*domain.FileUpload, error) {
	query := `
		SELECT id, file_name, file_type, IFNULL(file_hash, ''), status, rows_processed, rows_inserted, IFNULL(error_log, ''), created_at, completed_at
		FROM file_uploads
		WHERE id = ?
	`
	var u domain.FileUpload
	var createdAtStr string
	var completedAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	u.CreatedAt = parseTime(createdAtStr)
	if completedAtStr.Valid {
		t := parseTime(completedAtStr.String)
		if !t.IsZero() {
			u.CompletedAt = &t
		}
	}

	return &u, nil
}

func (r *SQLiteAlarmRepository) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	if hash == "" {
		return nil, domain.ErrUserNotFound
	}

	query := `
		SELECT id, file_name, file_type, IFNULL(file_hash, ''), status, rows_processed, rows_inserted, IFNULL(error_log, ''), created_at, completed_at
		FROM file_uploads
		WHERE file_hash = ? AND status = 'completed'
		LIMIT 1
	`
	var u domain.FileUpload
	var createdAtStr string
	var completedAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, query, hash).Scan(
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	u.CreatedAt = parseTime(createdAtStr)
	if completedAtStr.Valid {
		t := parseTime(completedAtStr.String)
		if !t.IsZero() {
			u.CompletedAt = &t
		}
	}

	return &u, nil
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
			alarm_type, duration_seconds, event_timestamp,
			normalization_timestamp, source
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
