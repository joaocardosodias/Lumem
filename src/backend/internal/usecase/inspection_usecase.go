package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
)

type CreateInspectionInputDTO struct {
	PredictionID *int64  `json:"prediction_id"`
	StationCode  string  `json:"station_code"`
	Platform     string  `json:"platform"`
	DoorNumber   int     `json:"door_number"`
	Priority     string  `json:"priority"`
	AssignedTo   *string `json:"assigned_to"`
	Notes        string  `json:"notes"`
	ScheduledFor string  `json:"scheduled_for"`
}

type UpdateInspectionStatusInputDTO struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

type InspectionOutputDTO struct {
	ID           int64   `json:"id"`
	PredictionID *int64  `json:"prediction_id,omitempty"`
	StationCode  string  `json:"station_code"`
	Platform     string  `json:"platform"`
	DoorNumber   int     `json:"door_number"`
	Status       string  `json:"status"`
	Priority     string  `json:"priority"`
	AssignedTo   *string `json:"assigned_to,omitempty"`
	Notes        string  `json:"notes,omitempty"`
	ScheduledFor string  `json:"scheduled_for,omitempty"`
	CompletedAt  *string `json:"completed_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type InspectionUseCase struct {
	repo domain.InspectionRepository
}

func NewInspectionUseCase(repo domain.InspectionRepository) *InspectionUseCase {
	return &InspectionUseCase{repo: repo}
}

func (uc *InspectionUseCase) Create(ctx context.Context, input CreateInspectionInputDTO) (*InspectionOutputDTO, error) {
	stationCode := strings.ToUpper(strings.TrimSpace(input.StationCode))
	if stationCode == "" {
		return nil, errors.New("o codigo da estacao (station_code) e obrigatorio")
	}

	platform := strings.ToUpper(strings.TrimSpace(input.Platform))
	if platform == "" {
		platform = "P1"
	}

	if input.DoorNumber <= 0 || input.DoorNumber > 24 {
		return nil, errors.New("o numero da porta deve estar entre 1 e 24")
	}

	priority := strings.ToLower(strings.TrimSpace(input.Priority))
	if priority == "" {
		priority = "high"
	}

	ins := &domain.Inspection{
		PredictionID: input.PredictionID,
		StationCode:  stationCode,
		Platform:     platform,
		DoorNumber:   input.DoorNumber,
		Status:       "pending",
		Priority:     priority,
		AssignedTo:   input.AssignedTo,
		Notes:        strings.TrimSpace(input.Notes),
		ScheduledFor: strings.TrimSpace(input.ScheduledFor),
	}

	if err := uc.repo.Create(ctx, ins); err != nil {
		return nil, err
	}

	return toInspectionDTO(ins), nil
}

func (uc *InspectionUseCase) UpdateStatus(ctx context.Context, id int64, input UpdateInspectionStatusInputDTO) (*InspectionOutputDTO, error) {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != "pending" && status != "in_progress" && status != "completed" && status != "canceled" {
		return nil, errors.New("status invalido (valores permitidos: pending, in_progress, completed, canceled)")
	}

	var completedAt *time.Time
	if status == "completed" {
		now := time.Now()
		completedAt = &now
	}

	if err := uc.repo.UpdateStatus(ctx, id, status, strings.TrimSpace(input.Notes), completedAt); err != nil {
		return nil, err
	}

	ins, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return toInspectionDTO(ins), nil
}

func (uc *InspectionUseCase) List(ctx context.Context, limit, offset int, status string) ([]*InspectionOutputDTO, error) {
	items, err := uc.repo.List(ctx, limit, offset, strings.TrimSpace(status))
	if err != nil {
		return nil, err
	}

	dtos := make([]*InspectionOutputDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, toInspectionDTO(item))
	}
	return dtos, nil
}

func (uc *InspectionUseCase) GetByID(ctx context.Context, id int64) (*InspectionOutputDTO, error) {
	item, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toInspectionDTO(item), nil
}

func toInspectionDTO(i *domain.Inspection) *InspectionOutputDTO {
	var completedStr *string
	if i.CompletedAt != nil {
		s := i.CompletedAt.Format("2006-01-02 15:04:05")
		completedStr = &s
	}

	return &InspectionOutputDTO{
		ID:           i.ID,
		PredictionID: i.PredictionID,
		StationCode:  i.StationCode,
		Platform:     i.Platform,
		DoorNumber:   i.DoorNumber,
		Status:       i.Status,
		Priority:     i.Priority,
		AssignedTo:   i.AssignedTo,
		Notes:        i.Notes,
		ScheduledFor: i.ScheduledFor,
		CompletedAt:  completedStr,
		CreatedAt:    i.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    i.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
