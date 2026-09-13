package usecase

import (
	"context"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type GetDoorPredictionHistoryUseCase struct {
	predRepo port.PredictionRepository
}

func NewGetDoorPredictionHistoryUseCase(predRepo port.PredictionRepository) *GetDoorPredictionHistoryUseCase {
	return &GetDoorPredictionHistoryUseCase{predRepo: predRepo}
}

func (uc *GetDoorPredictionHistoryUseCase) Execute(ctx context.Context, doorID, limit int) ([]DoorPredictionHistoryOutputDTO, error) {
	if doorID <= 0 {
		return nil, domain.ErrInvalidDoorID
	}

	items, err := uc.predRepo.GetDoorHistory(ctx, doorID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]DoorPredictionHistoryOutputDTO, 0, len(items))
	for _, item := range items {
		result = append(result, DoorPredictionHistoryOutputDTO{
			ID:            item.ID,
			RunID:         item.RunID,
			StationCode:   item.StationCode,
			Platform:      item.Platform,
			DoorNumber:    item.DoorNumber,
			WillFail:      item.WillFail,
			Probability:   item.Probability,
			HorizonDays:   item.HorizonDays,
			ThresholdUsed: item.ThresholdUsed,
			ModelVersion:  item.ModelVersion,
			CreatedAt:     item.CreatedAt,
			WindowEndsAt:  item.CreatedAt.AddDate(0, 0, item.HorizonDays),
		})
	}
	return result, nil
}
