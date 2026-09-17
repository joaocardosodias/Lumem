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
	config       *domain.MLConfig
	runs         []domain.MLModelRun
	trainingJobs map[string]domain.MLTrainingJob
	driftReports []domain.MLDriftReport
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

func (m *mockMLOpsRepo) GetModelByVersion(ctx context.Context, version string) (*domain.MLModelRun, error) {
	for index := range m.runs {
		if m.runs[index].Version == version {
			return &m.runs[index], nil
		}
	}
	return nil, domain.ErrModelNotFound
}

func (m *mockMLOpsRepo) ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error) {
	return m.runs, nil
}

func (m *mockMLOpsRepo) SaveModelRun(ctx context.Context, run *domain.MLModelRun) error {
	m.runs = append([]domain.MLModelRun{*run}, m.runs...)
	return nil
}

func (m *mockMLOpsRepo) GetThresholdMetrics(ctx context.Context, version string) ([]domain.ThresholdMetric, error) {
	return []domain.ThresholdMetric{}, nil
}

func (m *mockMLOpsRepo) GetThresholdMetric(ctx context.Context, version string, threshold float64) (*domain.ThresholdMetric, error) {
	return &domain.ThresholdMetric{Threshold: threshold}, nil
}

func (m *mockMLOpsRepo) SaveThresholdMetrics(ctx context.Context, modelRunID string, metrics []domain.ThresholdMetric) error {
	return nil
}

func (m *mockMLOpsRepo) SaveTrainingJob(ctx context.Context, job *domain.MLTrainingJob) error {
	if m.trainingJobs == nil {
		m.trainingJobs = make(map[string]domain.MLTrainingJob)
	}
	m.trainingJobs[job.ID] = *job
	return nil
}

func (m *mockMLOpsRepo) GetTrainingJob(ctx context.Context, id string) (*domain.MLTrainingJob, error) {
	job, ok := m.trainingJobs[id]
	if !ok {
		return nil, domain.ErrModelNotFound
	}
	return &job, nil
}

