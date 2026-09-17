package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

var _ port.MLOpsRepository = (*PostgresMLOpsRepository)(nil)

type PostgresMLOpsRepository struct {
	db DBExecutor
}

func NewPostgresMLOpsRepository(db DBExecutor) *PostgresMLOpsRepository {
	return &PostgresMLOpsRepository{db: db}
}

func (r *PostgresMLOpsRepository) GetConfig(ctx context.Context) (*domain.MLConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, threshold, default_horizon_days, auto_retrain_enabled, updated_at, COALESCE(updated_by, '')
		FROM ml_configs WHERE id = 'active' LIMIT 1
	`)
	var config domain.MLConfig
	var autoRetrain int
	if err := row.Scan(&config.ID, &config.Threshold, &config.DefaultHorizonDays, &autoRetrain, &config.UpdatedAt, &config.UpdatedBy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: configuracao ativa ausente", domain.ErrInvalidConfig)
		}
		return nil, err
	}
	config.AutoRetrainEnabled = autoRetrain == 1
	return &config, nil
}

func (r *PostgresMLOpsRepository) UpdateConfig(ctx context.Context, config *domain.MLConfig) error {
	autoRetrain := 0
	if config.AutoRetrainEnabled {
		autoRetrain = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ml_configs (id, threshold, default_horizon_days, auto_retrain_enabled, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			threshold = EXCLUDED.threshold,
			default_horizon_days = EXCLUDED.default_horizon_days,
			auto_retrain_enabled = EXCLUDED.auto_retrain_enabled,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`, config.ID, config.Threshold, config.DefaultHorizonDays, autoRetrain, time.Now(), config.UpdatedBy)
	return err
}

const modelColumns = `
	id, version, algorithm, status, train_metrics, test_metrics, feature_importance,
	artifact_path, artifact_uri, artifact_sha256, dataset_sha256, target_name,
	train_end, test_start, test_end, test_rows, positive_rows, recommended_threshold,
	manifest, started_at, finished_at, activated_at, COALESCE(created_by, '')
`

type scanner interface {
	Scan(dest ...any) error
}

func scanModel(row scanner) (*domain.MLModelRun, error) {
	var model domain.MLModelRun
	var trainJSON, testJSON, featureJSON, manifestJSON []byte
	err := row.Scan(
		&model.ID, &model.Version, &model.Algorithm, &model.Status,
		&trainJSON, &testJSON, &featureJSON,
		&model.ArtifactPath, &model.ArtifactURI, &model.ArtifactSHA256,
		&model.DatasetSHA256, &model.TargetName,
		&model.TrainEnd, &model.TestStart, &model.TestEnd,
		&model.TestRows, &model.PositiveRows, &model.RecommendedThreshold,
		&manifestJSON, &model.StartedAt, &model.FinishedAt, &model.ActivatedAt, &model.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(trainJSON, &model.TrainMetrics); err != nil {
		return nil, fmt.Errorf("decodificar metricas de treino: %w", err)
	}
	if err := json.Unmarshal(testJSON, &model.TestMetrics); err != nil {
		return nil, fmt.Errorf("decodificar metricas de teste: %w", err)
	}
	if err := json.Unmarshal(featureJSON, &model.FeatureImportance); err != nil {
		return nil, fmt.Errorf("decodificar importancia de features: %w", err)
	}
	if err := json.Unmarshal(manifestJSON, &model.Manifest); err != nil {
		return nil, fmt.Errorf("decodificar manifest: %w", err)
	}
	return &model, nil
}

func (r *PostgresMLOpsRepository) GetActiveModel(ctx context.Context) (*domain.MLModelRun, error) {
	model, err := scanModel(r.db.QueryRowContext(ctx, `
		SELECT `+modelColumns+`
		FROM ml_model_runs
		WHERE status = 'production'
		ORDER BY activated_at DESC NULLS LAST, finished_at DESC
		LIMIT 1
	`))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrModelNotFound
	}
	if err != nil {
		return nil, err
	}
	model.ThresholdMetrics, err = r.GetThresholdMetrics(ctx, model.Version)
	return model, err
}

func (r *PostgresMLOpsRepository) GetModelByVersion(ctx context.Context, version string) (*domain.MLModelRun, error) {
	model, err := scanModel(r.db.QueryRowContext(ctx, `
		SELECT `+modelColumns+`
		FROM ml_model_runs WHERE version = $1 LIMIT 1
	`, version))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrModelNotFound
	}
	if err != nil {
		return nil, err
	}
	model.ThresholdMetrics, err = r.GetThresholdMetrics(ctx, version)
	return model, err
}

