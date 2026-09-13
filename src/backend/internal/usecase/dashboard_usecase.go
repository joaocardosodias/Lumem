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

func (uc *DashboardUseCase) TriggerPredictionsWithThreshold(ctx context.Context, horizonDays int, threshold *float64) (*PredictionRunOutputDTO, error) {
	return uc.triggerPredUC.ExecuteWithThreshold(ctx, horizonDays, threshold)
}

func (uc *DashboardUseCase) SetMLOpsRepository(repo port.MLOpsRepository) {
	if uc.triggerPredUC != nil {
		uc.triggerPredUC.SetMLOpsRepository(repo)
	}
}

func (uc *DashboardUseCase) GetLatestPredictions(ctx context.Context) (*PredictionRunOutputDTO, error) {
	return uc.latestPredUC.Execute(ctx)
}
