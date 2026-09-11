package usecase

import (
	"context"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type GetDoorTimelineUseCase struct {
	dashRepo port.DashboardRepository
}

func NewGetDoorTimelineUseCase(dashRepo port.DashboardRepository) *GetDoorTimelineUseCase {
	return &GetDoorTimelineUseCase{dashRepo: dashRepo}
}

func (uc *GetDoorTimelineUseCase) Execute(ctx context.Context, doorID int) (*DoorTimelineOutputDTO, error) {
	if doorID <= 0 {
		return nil, domain.ErrInvalidDoorID
	}

	t, err := uc.dashRepo.GetDoorTimeline(ctx, doorID)
	if err != nil {
		return nil, err
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
		inspections = append(inspections, TimelineInspectionOutputDTO{
			ID:          i.ID,
			Status:      i.Status,
			Notes:       i.Notes,
			CompletedAt: i.CompletedAt,
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
		LastEvaluatedAt: t.LastEvaluatedAt,
		RecentAlarms:    alarms,
		Maintenances:    maints,
		Inspections:     inspections,
	}, nil
}
