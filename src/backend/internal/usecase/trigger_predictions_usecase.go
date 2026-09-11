package usecase

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type TriggerPredictionsUseCase struct {
	predRepo port.PredictionRepository
	mlClient port.MLClient
	eventBus port.EventBus
	uow      port.UnitOfWork
}

func NewTriggerPredictionsUseCase(
	predRepo port.PredictionRepository,
	mlClient port.MLClient,
	eventBus port.EventBus,
	uow port.UnitOfWork,
) *TriggerPredictionsUseCase {
	return &TriggerPredictionsUseCase{
		predRepo: predRepo,
		mlClient: mlClient,
		eventBus: eventBus,
		uow:      uow,
	}
}

func countDoorsAtRisk(items []domain.PredictionItem) int {
	doorsAtRisk := 0
	for _, item := range items {
		if item.WillFail {
			doorsAtRisk++
		}
	}
	return doorsAtRisk
}

func assignItemRunIDs(items []domain.PredictionItem, runID string) {
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].RunID = runID
	}
}

func createAutomaticInspection(ctx context.Context, store port.UnitOfWorkStore, p domain.PredictionItem, runID string) error {
	predID := p.ID
	ins, err := domain.NewInspection(domain.CreateInspectionParams{
		ID:           uuid.New().String(),
		PredictionID: &predID,
		StationCode:  p.StationCode,
		Platform:     p.Platform,
		DoorNumber:   p.DoorNumber,
		Notes:        "",
	})
	if err != nil {
		return err
	}
	return store.Inspections().Create(ctx, ins)
}

func (uc *TriggerPredictionsUseCase) saveRunAndInspections(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	return uc.uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		if err := store.Predictions().SaveRun(ctx, run, items); err != nil {
			return err
		}
		for _, p := range items {
			if p.WillFail {
				if err := createAutomaticInspection(ctx, store, p, run.ID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (uc *TriggerPredictionsUseCase) publishPredictionEvents(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem, timestamp time.Time) {
	if uc.eventBus == nil {
		return
	}

	for _, p := range items {
		if p.WillFail {
			uc.eventBus.Publish(ctx, domain.DoorRiskDetectedEvent{
				RunID:       run.ID,
				StationCode: p.StationCode,
				Platform:    p.Platform,
				DoorNumber:  p.DoorNumber,
				Timestamp:   timestamp,
			})
		}
	}

	uc.eventBus.Publish(ctx, domain.PredictionRunCompletedEvent{
		RunID:          run.ID,
		HorizonDays:    run.HorizonDays,
		TotalEvaluated: run.TotalDoorsEvaluated,
		TotalAtRisk:    run.TotalDoorsAtRisk,
		Timestamp:      timestamp,
	})
}

func toPredictionItemDTOs(items []domain.PredictionItem) []PredictionItemOutputDTO {
	itemsDTO := make([]PredictionItemOutputDTO, 0, len(items))
	for _, p := range items {
		itemsDTO = append(itemsDTO, PredictionItemOutputDTO{
			ID:          p.ID,
			RunID:       p.RunID,
			StationCode: p.StationCode,
			Platform:    p.Platform,
			DoorNumber:  p.DoorNumber,
			WillFail:    p.WillFail,
		})
	}
	return itemsDTO
}

func (uc *TriggerPredictionsUseCase) Execute(ctx context.Context, horizonDays int) (*PredictionRunOutputDTO, error) {
	if horizonDays <= 0 {
		horizonDays = 7
	}

	doors, err := uc.predRepo.ListDoorsForInference(ctx)
	if err != nil {
		return nil, err
	}

	if len(doors) == 0 {
		return nil, domain.ErrNoActiveDoors
	}

	predictedItems, err := uc.mlClient.PredictBatch(ctx, doors)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	runID := uuid.New().String()
	run := &domain.PredictionRun{
		ID:                  runID,
		HorizonDays:         horizonDays,
		TotalDoorsEvaluated: len(doors),
		TotalDoorsAtRisk:    countDoorsAtRisk(predictedItems),
		CreatedAt:           now,
	}

	assignItemRunIDs(predictedItems, runID)

	if err := uc.saveRunAndInspections(ctx, run, predictedItems); err != nil {
		return nil, err
	}

	uc.publishPredictionEvents(ctx, run, predictedItems, now)

	return &PredictionRunOutputDTO{
		ID:                  run.ID,
		HorizonDays:         run.HorizonDays,
		TotalDoorsEvaluated: run.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    run.TotalDoorsAtRisk,
		CreatedAt:           now,
		Predictions:         toPredictionItemDTOs(predictedItems),
	}, nil
}
