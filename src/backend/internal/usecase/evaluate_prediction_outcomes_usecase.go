package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type EvaluatePredictionOutcomesUseCase struct {
	repository port.PredictionOutcomeRepository
}

func NewEvaluatePredictionOutcomesUseCase(repository port.PredictionOutcomeRepository) *EvaluatePredictionOutcomesUseCase {
	return &EvaluatePredictionOutcomesUseCase{repository: repository}
}

func (uc *EvaluatePredictionOutcomesUseCase) Execute(ctx context.Context) (int64, error) {
	return uc.repository.EvaluateMaturePredictions(ctx)
}

func (uc *EvaluatePredictionOutcomesUseCase) GetPerformance(ctx context.Context) (*domain.ProductionPerformance, error) {
	return uc.repository.GetProductionPerformance(ctx)
}

func RunPredictionOutcomeScheduler(ctx context.Context, uc *EvaluatePredictionOutcomesUseCase, interval time.Duration, logger *slog.Logger) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	evaluate := func() {
		runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		count, err := uc.Execute(runCtx)
		if err != nil {
			logger.Error("falha ao avaliar predicoes maduras", slog.String("error", err.Error()))
			return
		}
		logger.Info("predicoes maduras avaliadas", slog.Int64("new_outcomes", count))
	}
	evaluate()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			evaluate()
		}
	}
}
