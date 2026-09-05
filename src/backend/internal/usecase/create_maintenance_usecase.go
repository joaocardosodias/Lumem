package usecase

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
)

type CreateMaintenanceInputDTO struct {
	SAPOrderNumber       string  `json:"sap_order_number"`
	OrderType            string  `json:"order_type"`
	StationCode          string  `json:"station_code"`
	InstallationLocation string  `json:"installation_location"`
	StartTimestamp       string  `json:"start_timestamp"`
	EndTimestamp         string  `json:"end_timestamp"`
	DurationHours        float64 `json:"duration_hours"`
	Description          string  `json:"description"`
	Symptom              string  `json:"symptom"`
	Cause                string  `json:"cause"`
	ActionTaken          string  `json:"action_taken"`
}

type MaintenanceOutputDTO struct {
	ID                   string    `json:"id"`
	SAPOrderNumber       string    `json:"sap_order_number,omitempty"`
	OrderType            string    `json:"order_type"`
	StationCode          string    `json:"station_code"`
	InstallationLocation string    `json:"installation_location,omitempty"`
	StartTimestamp       string    `json:"start_timestamp"`
	EndTimestamp         string    `json:"end_timestamp,omitempty"`
	DurationHours        float64   `json:"duration_hours,omitempty"`
	Description          string    `json:"description,omitempty"`
	Symptom              string    `json:"symptom,omitempty"`
	Cause                string    `json:"cause,omitempty"`
	ActionTaken          string    `json:"action_taken,omitempty"`
	Source               string    `json:"source"`
	CreatedAt            time.Time `json:"created_at"`
}

type CreateMaintenanceUseCase struct {
	repo domain.MaintenanceRepository
}

func NewCreateMaintenanceUseCase(repo domain.MaintenanceRepository) *CreateMaintenanceUseCase {
	return &CreateMaintenanceUseCase{repo: repo}
}

func (uc *CreateMaintenanceUseCase) Execute(ctx context.Context, input CreateMaintenanceInputDTO) (*MaintenanceOutputDTO, error) {
	event, err := domain.NewMaintenanceEvent(
		uuid.New().String(),
		input.SAPOrderNumber,
		input.OrderType,
		input.StationCode,
		input.InstallationLocation,
		input.StartTimestamp,
		input.EndTimestamp,
		input.DurationHours,
		input.Description,
		input.Symptom,
		input.Cause,
		input.ActionTaken,
		"manual_form",
	)
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
