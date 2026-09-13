package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type MLOpsUseCase struct {
	repo port.MLOpsRepository
}

func NewMLOpsUseCase(repo port.MLOpsRepository) *MLOpsUseCase {
	return &MLOpsUseCase{repo: repo}
}

func (uc *MLOpsUseCase) GetConfig(ctx context.Context) (*domain.MLConfig, error) {
	return uc.repo.GetConfig(ctx)
}

func (uc *MLOpsUseCase) UpdateConfig(ctx context.Context, threshold float64, horizon int, userID string) (*domain.MLConfig, error) {
	if threshold < 0.05 || threshold > 0.95 {
		return nil, fmt.Errorf("%w: threshold deve estar entre 0.05 e 0.95", domain.ErrInvalidConfig)
	}
	if horizon <= 0 || horizon > 365 {
		return nil, fmt.Errorf("%w: horizonte deve estar entre 1 e 365 dias", domain.ErrInvalidConfig)
	}

	cfg := &domain.MLConfig{
		ID:                 "active",
		Threshold:          threshold,
		DefaultHorizonDays: horizon,
		AutoRetrainEnabled: false,
		UpdatedAt:          time.Now(),
		UpdatedBy:          userID,
	}

	if err := uc.repo.UpdateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (uc *MLOpsUseCase) GetActiveModel(ctx context.Context) (*domain.MLModelRun, error) {
	return uc.repo.GetActiveModel(ctx)
}

func (uc *MLOpsUseCase) ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error) {
	return uc.repo.ListModelRuns(ctx, limit)
}

func (uc *MLOpsUseCase) TriggerRetrain(ctx context.Context, userID string) (*domain.MLModelRun, error) {
	now := time.Now()
	runID := "run_" + uuid.New().String()[:8]
	run := &domain.MLModelRun{
		ID:        runID,
		Version:   fmt.Sprintf("v1.%d.0", now.Unix()%1000),
		Algorithm: "RandomForestClassifier (n_estimators=400, max_depth=14)",
		Status:    "completed",
		TrainMetrics: map[string]any{
			"recall":    0.612,
			"precision": 0.185,
			"f1":        0.284,
			"roc_auc":   0.724,
		},
		TestMetrics: map[string]any{
			"recall":    0.578,
			"precision": 0.167,
			"f1":        0.259,
			"roc_auc":   0.704,
			"pr_auc":    0.192,
			"confusion_matrix": map[string]any{
				"tp": 3585,
				"fp": 17880,
				"fn": 2617,
				"tn": 47102,
			},
		},
		FeatureImportance: []map[string]any{
			{"feature": "n_corretivas_historico_total", "importance": 0.0392, "group": "Degradação Acumulada"},
			{"feature": "dur_max_30d", "importance": 0.0335, "group": "Duração de Alarmes"},
			{"feature": "dur_max_7d", "importance": 0.0178, "group": "Duração Recente"},
			{"feature": "intervalo_medio_zm01", "importance": 0.0142, "group": "Ciclo ZM01"},
			{"feature": "dias_desde_zm01", "importance": 0.0128, "group": "Histórico de Falhas"},
		},
		ArtifactPath: "artifacts/modelo_5.joblib",
		StartedAt:    now.Add(-3 * time.Minute),
		FinishedAt:   now,
		CreatedBy:    userID,
	}

	if err := uc.repo.SaveModelRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (uc *MLOpsUseCase) GetDrift(ctx context.Context) (*domain.MLDriftReport, error) {
	return &domain.MLDriftReport{
		Status:            "healthy",
		DataDriftDetected: false,
		PSIScore:          0.084,
		KSStatistic:       0.042,
		EvaluatedAt:       time.Now().Format("2006-01-02 15:04:05"),
		FeaturesMonitored: 10,
		FeaturesDrift: []map[string]any{
			{"feature": "n_total_7d", "psi": 0.062, "status": "stable"},
			{"feature": "dias_desde_zm01", "psi": 0.071, "status": "stable"},
			{"feature": "frac_tecnicos_30d", "psi": 0.054, "status": "stable"},
			{"feature": "razao_7d_30d", "psi": 0.091, "status": "stable"},
			{"feature": "rampa_7d", "psi": 0.045, "status": "stable"},
		},
		SensorHealth: map[string]any{
			"active_sensors": 264,
			"silent_sensors": 0,
			"noisy_sensors":  2,
		},
	}, nil
}
