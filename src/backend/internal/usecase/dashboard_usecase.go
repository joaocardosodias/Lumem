package usecase

import (
	"context"

	"github.com/g04/backend/internal/usecase/port"
)

type DashboardUseCase struct {
	summaryUC     *GetDashboardSummaryUseCase
	timelineUC    *GetDoorTimelineUseCase
	statusUC      *GetStationDoorsStatusUseCase
	triggerPredUC *TriggerPredictionsUseCase
	latestPredUC  *GetLatestPredictionsUseCase
	historyPredUC *GetDoorPredictionHistoryUseCase
}

func NewDashboardUseCase(
	dashRepo port.DashboardRepository,
	predRepo port.PredictionRepository,
	mlClient port.MLClient,
	eventBus port.EventBus,
	uow port.UnitOfWork,
) *DashboardUseCase {
	return &DashboardUseCase{
		summaryUC:     NewGetDashboardSummaryUseCase(dashRepo),
		timelineUC:    NewGetDoorTimelineUseCase(dashRepo),
		statusUC:      NewGetStationDoorsStatusUseCase(dashRepo),
		triggerPredUC: NewTriggerPredictionsUseCase(predRepo, mlClient, eventBus, uow),
		latestPredUC:  NewGetLatestPredictionsUseCase(predRepo),
		historyPredUC: NewGetDoorPredictionHistoryUseCase(predRepo),
	}
}

func (uc *DashboardUseCase) GetSummary(ctx context.Context) (*DashboardSummaryOutputDTO, error) {
	return uc.summaryUC.Execute(ctx)
}

func (uc *DashboardUseCase) GetDoorTimeline(ctx context.Context, doorID int) (*DoorTimelineOutputDTO, error) {
	return uc.timelineUC.Execute(ctx, doorID)
}

func (uc *DashboardUseCase) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]DoorStatusOutputDTO, error) {
	return uc.statusUC.Execute(ctx, stationCode)
}

func (uc *DashboardUseCase) TriggerPredictions(ctx context.Context, horizonDays int) (*PredictionRunOutputDTO, error) {
	return uc.triggerPredUC.Execute(ctx, horizonDays)
}

func (uc *DashboardUseCase) TriggerPredictionsForModel(ctx context.Context, horizonDays int, modelVersion string) (*PredictionRunOutputDTO, error) {
	return uc.triggerPredUC.ExecuteForModel(ctx, horizonDays, modelVersion)
}

func (uc *DashboardUseCase) SetMLOpsRepository(repo port.MLOpsRepository) {
	if uc.triggerPredUC != nil {
		uc.triggerPredUC.SetMLOpsRepository(repo)
	}
}

func (uc *DashboardUseCase) GetLatestPredictions(ctx context.Context) (*PredictionRunOutputDTO, error) {
	return uc.latestPredUC.Execute(ctx)
}

func (uc *DashboardUseCase) GetDoorPredictionHistory(ctx context.Context, doorID, limit int) ([]DoorPredictionHistoryOutputDTO, error) {
	return uc.historyPredUC.Execute(ctx, doorID, limit)
}

func (uc *DashboardUseCase) RunPredictionExperiment(ctx context.Context, threshold float64) (*PredictionRunOutputDTO, error) {
	return uc.triggerPredUC.ExecuteExperiment(ctx, threshold)
}

func (uc *DashboardUseCase) ListPredictionExperiments(ctx context.Context, limit int) ([]PredictionRunOutputDTO, error) {
	return uc.triggerPredUC.ListExperiments(ctx, limit)
}

func (uc *DashboardUseCase) PromotePredictionExperiment(ctx context.Context, id, userID string) (*PredictionRunOutputDTO, error) {
	return uc.triggerPredUC.PromoteExperiment(ctx, id, userID)
}
