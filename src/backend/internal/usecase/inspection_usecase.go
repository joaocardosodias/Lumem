package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
)

type CreateInspectionInputDTO struct {
	StationCode  string  `json:"station_code"`
	Platform     string  `json:"platform"`
	DoorNumber   int     `json:"door_number"`
	ScheduledFor string  `json:"scheduled_for"`
	Priority     string  `json:"priority"`
	Notes        string  `json:"notes"`
	PredictionID *string `json:"prediction_id"`
}

type UpdateInspectionStatusInputDTO struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

type InspectionOutputDTO struct {
	ID           string    `json:"id"`
	StationCode  string    `json:"station_code"`
	Platform     string    `json:"platform"`
	DoorNumber   int       `json:"door_number"`
	ScheduledFor string    `json:"scheduled_for,omitempty"`
	Priority     string    `json:"priority"`
	Status       string    `json:"status"`
	Notes        string    `json:"notes,omitempty"`
	CompletedAt  *string   `json:"completed_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type InspectionUseCase struct {
	repo domain.InspectionRepository
}

func NewInspectionUseCase(repo domain.InspectionRepository) *InspectionUseCase {
	return &InspectionUseCase{repo: repo}
}

func (uc *InspectionUseCase) Create(ctx context.Context, input CreateInspectionInputDTO) (*InspectionOutputDTO, error) {
	ins, err := domain.NewInspection(
		uuid.New().String(),
		input.PredictionID,
		input.StationCode,
		input.Platform,
		input.DoorNumber,
		input.ScheduledFor,
		input.Priority,
		input.Notes,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, ins); err != nil {
		return nil, err
	}

	return toInspectionDTO(ins), nil
}

func (uc *InspectionUseCase) UpdateStatus(ctx context.Context, id string, input UpdateInspectionStatusInputDTO) (*InspectionOutputDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInspectionIDRequired
	}

	ins, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	completedAt, err := ins.UpdateStatus(input.Status, input.Notes)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateStatus(ctx, id, ins.Status, ins.Notes, completedAt); err != nil {
		return nil, err
	}

	return toInspectionDTO(ins), nil
}

func (uc *InspectionUseCase) List(ctx context.Context, limit, offset int, statusFilter string) ([]*InspectionOutputDTO, error) {
	inspections, err := uc.repo.List(ctx, limit, offset, strings.TrimSpace(statusFilter))
	if err != nil {
		return nil, err
	}

	dtos := make([]*InspectionOutputDTO, 0, len(inspections))
	for _, ins := range inspections {
		dtos = append(dtos, toInspectionDTO(ins))
	}
	return dtos, nil
}

func (uc *InspectionUseCase) GetByID(ctx context.Context, id string) (*InspectionOutputDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, domain.ErrInspectionIDRequired
	}

	ins, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toInspectionDTO(ins), nil
}

func toInspectionDTO(ins *domain.Inspection) *InspectionOutputDTO {
	var compStr *string
	if ins.CompletedAt != nil {
		s := ins.CompletedAt.Format("2006-01-02 15:04:05")
		compStr = &s
	}

	return &InspectionOutputDTO{
		ID:           ins.ID,
		StationCode:  ins.StationCode,
		Platform:     ins.Platform,
		DoorNumber:   ins.DoorNumber,
		ScheduledFor: ins.ScheduledFor,
		Priority:     ins.Priority,
		Status:       ins.Status,
		Notes:        ins.Notes,
		CompletedAt:  compStr,
		CreatedAt:    ins.CreatedAt,
		UpdatedAt:    ins.UpdatedAt,
	}
}
