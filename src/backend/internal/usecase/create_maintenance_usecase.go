package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
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
	ID                   int64   `json:"id"`
	SAPOrderNumber       string  `json:"sap_order_number,omitempty"`
	OrderType            string  `json:"order_type"`
	StationCode          string  `json:"station_code"`
	InstallationLocation string  `json:"installation_location,omitempty"`
	StartTimestamp       string  `json:"start_timestamp"`
	EndTimestamp         string  `json:"end_timestamp,omitempty"`
	DurationHours        float64 `json:"duration_hours,omitempty"`
	Description          string  `json:"description,omitempty"`
	Symptom              string  `json:"symptom,omitempty"`
	Cause                string  `json:"cause,omitempty"`
	ActionTaken          string  `json:"action_taken,omitempty"`
	Source               string  `json:"source"`
	CreatedAt            string  `json:"created_at"`
}

type CreateMaintenanceUseCase struct {
	repo domain.MaintenanceRepository
}

func NewCreateMaintenanceUseCase(repo domain.MaintenanceRepository) *CreateMaintenanceUseCase {
	return &CreateMaintenanceUseCase{repo: repo}
}

func (uc *CreateMaintenanceUseCase) Execute(ctx context.Context, input CreateMaintenanceInputDTO) (*MaintenanceOutputDTO, error) {
	orderType := strings.ToUpper(strings.TrimSpace(input.OrderType))
	if orderType == "" {
		return nil, errors.New("o tipo da ordem (order_type) e obrigatorio")
	}

	stationCode := strings.ToUpper(strings.TrimSpace(input.StationCode))
	if stationCode == "" {
		return nil, errors.New("o codigo da estacao (station_code) e obrigatorio")
	}

	startTime := strings.TrimSpace(input.StartTimestamp)
	if startTime == "" {
		startTime = time.Now().Format("2006-01-02 15:04:05")
	}

	event := &domain.MaintenanceEvent{
		SAPOrderNumber:       strings.TrimSpace(input.SAPOrderNumber),
		OrderType:            orderType,
		StationCode:          stationCode,
		InstallationLocation: strings.TrimSpace(input.InstallationLocation),
		StartTimestamp:       startTime,
		EndTimestamp:         strings.TrimSpace(input.EndTimestamp),
		DurationHours:        input.DurationHours,
		Description:          strings.TrimSpace(input.Description),
		Symptom:              strings.TrimSpace(input.Symptom),
		Cause:                strings.TrimSpace(input.Cause),
		ActionTaken:          strings.TrimSpace(input.ActionTaken),
		Source:               "manual_form",
	}

	if err := uc.repo.Create(ctx, event); err != nil {
		return nil, err
	}

	return toOutputDTO(event), nil
}

func (uc *CreateMaintenanceUseCase) List(ctx context.Context, limit, offset int) ([]*MaintenanceOutputDTO, error) {
	events, err := uc.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*MaintenanceOutputDTO, 0, len(events))
	for _, e := range events {
		dtos = append(dtos, toOutputDTO(e))
	}
	return dtos, nil
}

func toOutputDTO(e *domain.MaintenanceEvent) *MaintenanceOutputDTO {
	createdAtStr := ""
	if !e.CreatedAt.IsZero() {
		createdAtStr = e.CreatedAt.Format("2006-01-02 15:04:05")
	}

	return &MaintenanceOutputDTO{
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
		CreatedAt:            createdAtStr,
	}
}
