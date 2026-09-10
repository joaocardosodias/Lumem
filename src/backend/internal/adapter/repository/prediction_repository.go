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
)

var _ port.PredictionRepository = (*PostgresPredictionRepository)(nil)

type PostgresPredictionRepository struct {
	db DBExecutor
}

func NewPostgresPredictionRepository(db DBExecutor) *PostgresPredictionRepository {
	return &PostgresPredictionRepository{db: db}
}

func (r *PostgresPredictionRepository) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.station_id, s.code, d.platform, d.door_number, d.sap_tag, d.status, d.created_at
		FROM doors d
		JOIN stations s ON d.station_id = s.id
		WHERE d.status = 'active'
		ORDER BY d.station_id ASC, d.platform ASC, d.door_number ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var doors []domain.Door
	for rows.Next() {
		var d domain.Door
		var createdAt time.Time
		if err := rows.Scan(&d.ID, &d.StationID, &d.StationCode, &d.Platform, &d.DoorNumber, &d.SAPTag, &d.Status, &createdAt); err != nil {
			return nil, err
		}
		d.CreatedAt = createdAt
		doors = append(doors, d)
	}
	return doors, rows.Err()
}

func (r *PostgresPredictionRepository) SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	now := time.Now()
	run.CreatedAt = now
	if run.ID == "" {
		run.ID = uuid.New().String()
	}

	query := `
		INSERT INTO prediction_runs (id, horizon_days, total_doors_evaluated, total_doors_at_risk, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.HorizonDays,
		run.TotalDoorsEvaluated,
		run.TotalDoorsAtRisk,
		now,
	)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	const batchSize = 100
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		chunk := items[i:end]

		var queryBuilder strings.Builder
		queryBuilder.WriteString("INSERT INTO predictions (id, run_id, station_code, platform, door_number, will_fail, created_at) VALUES ")

		args := make([]interface{}, 0, len(chunk)*7)
		for j := range chunk {
			if chunk[j].ID == "" {
				chunk[j].ID = uuid.New().String()
			}
			if j > 0 {
				queryBuilder.WriteString(", ")
			}
			idx := j * 7
			fmt.Fprintf(&queryBuilder, "($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				idx+1, idx+2, idx+3, idx+4, idx+5, idx+6, idx+7)

			args = append(args, chunk[j].ID, run.ID, chunk[j].StationCode, chunk[j].Platform, chunk[j].DoorNumber, chunk[j].WillFail, now)
		}

		if _, err := r.db.ExecContext(ctx, queryBuilder.String(), args...); err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresPredictionRepository) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	var run domain.PredictionRun
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT id, horizon_days, total_doors_evaluated, total_doors_at_risk, created_at
		FROM prediction_runs
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&run.ID, &run.HorizonDays, &run.TotalDoorsEvaluated, &run.TotalDoorsAtRisk, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNoPredictions
		}
		return nil, err
	}
	run.CreatedAt = createdAt

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, run_id, station_code, platform, door_number, will_fail
		FROM predictions
		WHERE run_id = $1
		ORDER BY will_fail DESC, station_code ASC, platform ASC, door_number ASC
	`, run.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p domain.PredictionItem
			if err := rows.Scan(&p.ID, &p.RunID, &p.StationCode, &p.Platform, &p.DoorNumber, &p.WillFail); err == nil {
				run.Predictions = append(run.Predictions, p)
			}
		}
	}

	return &run, nil
}
