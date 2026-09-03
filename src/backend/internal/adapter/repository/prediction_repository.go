package repository

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
)

var _ domain.PredictionRepository = (*PostgresPredictionRepository)(nil)

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

	stmt, err := r.db.PrepareContext(ctx, `
		INSERT INTO predictions (id, run_id, station_code, platform, door_number, will_fail, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		_, err := stmt.ExecContext(ctx,
			item.ID,
			run.ID,
			item.StationCode,
			item.Platform,
			item.DoorNumber,
			item.WillFail,
			now,
		)
		if err != nil {
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
		return nil, err
	}
	run.CreatedAt = createdAt

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, run_id, station_code, platform, door_number, will_fail
		FROM predictions
		WHERE run_id = $1 AND will_fail = TRUE
		ORDER BY station_code ASC, platform ASC, door_number ASC
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
