package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type CreateInspectionInputDTO struct {
	StationCode  string
	Platform     string
	DoorNumber   int
	ScheduledFor string
	Priority     string
	Notes        string
	PredictionID *string
}

type UpdateInspectionStatusInputDTO struct {
	Status string
	Notes  string
}

type InspectionOutputDTO struct {
	ID           string
	StationCode  string
	Platform     string
	DoorNumber   int
	ScheduledFor string
	Priority     string
	Status       string
	Notes        string
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type InspectionUseCase struct {
	repo port.InspectionRepository
}

func NewInspectionUseCase(repo port.InspectionRepository) *InspectionUseCase {
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
	return &InspectionOutputDTO{
		ID:           ins.ID,
		StationCode:  ins.StationCode,
		Platform:     ins.Platform,
		DoorNumber:   ins.DoorNumber,
		ScheduledFor: ins.ScheduledFor,
		Priority:     ins.Priority,
		Status:       ins.Status,
		Notes:        ins.Notes,
		CompletedAt:  ins.CompletedAt,
		CreatedAt:    ins.CreatedAt,
		UpdatedAt:    ins.UpdatedAt,
	}
}
