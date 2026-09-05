package usecase

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type CreateMaintenanceInputDTO struct {
	SAPOrderNumber       string
	OrderType            string
	StationCode          string
	InstallationLocation string
	StartTimestamp       string
	EndTimestamp         string
	DurationHours        float64
	Description          string
	Symptom              string
	Cause                string
	ActionTaken          string
}

type MaintenanceOutputDTO struct {
	ID                   string
	SAPOrderNumber       string
	OrderType            string
	StationCode          string
	InstallationLocation string
	StartTimestamp       string
	EndTimestamp         string
	DurationHours        float64
	Description          string
	Symptom              string
	Cause                string
	ActionTaken          string
	Source               string
	CreatedAt            time.Time
}

type CreateMaintenanceUseCase struct {
	repo port.MaintenanceRepository
}

func NewCreateMaintenanceUseCase(repo port.MaintenanceRepository) *CreateMaintenanceUseCase {
	return &CreateMaintenanceUseCase{repo: repo}
}

func (uc *CreateMaintenanceUseCase) Execute(ctx context.Context, input CreateMaintenanceInputDTO) (*MaintenanceOutputDTO, error) {
	event, err := domain.NewMaintenanceEvent(domain.CreateMaintenanceParams{
		ID:                   uuid.New().String(),
		SAPOrderNumber:       input.SAPOrderNumber,
		OrderType:            input.OrderType,
		StationCode:          input.StationCode,
		InstallationLocation: input.InstallationLocation,
		StartTimestamp:       input.StartTimestamp,
		EndTimestamp:         input.EndTimestamp,
		DurationHours:        input.DurationHours,
		Description:          input.Description,
		Symptom:              input.Symptom,
		Cause:                input.Cause,
		ActionTaken:          input.ActionTaken,
		Source:               "manual_form",
	})
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, event); err != nil {
		return nil, err
	}

	return &MaintenanceOutputDTO{
		ID:                   event.ID,
		SAPOrderNumber:       event.SAPOrderNumber,
		OrderType:            event.OrderType,
		StationCode:          event.StationCode,
		InstallationLocation: event.InstallationLocation,
		StartTimestamp:       event.StartTimestamp,
		EndTimestamp:         event.EndTimestamp,
		DurationHours:        event.DurationHours,
		Description:          event.Description,
		Symptom:              event.Symptom,
		Cause:                event.Cause,
		ActionTaken:          event.ActionTaken,
		Source:               event.Source,
		CreatedAt:            event.CreatedAt,
	}, nil
}

func (uc *CreateMaintenanceUseCase) List(ctx context.Context, limit, offset int) ([]*MaintenanceOutputDTO, error) {
	events, err := uc.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*MaintenanceOutputDTO, 0, len(events))
	for _, e := range events {
		dtos = append(dtos, &MaintenanceOutputDTO{
			ID:                   e.ID,
			SAPOrderNumber:       e.SAPOrderNumber,
			OrderType:            e.OrderType,
			StationCode:          e.StationCode,
			InstallationLocation: e.InstallationLocation,
			StartTimestamp:       e.StartTimestamp,
			EndTimestamp:         e.EndTimestamp,
			DurationHours:        e.DurationHours,
			Description:          e.Description,
			Symptom:              e.Symptom,
			Cause:                e.Cause,
			ActionTaken:          e.ActionTaken,
			Source:               e.Source,
			CreatedAt:            e.CreatedAt,
		})
	}

	return dtos, nil
}
