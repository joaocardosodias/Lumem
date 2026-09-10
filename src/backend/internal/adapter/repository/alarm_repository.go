package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ port.AlarmRepository = (*PostgresAlarmRepository)(nil)
var _ port.UploadRepository = (*PostgresAlarmRepository)(nil)

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
	if err != nil {
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr) && pgErr.Code == "23505") || strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "idx_file_uploads_hash") {
			return domain.ErrDuplicateUpload
		}
		return err
	}
	return nil
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUploadNotFound
		}
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

func (r *PostgresAlarmRepository) DeleteByUploadID(ctx context.Context, uploadID string) error {
	query := `DELETE FROM telemetry_alarms WHERE upload_id = $1`
	_, err := r.db.ExecContext(ctx, query, uploadID)
	return err
}

func (r *PostgresAlarmRepository) ReconcileStaleUploads(ctx context.Context) error {
	query := `
		UPDATE file_uploads
		SET status = 'failed', error_log = 'processo interrompido por reinicializacao do servidor', completed_at = NOW()
		WHERE status = 'processing'
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *PostgresAlarmRepository) List(ctx context.Context, filter port.AlarmFilter) ([]*domain.Alarm, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := `
		SELECT id, COALESCE(upload_id, ''), tag, station_code,
		       COALESCE(platform, ''), COALESCE(door_number, 0),
		       alarm_type, duration_seconds, event_timestamp,
		       COALESCE(normalization_timestamp, ''), source, created_at
		FROM telemetry_alarms
	`
	var conditions []string
	var args []any
	argIdx := 1

	if filter.StationCode != "" {
		conditions = append(conditions, fmt.Sprintf("station_code = $%d", argIdx))
		args = append(args, strings.ToUpper(filter.StationCode))
		argIdx++
	}
	if filter.Platform != "" {
		conditions = append(conditions, fmt.Sprintf("platform = $%d", argIdx))
		args = append(args, strings.ToUpper(filter.Platform))
		argIdx++
	}
	if filter.DoorNumber > 0 {
		conditions = append(conditions, fmt.Sprintf("door_number = $%d", argIdx))
		args = append(args, filter.DoorNumber)
		argIdx++
	}
	if filter.AlarmType != "" {
		conditions = append(conditions, fmt.Sprintf("alarm_type = $%d", argIdx))
		args = append(args, filter.AlarmType)
		argIdx++
	}
	if filter.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("tag ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Tag+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY event_timestamp DESC, created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alarms []*domain.Alarm
	for rows.Next() {
		var a domain.Alarm
		var dur float64
		var createdAt time.Time

		err := rows.Scan(
			&a.ID,
			&a.UploadID,
			&a.Tag,
			&a.StationCode,
			&a.Platform,
			&a.DoorNumber,
			&a.AlarmType,
			&dur,
			&a.EventTimestamp,
			&a.NormalizationTimestamp,
			&a.Source,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		a.DurationSeconds = int(dur)
		a.CreatedAt = createdAt
		alarms = append(alarms, &a)
	}

	return alarms, nil
}
