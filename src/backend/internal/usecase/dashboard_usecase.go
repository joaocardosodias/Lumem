package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/domain"
)

type DashboardUseCase struct {
	dashRepo domain.DashboardRepository
	predRepo domain.PredictionRepository
	mlClient *mlclient.Client
}

func NewDashboardUseCase(
	dashRepo domain.DashboardRepository,
	predRepo domain.PredictionRepository,
	mlClient *mlclient.Client,
) *DashboardUseCase {
	return &DashboardUseCase{
		dashRepo: dashRepo,
		predRepo: predRepo,
		mlClient: mlClient,
	}
}

func (uc *DashboardUseCase) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	return uc.dashRepo.GetSummary(ctx)
}

func (uc *DashboardUseCase) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	if doorID <= 0 {
		return nil, errors.New("id da porta invalido")
	}
	return uc.dashRepo.GetDoorTimeline(ctx, doorID)
}

func (uc *DashboardUseCase) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	code := strings.ToUpper(strings.TrimSpace(stationCode))
	if code == "" {
		return nil, errors.New("codigo da estacao e obrigatorio")
	}
	return uc.dashRepo.GetStationDoorsStatus(ctx, code)
}

func (uc *DashboardUseCase) TriggerPredictions(ctx context.Context, horizonDays int) (*domain.PredictionRun, error) {
	if horizonDays <= 0 {
		horizonDays = 7
	}

	doors, err := uc.predRepo.ListDoorsForInference(ctx)
	if err != nil {
		return nil, err
	}

	if len(doors) == 0 {
		return nil, errors.New("nenhuma porta ativa encontrada para inferencia")
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
	run := &domain.PredictionRun{
		HorizonDays:         horizonDays,
		TotalDoorsEvaluated: len(doors),
		TotalDoorsAtRisk:    doorsAtRisk,
		CreatedAt:           now,
	}

	if err := uc.predRepo.SaveRun(ctx, run, predictedItems); err != nil {
		return nil, err
	}

	run.Predictions = predictedItems
	return run, nil
}

func (uc *DashboardUseCase) GetLatestPredictions(ctx context.Context) (*domain.PredictionRun, error) {
	return uc.predRepo.GetLatestRun(ctx)
}
