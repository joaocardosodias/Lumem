package port

import (
	"context"

	"github.com/g04/backend/internal/domain"
)

type MLOpsRepository interface {
	GetConfig(ctx context.Context) (*domain.MLConfig, error)
	UpdateConfig(ctx context.Context, config *domain.MLConfig) error
	GetActiveModel(ctx context.Context) (*domain.MLModelRun, error)
	GetModelByVersion(ctx context.Context, version string) (*domain.MLModelRun, error)
	ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error)
	SaveModelRun(ctx context.Context, run *domain.MLModelRun) error
	GetThresholdMetrics(ctx context.Context, version string) ([]domain.ThresholdMetric, error)
	GetThresholdMetric(ctx context.Context, version string, threshold float64) (*domain.ThresholdMetric, error)
	SaveThresholdMetrics(ctx context.Context, modelRunID string, metrics []domain.ThresholdMetric) error
	SaveTrainingJob(ctx context.Context, job *domain.MLTrainingJob) error
	GetTrainingJob(ctx context.Context, id string) (*domain.MLTrainingJob, error)
	ListTrainingJobs(ctx context.Context, limit int) ([]domain.MLTrainingJob, error)
	SaveDriftReport(ctx context.Context, report *domain.MLDriftReport) error
	PromoteModel(ctx context.Context, version, userID string) error
}
