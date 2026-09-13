package port

import (
	"context"

	"github.com/g04/backend/internal/domain"
)

type MLOpsRepository interface {
	GetConfig(ctx context.Context) (*domain.MLConfig, error)
	UpdateConfig(ctx context.Context, config *domain.MLConfig) error
	GetActiveModel(ctx context.Context) (*domain.MLModelRun, error)
	ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error)
	SaveModelRun(ctx context.Context, run *domain.MLModelRun) error
}
