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

	itemsDTO := make([]PredictionItemOutputDTO, 0, len(run.Predictions))
	for _, p := range run.Predictions {
		itemsDTO = append(itemsDTO, PredictionItemOutputDTO{
			ID:          p.ID,
			RunID:       p.RunID,
			StationCode: p.StationCode,
			Platform:    p.Platform,
			DoorNumber:  p.DoorNumber,
			WillFail:    p.WillFail,
		})
	}

	return &PredictionRunOutputDTO{
		ID:                  run.ID,
		HorizonDays:         run.HorizonDays,
		TotalDoorsEvaluated: run.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    run.TotalDoorsAtRisk,
		CreatedAt:           run.CreatedAt,
		Predictions:         itemsDTO,
	}, nil
}