func (m *mockMLOpsRepo) ListTrainingJobs(ctx context.Context, limit int) ([]domain.MLTrainingJob, error) {
	jobs := make([]domain.MLTrainingJob, 0, len(m.trainingJobs))
	for _, job := range m.trainingJobs {
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (m *mockMLOpsRepo) SaveDriftReport(ctx context.Context, report *domain.MLDriftReport) error {
	m.driftReports = append(m.driftReports, *report)
	return nil
}

func (m *mockMLOpsRepo) PromoteModel(ctx context.Context, version, userID string) error {
	return nil
}

type mockMLOpsClient struct {
	activationCalls []string
}

func (m *mockMLOpsClient) GetActiveModelMetadata(ctx context.Context) (map[string]any, error) {
	return nil, domain.ErrPredictionServiceUnavailable
}

func (m *mockMLOpsClient) StartTraining(ctx context.Context, requestedBy string) (*domain.MLTrainingJob, error) {
	return &domain.MLTrainingJob{ID: "job-1", RequestedVersion: "v2", Status: "queued", CreatedAt: time.Now()}, nil
}

func (m *mockMLOpsClient) ResumeTraining(ctx context.Context, id, requestedBy string) (*domain.MLTrainingJob, error) {
	return &domain.MLTrainingJob{
		ID:               "job-resumed",
		RequestedVersion: "v2",
		Status:           "queued",
		RequestedBy:      requestedBy,
		CreatedAt:        time.Now(),
		Parameters: map[string]any{
			"resumed_from_job_id": id,
		},
	}, nil
}

func (m *mockMLOpsClient) GetTrainingJob(ctx context.Context, id string) (*domain.MLTrainingJob, error) {
	return &domain.MLTrainingJob{ID: id, RequestedVersion: "v2", Status: "running", CreatedAt: time.Now()}, nil
}

func (m *mockMLOpsClient) GetDrift(ctx context.Context, version string) (*domain.MLDriftReport, error) {
	detected := false
	psi := 0.02
	ks := 0.03
	return &domain.MLDriftReport{Status: "healthy", ModelVersion: version, DataDriftDetected: &detected, PSIScore: &psi, KSStatistic: &ks}, nil
}

func (m *mockMLOpsClient) ActivateModel(ctx context.Context, version string) error {
	m.activationCalls = append(m.activationCalls, version)
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

	updated, err := uc.UpdateConfig(context.Background(), 0.32, 7, "usr_engineer_01")
	if err != nil {
		t.Fatalf("esperava sucesso ao atualizar config: %v", err)
	}
	if updated.Threshold != 0.32 || updated.DefaultHorizonDays != 7 {
		t.Errorf("valores atualizados incorretos: %+v", updated)
	}

	_, err = uc.UpdateConfig(context.Background(), 0.01, 7, "usr_engineer_01")
	if err == nil {
		t.Errorf("esperava erro para threshold fora dos limites")
	}

	_, err = uc.UpdateConfig(context.Background(), 0.30, 14, "usr_engineer_01")
	if err == nil {
		t.Errorf("esperava erro para horizonte invalido")
	}
}

func TestMLOpsUseCase_TriggerRetrainAndDrift(t *testing.T) {
	repo := &mockMLOpsRepo{}
	uc := usecase.NewMLOpsUseCase(repo, &mockMLOpsClient{})

	run, err := uc.TriggerRetrain(context.Background(), "usr_engineer_01")
	if err != nil {
		t.Fatalf("esperava sucesso ao disparar retreino: %v", err)
	}
	if run.Status != "queued" {
		t.Errorf("esperava status queued, obteve %s", run.Status)
	}

	current, err := uc.GetTrainingJob(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("esperava sucesso ao consultar job: %v", err)
	}
	if current.Status != "running" {
		t.Errorf("esperava status running, obteve %s", current.Status)
	}

	drift, err := uc.GetDrift(context.Background())
	if err != nil {
		t.Fatalf("esperava sucesso ao obter drift: %v", err)
	}
	if drift.Status != "healthy" {
		t.Errorf("esperava status healthy, obteve %s", drift.Status)
	}
	if len(repo.driftReports) != 1 || repo.driftReports[0].ModelVersion != "v1.0.0" {
		t.Fatalf("esperava auditoria persistida para v1.0.0, obteve %+v", repo.driftReports)
	}
}

func TestMLOpsUseCase_ResumeTrainingFromFailedStage(t *testing.T) {
	repo := &mockMLOpsRepo{trainingJobs: map[string]domain.MLTrainingJob{
		"job-failed": {
			ID:               "job-failed",
			RequestedVersion: "v2",
			Status:           "failed",
			CreatedAt:        time.Now(),
			Progress: domain.MLTrainingProgress{Stages: []domain.MLTrainingStage{
				{Key: "snapshot_acquisition", Status: "completed"},
				{Key: "data_loading", Status: "failed"},
			}},
		},
	}}
	uc := usecase.NewMLOpsUseCase(repo, &mockMLOpsClient{})

	resumed, err := uc.ResumeTraining(context.Background(), "job-failed", "usr_engineer_01")
	if err != nil {
		t.Fatalf("esperava sucesso ao retomar treinamento: %v", err)
	}
	if resumed.Status != "queued" || resumed.Parameters["resumed_from_job_id"] != "job-failed" {
		t.Fatalf("job retomado incorreto: %+v", resumed)
	}
}

func TestMLOpsUseCase_PromoteValidatesCandidateBeforeExternalActivation(t *testing.T) {
	repo := &mockMLOpsRepo{runs: []domain.MLModelRun{{
		ID: "run-active", Version: "v1", Status: "production",
	}}}
	client := &mockMLOpsClient{}
	uc := usecase.NewMLOpsUseCase(repo, client)

	_, err := uc.PromoteModel(context.Background(), "v-inexistente", "usr_engineer_01")
	if err != domain.ErrModelNotFound {
		t.Fatalf("esperava ErrModelNotFound, obteve %v", err)
	}
	if len(client.activationCalls) != 0 {
		t.Fatalf("ativacao externa ocorreu antes da validacao: %v", client.activationCalls)
	}
}
