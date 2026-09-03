package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
)

type DashboardUseCase struct {
	dashRepo domain.DashboardRepository
	predRepo domain.PredictionRepository
	mlClient domain.MLClient
}

func NewDashboardUseCase(
	dashRepo domain.DashboardRepository,
	predRepo domain.PredictionRepository,
	mlClient domain.MLClient,
) *DashboardUseCase {
	return &DashboardUseCase{
		dashRepo: dashRepo,
		predRepo: predRepo,
		mlClient: mlClient,
	}
}

func (uc *DashboardUseCase) GetSummary(ctx context.Context) (*DashboardSummaryOutputDTO, error) {
	s, err := uc.dashRepo.GetSummary(ctx)
	if err != nil {
		return nil, err
	}

	critical := make([]StationRiskOutputDTO, 0, len(s.CriticalStations))
	for _, c := range s.CriticalStations {
		critical = append(critical, StationRiskOutputDTO{
			StationCode: c.StationCode,
			DoorsAtRisk: c.DoorsAtRisk,
		})
	}

	return &DashboardSummaryOutputDTO{
		TotalDoors:           s.TotalDoors,
		DoorsAtRisk:          s.DoorsAtRisk,
		PendingInspections:   s.PendingInspections,
		CompletedInspections: s.CompletedInspections,
		CriticalStations:     critical,
	}, nil
}

func (uc *DashboardUseCase) GetDoorTimeline(ctx context.Context, doorID int) (*DoorTimelineOutputDTO, error) {
	if doorID <= 0 {
		return nil, errors.New("id da porta invalido")
	}

	t, err := uc.dashRepo.GetDoorTimeline(ctx, doorID)
	if err != nil {
		return nil, err
	}

	var evalStr *string
	if t.LastEvaluatedAt != nil {
		s := t.LastEvaluatedAt.Format("2006-01-02 15:04:05")
		evalStr = &s
	}

	alarms := make([]TimelineAlarmOutputDTO, 0, len(t.RecentAlarms))
	for _, a := range t.RecentAlarms {
		alarms = append(alarms, TimelineAlarmOutputDTO{
			ID:                     a.ID,
			Tag:                    a.Tag,
			AlarmType:              a.AlarmType,
			DurationSeconds:        a.DurationSeconds,
			EventTimestamp:         a.EventTimestamp,
			NormalizationTimestamp: a.NormalizationTimestamp,
		})
	}

	maints := make([]TimelineMaintenanceOutputDTO, 0, len(t.Maintenances))
	for _, m := range t.Maintenances {
		maints = append(maints, TimelineMaintenanceOutputDTO{
			ID:             m.ID,
			OrderType:      m.OrderType,
			SAPOrderNumber: m.SAPOrderNumber,
			StartTimestamp: m.StartTimestamp,
			DurationHours:  m.DurationHours,
			Description:    m.Description,
			Symptom:        m.Symptom,
			Source:         m.Source,
		})
	}

	inspections := make([]TimelineInspectionOutputDTO, 0, len(t.Inspections))
	for _, i := range t.Inspections {
		var compStr *string
		if i.CompletedAt != nil {
			s := i.CompletedAt.Format("2006-01-02 15:04:05")
			compStr = &s
		}
		inspections = append(inspections, TimelineInspectionOutputDTO{
			ID:           i.ID,
			Status:       i.Status,
			Priority:     i.Priority,
			Notes:        i.Notes,
			ScheduledFor: i.ScheduledFor,
			CompletedAt:  compStr,
		})
	}

	return &DoorTimelineOutputDTO{
		DoorID:          t.Door.ID,
		StationName:     t.StationName,
		Platform:        t.Door.Platform,
		DoorNumber:      t.Door.DoorNumber,
		SAPTag:          t.Door.SAPTag,
		Status:          t.Door.Status,
		WillFail:        t.WillFail,
		LastEvaluatedAt: evalStr,
		RecentAlarms:    alarms,
		Maintenances:    maints,
		Inspections:     inspections,
	}, nil
}

func (uc *DashboardUseCase) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]DoorStatusOutputDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(stationCode))
	if code == "" {
		return nil, errors.New("codigo da estacao e obrigatorio")
	}

	items, err := uc.dashRepo.GetStationDoorsStatus(ctx, code)
	if err != nil {
		return nil, err
	}

	dtos := make([]DoorStatusOutputDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, DoorStatusOutputDTO{
			ID:         item.ID,
			Platform:   item.Platform,
			DoorNumber: item.DoorNumber,
			SAPTag:     item.SAPTag,
			Status:     item.Status,
			WillFail:   item.WillFail,
		})
	}
	return dtos, nil
}

func (uc *DashboardUseCase) TriggerPredictions(ctx context.Context, horizonDays int) (*PredictionRunOutputDTO, error) {
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

	itemsDTO := make([]PredictionItemOutputDTO, 0, len(predictedItems))
	for _, p := range predictedItems {
		itemsDTO = append(itemsDTO, PredictionItemOutputDTO{
			ID:          p.ID,
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
		CreatedAt:           now.Format("2006-01-02 15:04:05"),
		Predictions:         itemsDTO,
	}, nil
}

func (uc *DashboardUseCase) GetLatestPredictions(ctx context.Context) (*PredictionRunOutputDTO, error) {
	run, err := uc.predRepo.GetLatestRun(ctx)
	if err != nil {
		return nil, err
	}

	itemsDTO := make([]PredictionItemOutputDTO, 0, len(run.Predictions))
	for _, p := range run.Predictions {
		itemsDTO = append(itemsDTO, PredictionItemOutputDTO{
			ID:          p.ID,
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
		CreatedAt:           run.CreatedAt.Format("2006-01-02 15:04:05"),
		Predictions:         itemsDTO,
	}, nil
}
