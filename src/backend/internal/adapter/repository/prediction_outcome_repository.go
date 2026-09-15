package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

var _ port.PredictionOutcomeRepository = (*PostgresPredictionOutcomeRepository)(nil)

type PostgresPredictionOutcomeRepository struct {
	db DBExecutor
}

func NewPostgresPredictionOutcomeRepository(db DBExecutor) *PostgresPredictionOutcomeRepository {
	return &PostgresPredictionOutcomeRepository{db: db}
}

func (r *PostgresPredictionOutcomeRepository) EvaluateMaturePredictions(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO prediction_outcomes (
			prediction_id, window_started_at, window_ended_at,
			failure_observed, failure_source, outcome, evaluated_at
		)
		SELECT p.id,
		       pr.created_at,
		       pr.created_at + make_interval(days => pr.horizon_days),
		       failure.id IS NOT NULL,
		       COALESCE('maintenance_events:' || failure.id, 'maintenance_events:ZM01:none'),
		       CASE
		           WHEN p.will_fail AND failure.id IS NOT NULL THEN 'true_positive'
		           WHEN p.will_fail AND failure.id IS NULL THEN 'false_positive'
		           WHEN NOT p.will_fail AND failure.id IS NOT NULL THEN 'false_negative'
		           ELSE 'true_negative'
		       END,
		       CURRENT_TIMESTAMP
		FROM predictions p
		JOIN prediction_runs pr ON pr.id = p.run_id
		LEFT JOIN LATERAL (
			SELECT m.id
			FROM maintenance_events m
			WHERE m.order_type = 'ZM01'
			  AND m.station_code = p.station_code
			  AND CASE
			      WHEN m.start_timestamp ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}' THEN LEFT(m.start_timestamp, 10)::date
			      WHEN m.start_timestamp ~ '^[0-9]{2}/[0-9]{2}/[0-9]{4}' THEN TO_DATE(LEFT(m.start_timestamp, 10), 'DD/MM/YYYY')
			      ELSE NULL
			  END >= pr.created_at::date
			  AND CASE
			      WHEN m.start_timestamp ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}' THEN LEFT(m.start_timestamp, 10)::date
			      WHEN m.start_timestamp ~ '^[0-9]{2}/[0-9]{2}/[0-9]{4}' THEN TO_DATE(LEFT(m.start_timestamp, 10), 'DD/MM/YYYY')
			      ELSE NULL
			  END <= (pr.created_at + make_interval(days => pr.horizon_days))::date
			  AND m.installation_location ~ (
			      'PSD-' || p.station_code || '-(PLT' || regexp_replace(p.platform, '[^0-9]', '', 'g') ||
			      '|P' || regexp_replace(p.platform, '[^0-9]', '', 'g') || 'S[0-9]+)-PP' ||
			      lpad(p.door_number::text, 2, '0')
			  )
			ORDER BY m.start_timestamp
			LIMIT 1
		) failure ON TRUE
		WHERE pr.is_experiment = FALSE
		  AND pr.published_at IS NOT NULL
		  AND pr.created_at + make_interval(days => pr.horizon_days) <= CURRENT_TIMESTAMP
		ON CONFLICT (prediction_id) DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *PostgresPredictionOutcomeRepository) GetProductionPerformance(ctx context.Context) (*domain.ProductionPerformance, error) {
	var performance domain.ProductionPerformance
	var lastEvaluated sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		WITH eligible_runs AS (
			SELECT pr.id, pr.model_version, pr.threshold_used, pr.created_at
			FROM prediction_runs pr
			WHERE pr.is_experiment = FALSE
			  AND pr.published_at IS NOT NULL
			  AND pr.created_at + make_interval(days => pr.horizon_days) <= CURRENT_TIMESTAMP
		), cohort AS (
			SELECT model_version, threshold_used
			FROM eligible_runs
			ORDER BY created_at DESC
			LIMIT 1
		), mature AS (
			SELECT er.id
			FROM eligible_runs er
			JOIN cohort c
			  ON c.model_version = er.model_version
			 AND (
				(c.threshold_used IS NULL AND er.threshold_used IS NULL)
				OR ABS(c.threshold_used - er.threshold_used) < 0.000001
			 )
		), mature_summary AS (
			SELECT COALESCE((SELECT model_version FROM cohort), '') AS model_version,
			       (SELECT threshold_used FROM cohort) AS threshold_used,
			       COUNT(DISTINCT m.id) AS mature_runs,
			       COUNT(p.id) AS total_predictions
			FROM mature m
			LEFT JOIN predictions p ON p.run_id = m.id
		), outcome_summary AS (
			SELECT COUNT(*) AS evaluated_predictions,
			       COUNT(*) FILTER (WHERE po.outcome = 'true_positive') AS true_positives,
			       COUNT(*) FILTER (WHERE po.outcome = 'false_positive') AS false_positives,
			       COUNT(*) FILTER (WHERE po.outcome = 'false_negative') AS false_negatives,
			       COUNT(*) FILTER (WHERE po.outcome = 'true_negative') AS true_negatives,
			       MAX(po.evaluated_at) AS last_evaluated_at
			FROM prediction_outcomes po
			JOIN predictions p ON p.id = po.prediction_id
			WHERE p.run_id IN (SELECT id FROM mature)
		)
		SELECT ms.model_version, ms.threshold_used, ms.mature_runs,
		       os.evaluated_predictions, ms.total_predictions,
		       os.true_positives, os.false_positives, os.false_negatives,
		       os.true_negatives, os.last_evaluated_at
		FROM mature_summary ms CROSS JOIN outcome_summary os
	`).Scan(&performance.ModelVersion, &performance.Threshold, &performance.MatureRuns,
		&performance.EvaluatedItems, &performance.TotalItems,
		&performance.TruePositives, &performance.FalsePositives,
		&performance.FalseNegatives, &performance.TrueNegatives, &lastEvaluated)
	if err != nil {
		return nil, err
	}
	if performance.TotalItems > 0 {
		performance.LabelCoverage = float64(performance.EvaluatedItems) / float64(performance.TotalItems)
	}
	if performance.TruePositives+performance.FalsePositives > 0 {
		value := float64(performance.TruePositives) / float64(performance.TruePositives+performance.FalsePositives)
		performance.Precision = &value
	}
	if performance.TruePositives+performance.FalseNegatives > 0 {
		value := float64(performance.TruePositives) / float64(performance.TruePositives+performance.FalseNegatives)
		performance.Recall = &value
	}
	if performance.Precision != nil && performance.Recall != nil && *performance.Precision+*performance.Recall > 0 {
		value := 2 * *performance.Precision * *performance.Recall / (*performance.Precision + *performance.Recall)
		performance.F1Score = &value
	}
	if lastEvaluated.Valid {
		value := lastEvaluated.Time.Format(time.RFC3339)
		performance.LastEvaluatedAt = &value
	}
	performance.FailureDefinition = "ordem_corretiva_ZM01_na_mesma_porta_em_7_dias"
	return &performance, nil
}