func (r *PostgresMLOpsRepository) ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+modelColumns+`
		FROM ml_model_runs
		ORDER BY finished_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := make([]domain.MLModelRun, 0)
	for rows.Next() {
		model, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *model)
	}
	return runs, rows.Err()
}

func marshalJSON(value any, fallback string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fallback
	}
	return string(encoded)
}

func (r *PostgresMLOpsRepository) SaveModelRun(ctx context.Context, run *domain.MLModelRun) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ml_model_runs (
			id, version, algorithm, status, train_metrics, test_metrics, feature_importance,
			artifact_path, artifact_uri, artifact_sha256, dataset_sha256, target_name,
			train_end, test_start, test_end, test_rows, positive_rows, recommended_threshold,
			manifest, started_at, finished_at, activated_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb,
			$8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
			$19::jsonb, $20, $21, $22, NULLIF($23, '')
		)
		ON CONFLICT (version) DO UPDATE SET
			algorithm = EXCLUDED.algorithm,
			status = CASE
				WHEN ml_model_runs.status = 'production' THEN ml_model_runs.status
				ELSE EXCLUDED.status
			END,
			train_metrics = EXCLUDED.train_metrics,
			test_metrics = EXCLUDED.test_metrics,
			feature_importance = EXCLUDED.feature_importance,
			artifact_uri = EXCLUDED.artifact_uri,
			artifact_sha256 = EXCLUDED.artifact_sha256,
			dataset_sha256 = EXCLUDED.dataset_sha256,
			target_name = EXCLUDED.target_name,
			train_end = EXCLUDED.train_end,
			test_start = EXCLUDED.test_start,
			test_end = EXCLUDED.test_end,
			test_rows = EXCLUDED.test_rows,
			positive_rows = EXCLUDED.positive_rows,
			recommended_threshold = EXCLUDED.recommended_threshold,
			manifest = EXCLUDED.manifest,
			finished_at = EXCLUDED.finished_at
	`, run.ID, run.Version, run.Algorithm, run.Status,
		marshalJSON(run.TrainMetrics, "{}"), marshalJSON(run.TestMetrics, "{}"), marshalJSON(run.FeatureImportance, "[]"),
		run.ArtifactPath, run.ArtifactURI, run.ArtifactSHA256, run.DatasetSHA256, run.TargetName,
		run.TrainEnd, run.TestStart, run.TestEnd, run.TestRows, run.PositiveRows, run.RecommendedThreshold,
		marshalJSON(run.Manifest, "{}"), run.StartedAt, run.FinishedAt, run.ActivatedAt, run.CreatedBy,
	)
	return err
}

func (r *PostgresMLOpsRepository) GetThresholdMetrics(ctx context.Context, version string) ([]domain.ThresholdMetric, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tm.threshold, tm.true_positives, tm.false_positives, tm.false_negatives,
		       tm.true_negatives, tm.precision_value, tm.recall_value, tm.f1_score,
		       tm.alert_count, tm.alert_rate
		FROM ml_threshold_metrics tm
		JOIN ml_model_runs mr ON mr.id = tm.model_run_id
		WHERE mr.version = $1
		ORDER BY tm.threshold
	`, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	metrics := make([]domain.ThresholdMetric, 0)
	for rows.Next() {
		var metric domain.ThresholdMetric
		if err := rows.Scan(&metric.Threshold, &metric.TruePositives, &metric.FalsePositives,
			&metric.FalseNegatives, &metric.TrueNegatives, &metric.Precision, &metric.Recall,
			&metric.F1Score, &metric.AlertCount, &metric.AlertRate); err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, rows.Err()
}

func (r *PostgresMLOpsRepository) GetThresholdMetric(ctx context.Context, version string, threshold float64) (*domain.ThresholdMetric, error) {
	var metric domain.ThresholdMetric
	err := r.db.QueryRowContext(ctx, `
		SELECT tm.threshold, tm.true_positives, tm.false_positives, tm.false_negatives,
		       tm.true_negatives, tm.precision_value, tm.recall_value, tm.f1_score,
		       tm.alert_count, tm.alert_rate
		FROM ml_threshold_metrics tm
		JOIN ml_model_runs mr ON mr.id = tm.model_run_id
		WHERE mr.version = $1
		ORDER BY ABS(tm.threshold - $2)
		LIMIT 1
	`, version, threshold).Scan(&metric.Threshold, &metric.TruePositives, &metric.FalsePositives,
		&metric.FalseNegatives, &metric.TrueNegatives, &metric.Precision, &metric.Recall,
		&metric.F1Score, &metric.AlertCount, &metric.AlertRate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrModelNotFound
	}
	return &metric, err
}

func (r *PostgresMLOpsRepository) SaveThresholdMetrics(ctx context.Context, modelRunID string, metrics []domain.ThresholdMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	var query strings.Builder
	query.WriteString(`INSERT INTO ml_threshold_metrics (
		model_run_id, threshold, true_positives, false_positives, false_negatives,
		true_negatives, precision_value, recall_value, f1_score, alert_count, alert_rate
	) VALUES `)
	args := make([]any, 0, len(metrics)*11)
	for index, metric := range metrics {
		if index > 0 {
			query.WriteString(",")
		}
		base := index*11 + 1
		fmt.Fprintf(&query, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base, base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10)
		args = append(args, modelRunID, metric.Threshold, metric.TruePositives, metric.FalsePositives,
			metric.FalseNegatives, metric.TrueNegatives, metric.Precision, metric.Recall,
			metric.F1Score, metric.AlertCount, metric.AlertRate)
	}
	query.WriteString(` ON CONFLICT (model_run_id, threshold) DO UPDATE SET
		true_positives = EXCLUDED.true_positives,
		false_positives = EXCLUDED.false_positives,
		false_negatives = EXCLUDED.false_negatives,
		true_negatives = EXCLUDED.true_negatives,
		precision_value = EXCLUDED.precision_value,
		recall_value = EXCLUDED.recall_value,
		f1_score = EXCLUDED.f1_score,
		alert_count = EXCLUDED.alert_count,
		alert_rate = EXCLUDED.alert_rate`)
	_, err := r.db.ExecContext(ctx, query.String(), args...)
	return err
}

func (r *PostgresMLOpsRepository) SaveTrainingJob(ctx context.Context, job *domain.MLTrainingJob) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ml_training_jobs (
			id, requested_version, status, requested_by, parameters, manifest,
			progress, error_message, created_at, started_at, finished_at, updated_at
		) VALUES ($1, $2, $3, NULLIF($4, ''), $5::jsonb, $6::jsonb, $7::jsonb, NULLIF($8, ''), $9, $10, $11, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			requested_version = EXCLUDED.requested_version,
			status = EXCLUDED.status,
			manifest = EXCLUDED.manifest,
			progress = EXCLUDED.progress,
			error_message = EXCLUDED.error_message,
			started_at = EXCLUDED.started_at,
			finished_at = EXCLUDED.finished_at,
			updated_at = CURRENT_TIMESTAMP
	`, job.ID, job.RequestedVersion, job.Status, job.RequestedBy,
		marshalJSON(job.Parameters, "{}"), marshalJSON(job.Manifest, "{}"), marshalJSON(job.Progress, "{}"), job.ErrorMessage,
		job.CreatedAt, job.StartedAt, job.FinishedAt)
	return err
}

