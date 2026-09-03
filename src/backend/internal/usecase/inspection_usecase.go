package usecase

import (
	"context"
	"errors"
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
	stationCode := strings.TrimSpace(strings.ToUpper(input.StationCode))
	if stationCode == "" {
		return nil, errors.New("o codigo da estacao (station_code) e obrigatorio")
	}

	platform := strings.TrimSpace(strings.ToUpper(input.Platform))
	if platform != "P1" && platform != "P2" {
		return nil, errors.New("a plataforma deve ser P1 ou P2")
	}

	if input.DoorNumber < 1 || input.DoorNumber > 24 {
		return nil, errors.New("o numero da porta deve estar entre 1 e 24")
	}

	priority := strings.ToLower(strings.TrimSpace(input.Priority))
	if priority == "" {
		priority = "normal"
	}
	if priority != "low" && priority != "normal" && priority != "high" && priority != "critical" {
		return nil, errors.New("prioridade invalida (use: low, normal, high, critical)")
	}

	now := time.Now()
	ins := &domain.Inspection{
		ID:           uuid.New().String(),
		PredictionID: input.PredictionID,
		StationCode:  stationCode,
		Platform:     platform,
		DoorNumber:   input.DoorNumber,
		ScheduledFor: strings.TrimSpace(input.ScheduledFor),
		Priority:     priority,
		Status:       "pending",
		Notes:        strings.TrimSpace(input.Notes),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.repo.Create(ctx, ins); err != nil {
		return nil, err
	}

	return toInspectionDTO(ins), nil
}

func (uc *InspectionUseCase) UpdateStatus(ctx context.Context, id string, input UpdateInspectionStatusInputDTO) (*InspectionOutputDTO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id da inspecao e obrigatorio")
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != "pending" && status != "in_progress" && status != "completed" && status != "canceled" {
		return nil, errors.New("status invalido (use: pending, in_progress, completed, canceled)")
	}

	ins, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var completedAt *time.Time
	if status == "completed" {
		now := time.Now()
		completedAt = &now
	}

	notes := ins.Notes
	if strings.TrimSpace(input.Notes) != "" {
		notes = strings.TrimSpace(input.Notes)
	}

	if err := uc.repo.UpdateStatus(ctx, id, status, notes, completedAt); err != nil {
		return nil, err
	}

	ins.Status = status
	ins.Notes = notes
	ins.CompletedAt = completedAt
	ins.UpdatedAt = time.Now()

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
		return nil, errors.New("id da inspecao e obrigatorio")
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
