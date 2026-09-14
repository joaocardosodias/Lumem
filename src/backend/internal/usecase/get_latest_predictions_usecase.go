package usecase

import (
	"context"

	"github.com/g04/backend/internal/usecase/port"
)

type GetLatestPredictionsUseCase struct {
	predRepo port.PredictionRepository
}

func NewGetLatestPredictionsUseCase(predRepo port.PredictionRepository) *GetLatestPredictionsUseCase {
	return &GetLatestPredictionsUseCase{predRepo: predRepo}
}

func (uc *GetLatestPredictionsUseCase) Execute(ctx context.Context) (*PredictionRunOutputDTO, error) {
	run, err := uc.predRepo.GetLatestRun(ctx)
	if err != nil {
		return nil, err
	}
	run.AverageProbability, run.P95Probability, run.MaximumProbability = calculatePredictionMetrics(run.Predictions)
	run.EstimatedPrecision, run.EstimatedRecall, run.EstimatedF1 = calculateClassificationMetrics(run.ThresholdUsed)

	itemsDTO := make([]PredictionItemOutputDTO, 0, len(run.Predictions))
	for _, p := range run.Predictions {
		itemsDTO = append(itemsDTO, PredictionItemOutputDTO{
			ID:          p.ID,
			RunID:       p.RunID,
			StationCode: p.StationCode,
			Platform:    p.Platform,
			DoorNumber:  p.DoorNumber,
			WillFail:    p.WillFail,
			Probability: p.Probability,
		})
	}

	return &PredictionRunOutputDTO{
		ID:                  run.ID,
		HorizonDays:         run.HorizonDays,
		ThresholdUsed:       run.ThresholdUsed,
		ModelVersion:        run.ModelVersion,
		IsExperiment:        run.IsExperiment,
		PublishedAt:         run.PublishedAt,
		TotalDoorsEvaluated: run.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    run.TotalDoorsAtRisk,
		AverageProbability:  run.AverageProbability,
		P95Probability:      run.P95Probability,
		MaximumProbability:  run.MaximumProbability,
		EstimatedPrecision:  run.EstimatedPrecision,
		EstimatedRecall:     run.EstimatedRecall,
		EstimatedF1:         run.EstimatedF1,
		CreatedAt:           run.CreatedAt,
		Predictions:         itemsDTO,
	}, nil
}