func (r *PostgresMLOpsRepository) GetTrainingJob(ctx context.Context, id string) (*domain.MLTrainingJob, error) {
	job, err := scanTrainingJob(r.db.QueryRowContext(ctx, `
		SELECT id, requested_version, status, COALESCE(requested_by, ''), parameters,
		       manifest, progress, COALESCE(error_message, ''), created_at, started_at, finished_at
		FROM ml_training_jobs WHERE id = $1
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrModelNotFound
	}
	return job, err
}

func scanTrainingJob(row scanner) (*domain.MLTrainingJob, error) {
	var job domain.MLTrainingJob
	var parametersJSON, manifestJSON, progressJSON []byte
	err := row.Scan(&job.ID, &job.RequestedVersion, &job.Status, &job.RequestedBy,
		&parametersJSON, &manifestJSON, &progressJSON, &job.ErrorMessage, &job.CreatedAt, &job.StartedAt, &job.FinishedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(parametersJSON, &job.Parameters); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(manifestJSON, &job.Manifest); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(progressJSON, &job.Progress); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *PostgresMLOpsRepository) ListTrainingJobs(ctx context.Context, limit int) ([]domain.MLTrainingJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, requested_version, status, COALESCE(requested_by, ''), parameters,
		       manifest, progress, COALESCE(error_message, ''), created_at, started_at, finished_at
		FROM ml_training_jobs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := make([]domain.MLTrainingJob, 0)
	for rows.Next() {
		job, err := scanTrainingJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

func (r *PostgresMLOpsRepository) SaveDriftReport(ctx context.Context, report *domain.MLDriftReport) error {
	evaluatedAt := time.Now().UTC()
	if report.EvaluatedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, report.EvaluatedAt)
		if err != nil {
			return fmt.Errorf("data de avaliacao de drift invalida: %w", err)
		}
		evaluatedAt = parsed
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO ml_drift_reports (
			model_run_id, reference_dataset_sha256, status, data_drift_detected,
			psi_score, ks_statistic, evaluated_at, features
		)
		SELECT id, $2, $3, $4, $5, $6, $7, $8::jsonb
		FROM ml_model_runs
		WHERE version = $1
	`, report.ModelVersion, report.ReferenceDatasetSHA256, report.Status,
		report.DataDriftDetected, report.PSIScore, report.KSStatistic, evaluatedAt,
		marshalJSON(report.FeaturesDrift, "[]"))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrModelNotFound
	}
	return nil
}

func (r *PostgresMLOpsRepository) PromoteModel(ctx context.Context, version, userID string) error {
	result, err := r.db.ExecContext(ctx, `
		WITH target AS (
			SELECT id FROM ml_model_runs
			WHERE version = $1
			  AND status IN ('candidate', 'retired', 'production')
			  AND artifact_sha256 <> ''
			  AND EXISTS (SELECT 1 FROM ml_threshold_metrics tm WHERE tm.model_run_id = ml_model_runs.id)
		), retired AS (
			UPDATE ml_model_runs
			SET status = 'retired'
			WHERE status = 'production'
			  AND EXISTS (SELECT 1 FROM target)
			  AND id <> (SELECT id FROM target)
		)
		UPDATE ml_model_runs
		SET status = 'production', activated_at = CURRENT_TIMESTAMP, created_by = NULLIF($2, '')
		WHERE id = (SELECT id FROM target)
	`, version, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrModelNotFound
	}
	return nil
}
