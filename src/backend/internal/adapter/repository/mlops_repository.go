package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	query := `
		SELECT id, threshold, default_horizon_days, auto_retrain_enabled, updated_at, COALESCE(updated_by, '')
		FROM ml_configs
		WHERE id = 'active'
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query)

	var c domain.MLConfig
	var autoRetrain int
	var updatedAt time.Time

	err := row.Scan(&c.ID, &c.Threshold, &c.DefaultHorizonDays, &autoRetrain, &updatedAt, &c.UpdatedBy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &domain.MLConfig{
				ID:                 "active",
				Threshold:          0.27,
				DefaultHorizonDays: 7,
				AutoRetrainEnabled: false,
				UpdatedAt:          time.Now(),
				UpdatedBy:          "",
			}, nil
		}
		return nil, err
	}

	c.AutoRetrainEnabled = autoRetrain == 1
	c.UpdatedAt = updatedAt
	return &c, nil
}

func (r *PostgresMLOpsRepository) UpdateConfig(ctx context.Context, config *domain.MLConfig) error {
	autoRetrain := 0
	if config.AutoRetrainEnabled {
		autoRetrain = 1
	}

	query := `
		INSERT INTO ml_configs (id, threshold, default_horizon_days, auto_retrain_enabled, updated_at, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			threshold = EXCLUDED.threshold,
			default_horizon_days = EXCLUDED.default_horizon_days,
			auto_retrain_enabled = EXCLUDED.auto_retrain_enabled,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`
	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.Threshold,
		config.DefaultHorizonDays,
		autoRetrain,
		time.Now(),
		config.UpdatedBy,
	)
	return err
}

func (r *PostgresMLOpsRepository) GetActiveModel(ctx context.Context) (*domain.MLModelRun, error) {
	query := `
		SELECT id, version, algorithm, status, train_metrics, test_metrics, feature_importance, artifact_path, started_at, finished_at, COALESCE(created_by, '')
		FROM ml_model_runs
		ORDER BY finished_at DESC
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query)

	var m domain.MLModelRun
	var trainBytes, testBytes, featBytes []byte

	err := row.Scan(
		&m.ID,
		&m.Version,
		&m.Algorithm,
		&m.Status,
		&trainBytes,
		&testBytes,
		&featBytes,
		&m.ArtifactPath,
		&m.StartedAt,
		&m.FinishedAt,
		&m.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrModelNotFound
		}
		return nil, err
	}

	_ = json.Unmarshal(trainBytes, &m.TrainMetrics)
	_ = json.Unmarshal(testBytes, &m.TestMetrics)
	_ = json.Unmarshal(featBytes, &m.FeatureImportance)

	return &m, nil
}

func (r *PostgresMLOpsRepository) ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT id, version, algorithm, status, train_metrics, test_metrics, feature_importance, artifact_path, started_at, finished_at, COALESCE(created_by, '')
		FROM ml_model_runs
		ORDER BY finished_at DESC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []domain.MLModelRun
	for rows.Next() {
		var m domain.MLModelRun
		var trainBytes, testBytes, featBytes []byte

		err := rows.Scan(
			&m.ID,
			&m.Version,
			&m.Algorithm,
			&m.Status,
			&trainBytes,
			&testBytes,
			&featBytes,
			&m.ArtifactPath,
			&m.StartedAt,
			&m.FinishedAt,
			&m.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		_ = json.Unmarshal(trainBytes, &m.TrainMetrics)
		_ = json.Unmarshal(testBytes, &m.TestMetrics)
		_ = json.Unmarshal(featBytes, &m.FeatureImportance)

		runs = append(runs, m)
	}
	return runs, rows.Err()
}

func (r *PostgresMLOpsRepository) SaveModelRun(ctx context.Context, run *domain.MLModelRun) error {
	trainBytes, _ := json.Marshal(run.TrainMetrics)
	testBytes, _ := json.Marshal(run.TestMetrics)
	featBytes, _ := json.Marshal(run.FeatureImportance)

	query := `
		INSERT INTO ml_model_runs (id, version, algorithm, status, train_metrics, test_metrics, feature_importance, artifact_path, started_at, finished_at, created_by)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7::jsonb, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.Version,
		run.Algorithm,
		run.Status,
		string(trainBytes),
		string(testBytes),
		string(featBytes),
		run.ArtifactPath,
		run.StartedAt,
		run.FinishedAt,
		run.CreatedBy,
	)
	return err
}
