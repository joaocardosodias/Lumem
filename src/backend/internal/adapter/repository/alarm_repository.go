package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
)

var _ domain.AlarmRepository = (*PostgresAlarmRepository)(nil)

type PostgresAlarmRepository struct {
	db DBExecutor
}

func NewPostgresAlarmRepository(db DBExecutor) *PostgresAlarmRepository {
	return &PostgresAlarmRepository{db: db}
}

func (r *PostgresAlarmRepository) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	if upload.ID == "" {
		upload.ID = uuid.New().String()
	}

	query := `
		INSERT INTO file_uploads (id, file_name, file_type, file_hash, status, rows_processed, rows_inserted, error_log)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		upload.ID,
		upload.FileName,
		upload.FileType,
		upload.FileHash,
		upload.Status,
		upload.RowsProcessed,
		upload.RowsInserted,
		upload.ErrorLog,
	)
	return err
}

func (r *PostgresAlarmRepository) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	query := `
		UPDATE file_uploads
		SET status = $1, rows_processed = $2, rows_inserted = $3, error_log = $4, completed_at = $5
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		upload.Status,
		upload.RowsProcessed,
		upload.RowsInserted,
		upload.ErrorLog,
		upload.CompletedAt,
		upload.ID,
	)
	return err
}

func (r *PostgresAlarmRepository) GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error) {
	query := `
		SELECT id, file_name, file_type, COALESCE(file_hash, ''), status, rows_processed, rows_inserted, COALESCE(error_log, ''), created_at, completed_at
		FROM file_uploads
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var u domain.FileUpload
	var createdAt time.Time
	var completedAt *time.Time

	err := row.Scan(
		&u.ID,
		&u.FileName,
		&u.FileType,
		&u.FileHash,
		&u.Status,
		&u.RowsProcessed,
		&u.RowsInserted,
		&u.ErrorLog,
		&createdAt,
		&completedAt,
	)
	if err != nil {
		return nil, err
	}

	u.CreatedAt = createdAt
	u.CompletedAt = completedAt
	return &u, nil
}

func (r *PostgresAlarmRepository) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	query := `
		SELECT id, file_name, file_type, COALESCE(file_hash, ''), status, rows_processed, rows_inserted, COALESCE(error_log, ''), created_at, completed_at
		FROM file_uploads
		WHERE file_hash = $1
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, hash)

	var u domain.FileUpload
	var createdAt time.Time
	var completedAt *time.Time

	err := row.Scan(
		&u.ID,
		&u.FileName,
		&u.FileType,
		&u.FileHash,
		&u.Status,
		&u.RowsProcessed,
		&u.RowsInserted,
		&u.ErrorLog,
		&createdAt,
		&completedAt,
	)
	if err != nil {
		return nil, err
	}

	u.CreatedAt = createdAt
	u.CompletedAt = completedAt
	return &u, nil
}

func (r *PostgresAlarmRepository) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	if len(alarms) == 0 {
		return nil
	}

	const chunkSize = 50
	for i := 0; i < len(alarms); i += chunkSize {
		end := i + chunkSize
		if end > len(alarms) {
			end = len(alarms)
		}
		chunk := alarms[i:end]

		var sb strings.Builder
		sb.WriteString(`
			INSERT INTO telemetry_alarms (
				id, upload_id, tag, station_code, platform, door_number,
				alarm_type, duration_seconds, event_timestamp,
				normalization_timestamp, source
			) VALUES 
		`)

		vals := make([]any, 0, len(chunk)*11)
		for j, a := range chunk {
			if a.ID == "" {
				a.ID = uuid.New().String()
			}
			if j > 0 {
				sb.WriteString(", ")
			}
			idx := j * 11
			fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				idx+1, idx+2, idx+3, idx+4, idx+5, idx+6, idx+7, idx+8, idx+9, idx+10, idx+11)

			vals = append(vals,
				a.ID, a.UploadID, a.Tag, a.StationCode, a.Platform, a.DoorNumber,
				a.AlarmType, a.DurationSeconds, a.EventTimestamp,
				a.NormalizationTimestamp, a.Source,
			)
		}

		if _, err := r.db.ExecContext(ctx, sb.String(), vals...); err != nil {
			return err
		}
	}

	return nil
}
