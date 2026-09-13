package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

var _ port.MLOpsRepository = (*mockMLOpsRepo)(nil)

type mockMLOpsRepo struct {
	config *domain.MLConfig
	runs   []domain.MLModelRun
}

func (m *mockMLOpsRepo) GetConfig(ctx context.Context) (*domain.MLConfig, error) {
	if m.config == nil {
		return &domain.MLConfig{
			ID:                 "active",
			Threshold:          0.27,
			DefaultHorizonDays: 7,
			AutoRetrainEnabled: false,
			UpdatedAt:          time.Now(),
		}, nil
	}
	return m.config, nil
}

func (m *mockMLOpsRepo) UpdateConfig(ctx context.Context, config *domain.MLConfig) error {
	m.config = config
	return nil
}

func (m *mockMLOpsRepo) GetActiveModel(ctx context.Context) (*domain.MLModelRun, error) {
	if len(m.runs) == 0 {
		return &domain.MLModelRun{
			ID:        "run_1",
			Version:   "v1.0.0",
			Algorithm: "RandomForestClassifier",
		}, nil
	}
	return &m.runs[0], nil
}

func (m *mockMLOpsRepo) ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error) {
	return m.runs, nil
}

func (m *mockMLOpsRepo) SaveModelRun(ctx context.Context, run *domain.MLModelRun) error {
	m.runs = append([]domain.MLModelRun{*run}, m.runs...)
	return nil
}

func TestMLOpsUseCase_GetAndBuildConfig(t *testing.T) {
	repo := &mockMLOpsRepo{}
	uc := usecase.NewMLOpsUseCase(repo)

	cfg, err := uc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("esperava sucesso ao obter config: %v", err)
	}
	if cfg.Threshold != 0.27 {
		t.Errorf("esperava threshold 0.27, obteve %f", cfg.Threshold)
	}

	updated, err := uc.UpdateConfig(context.Background(), 0.32, 14, "usr_engineer_01")
	if err != nil {
		t.Fatalf("esperava sucesso ao atualizar config: %v", err)
	}
	if updated.Threshold != 0.32 || updated.DefaultHorizonDays != 14 {
		t.Errorf("valores atualizados incorretos: %+v", updated)
	}

	_, err = uc.UpdateConfig(context.Background(), 0.01, 14, "usr_engineer_01")
	if err == nil {
		t.Errorf("esperava erro para threshold fora dos limites")
	}

	_, err = uc.UpdateConfig(context.Background(), 0.30, 0, "usr_engineer_01")
	if err == nil {
		t.Errorf("esperava erro para horizonte invalido")
	}
}

func TestMLOpsUseCase_TriggerRetrainAndDrift(t *testing.T) {
	repo := &mockMLOpsRepo{}
	uc := usecase.NewMLOpsUseCase(repo)

	run, err := uc.TriggerRetrain(context.Background(), "usr_engineer_01")
	if err != nil {
		t.Fatalf("esperava sucesso ao disparar retreino: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("esperava status completed, obteve %s", run.Status)
	}

	runs, err := uc.ListModelRuns(context.Background(), 5)
	if err != nil {
		t.Fatalf("esperava sucesso ao listar runs: %v", err)
	}
	if len(runs) == 0 {
		t.Errorf("esperava pelo menos 1 run listado")
	}

	drift, err := uc.GetDrift(context.Background())
	if err != nil {
		t.Fatalf("esperava sucesso ao obter drift: %v", err)
	}
	if drift.Status != "healthy" {
		t.Errorf("esperava status healthy, obteve %s", drift.Status)
	}
}
