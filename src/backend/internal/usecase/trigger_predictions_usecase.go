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

	doorsAtRisk := 0
	for _, item := range predictedItems {
		if item.WillFail {
			doorsAtRisk++
		}
	}

	now := time.Now()
	runID := uuid.New().String()
	run := &domain.PredictionRun{
		ID:                  runID,
		HorizonDays:         horizonDays,
		TotalDoorsEvaluated: len(doors),
		TotalDoorsAtRisk:    doorsAtRisk,
		CreatedAt:           now,
	}

	for i := range predictedItems {
		predictedItems[i].ID = uuid.New().String()
		predictedItems[i].RunID = runID
	}

	err = uc.uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		if err := store.Predictions().SaveRun(ctx, run, predictedItems); err != nil {
			return err
		}
		for _, p := range predictedItems {
			if p.WillFail {
				notes := "Ordem gerada automaticamente pela IA (Rodada #" + run.ID + ")"
				predID := p.ID
				ins, err := domain.NewInspection(
					uuid.New().String(),
					&predID,
					p.StationCode,
					p.Platform,
					p.DoorNumber,
					"",
					"critical",
					notes,
				)
				if err != nil {
					return err
				}
				if err := store.Inspections().Create(ctx, ins); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	itemsDTO := make([]PredictionItemOutputDTO, 0, len(predictedItems))
	for _, p := range predictedItems {
		itemsDTO = append(itemsDTO, PredictionItemOutputDTO{
			ID:          p.ID,
			RunID:       p.RunID,
			StationCode: p.StationCode,
			Platform:    p.Platform,
			DoorNumber:  p.DoorNumber,
			WillFail:    p.WillFail,
		})

		if p.WillFail && uc.eventBus != nil {
			uc.eventBus.Publish(ctx, domain.DoorRiskDetectedEvent{
				RunID:       run.ID,
				StationCode: p.StationCode,
				Platform:    p.Platform,
				DoorNumber:  p.DoorNumber,
				Timestamp:   now,
			})
		}
	}

	if uc.eventBus != nil {
		uc.eventBus.Publish(ctx, domain.PredictionRunCompletedEvent{
			RunID:          run.ID,
			HorizonDays:    run.HorizonDays,
			TotalEvaluated: run.TotalDoorsEvaluated,
			TotalAtRisk:    run.TotalDoorsAtRisk,
			Timestamp:      now,
		})
	}

	return &PredictionRunOutputDTO{
		ID:                  run.ID,
		HorizonDays:         run.HorizonDays,
		TotalDoorsEvaluated: run.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    run.TotalDoorsAtRisk,
		CreatedAt:           now,
		Predictions:         itemsDTO,
	}, nil
}
