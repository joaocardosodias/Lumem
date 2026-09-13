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
	if !run.IsExperiment && run.PublishedAt == nil {
		run.PublishedAt = &now
	}
	if run.ID == "" {
		run.ID = uuid.New().String()
	}

	query := `
		INSERT INTO prediction_runs (id, horizon_days, threshold_used, model_version, is_experiment, published_at, total_doors_evaluated, total_doors_at_risk, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.HorizonDays,
		run.ThresholdUsed,
		run.ModelVersion,
		run.IsExperiment,
		run.PublishedAt,
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
		queryBuilder.WriteString("INSERT INTO predictions (id, run_id, station_code, platform, door_number, will_fail, probability, created_at) VALUES ")

		args := make([]interface{}, 0, len(chunk)*8)
		for j := range chunk {
			if chunk[j].ID == "" {
				chunk[j].ID = uuid.New().String()
			}
			if j > 0 {
				queryBuilder.WriteString(", ")
			}
			idx := j * 8
			fmt.Fprintf(&queryBuilder, "($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				idx+1, idx+2, idx+3, idx+4, idx+5, idx+6, idx+7, idx+8)

			args = append(args, chunk[j].ID, run.ID, chunk[j].StationCode, chunk[j].Platform, chunk[j].DoorNumber, chunk[j].WillFail, chunk[j].Probability, now)
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
		SELECT id, horizon_days, threshold_used, model_version, is_experiment, published_at, total_doors_evaluated, total_doors_at_risk, created_at
		FROM prediction_runs
		WHERE is_experiment = FALSE AND published_at IS NOT NULL
		ORDER BY published_at DESC
		LIMIT 1
	`).Scan(&run.ID, &run.HorizonDays, &run.ThresholdUsed, &run.ModelVersion, &run.IsExperiment, &run.PublishedAt, &run.TotalDoorsEvaluated, &run.TotalDoorsAtRisk, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNoPredictions
		}
		return nil, err
	}
	run.CreatedAt = createdAt

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, run_id, station_code, platform, door_number, will_fail, probability
		FROM predictions
		WHERE run_id = $1
		ORDER BY will_fail DESC, station_code ASC, platform ASC, door_number ASC
	`, run.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p domain.PredictionItem
			if err := rows.Scan(&p.ID, &p.RunID, &p.StationCode, &p.Platform, &p.DoorNumber, &p.WillFail, &p.Probability); err == nil {
				run.Predictions = append(run.Predictions, p)
			}
		}
	}

	return &run, nil
}

func (r *PostgresPredictionRepository) GetRunByID(ctx context.Context, id string) (*domain.PredictionRun, error) {
	var run domain.PredictionRun
	err := r.db.QueryRowContext(ctx, `
		SELECT id, horizon_days, threshold_used, model_version, is_experiment, published_at,
		       total_doors_evaluated, total_doors_at_risk, created_at
		FROM prediction_runs WHERE id = $1
	`, id).Scan(&run.ID, &run.HorizonDays, &run.ThresholdUsed, &run.ModelVersion, &run.IsExperiment,
		&run.PublishedAt, &run.TotalDoorsEvaluated, &run.TotalDoorsAtRisk, &run.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrExperimentNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, run_id, station_code, platform, door_number, will_fail, probability
		FROM predictions WHERE run_id = $1
		ORDER BY will_fail DESC, station_code, platform, door_number
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item domain.PredictionItem
		if err := rows.Scan(&item.ID, &item.RunID, &item.StationCode, &item.Platform, &item.DoorNumber, &item.WillFail, &item.Probability); err != nil {
			return nil, err
		}
		run.Predictions = append(run.Predictions, item)
	}
	return &run, rows.Err()
}

func (r *PostgresPredictionRepository) ListExperiments(ctx context.Context, limit int) ([]domain.PredictionRun, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT pr.id, pr.horizon_days, pr.threshold_used, pr.model_version, pr.is_experiment, pr.published_at,
		       pr.total_doors_evaluated, pr.total_doors_at_risk, pr.created_at,
		       AVG(p.probability),
		       PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY p.probability),
		       MAX(p.probability),
		       COALESCE(AVG(p.probability) FILTER (WHERE p.will_fail), 0),
		       COALESCE(SUM(p.probability) FILTER (WHERE p.will_fail), 0) / NULLIF(SUM(p.probability), 0)
		FROM prediction_runs pr
		LEFT JOIN predictions p ON p.run_id = pr.id AND p.probability IS NOT NULL
		WHERE pr.is_experiment = TRUE
		GROUP BY pr.id
		ORDER BY pr.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := make([]domain.PredictionRun, 0)
	for rows.Next() {
		var run domain.PredictionRun
		if err := rows.Scan(&run.ID, &run.HorizonDays, &run.ThresholdUsed, &run.ModelVersion, &run.IsExperiment,
			&run.PublishedAt, &run.TotalDoorsEvaluated, &run.TotalDoorsAtRisk, &run.CreatedAt,
			&run.AverageProbability, &run.P95Probability, &run.MaximumProbability,
			&run.EstimatedPrecision, &run.EstimatedRecall); err != nil {
			return nil, err
		}
		if run.EstimatedPrecision != nil && run.EstimatedRecall != nil {
			value := 0.0
			if *run.EstimatedPrecision+*run.EstimatedRecall > 0 {
				value = 2 * *run.EstimatedPrecision * *run.EstimatedRecall / (*run.EstimatedPrecision + *run.EstimatedRecall)
			}
			run.EstimatedF1 = &value
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (r *PostgresPredictionRepository) PromoteExperiment(ctx context.Context, id, userID string) error {
	result, err := r.db.ExecContext(ctx, `
		WITH promoted AS (
			UPDATE prediction_runs
			SET is_experiment = FALSE, published_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND is_experiment = TRUE
			RETURNING threshold_used, horizon_days
		)
		UPDATE ml_configs
		SET threshold = COALESCE((SELECT threshold_used FROM promoted), threshold),
		    default_horizon_days = COALESCE((SELECT horizon_days FROM promoted), default_horizon_days),
		    updated_at = CURRENT_TIMESTAMP,
		    updated_by = NULLIF($2, '')
		WHERE id = 'active' AND EXISTS (SELECT 1 FROM promoted)
	`, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrExperimentNotFound
	}
	return nil
}

func (r *PostgresPredictionRepository) GetDoorHistory(ctx context.Context, doorID, limit int) ([]domain.DoorPredictionHistoryItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.run_id, p.station_code, p.platform, p.door_number,
		       p.will_fail, p.probability, pr.horizon_days, pr.threshold_used,
		       pr.model_version, pr.created_at
		FROM doors d
		JOIN stations s ON s.id = d.station_id
		JOIN predictions p ON p.station_code = s.code
		    AND p.platform = d.platform
		    AND p.door_number = d.door_number
		JOIN prediction_runs pr ON pr.id = p.run_id
		WHERE d.id = $1
		  AND pr.is_experiment = FALSE AND pr.published_at IS NOT NULL
		ORDER BY pr.created_at DESC
		LIMIT $2
	`, doorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]domain.DoorPredictionHistoryItem, 0)
	for rows.Next() {
		var item domain.DoorPredictionHistoryItem
		if err := rows.Scan(
			&item.ID, &item.RunID, &item.StationCode, &item.Platform, &item.DoorNumber,
			&item.WillFail, &item.Probability, &item.HorizonDays, &item.ThresholdUsed,
			&item.ModelVersion, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, rows.Err()
}
