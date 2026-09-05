package domain

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInspectionNotFound      = errors.New("inspecao nao encontrada")
	ErrStationCodeRequired     = errors.New("o codigo da estacao (station_code) e obrigatorio")
	ErrInvalidPlatform         = errors.New("a plataforma deve ser P1 ou P2")
	ErrInvalidDoorNumber       = errors.New("o numero da porta deve estar entre 1 e 24")
	ErrInvalidPriority         = errors.New("prioridade invalida (use: low, normal, high, critical)")
	ErrInvalidInspectionStatus = errors.New("status invalido (use: pending, in_progress, completed, canceled)")
	ErrInspectionIDRequired    = errors.New("id da inspecao e obrigatorio")
)

type Inspection struct {
	ID           string     `json:"id"`
	PredictionID *string    `json:"prediction_id,omitempty"`
	StationCode  string     `json:"station_code"`
	Platform     string     `json:"platform"`
	DoorNumber   int        `json:"door_number"`
	ScheduledFor string     `json:"scheduled_for,omitempty"`
	Priority     string     `json:"priority"`
	Status       string     `json:"status"`
	Notes        string     `json:"notes,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func NewInspection(id string, predictionID *string, stationCode, platform string, doorNumber int, scheduledFor, priority, notes string) (*Inspection, error) {
	stationCode = strings.TrimSpace(strings.ToUpper(stationCode))
	if stationCode == "" {
		return nil, ErrStationCodeRequired
	}

	platform = strings.TrimSpace(strings.ToUpper(platform))
	if platform != "P1" && platform != "P2" {
		return nil, ErrInvalidPlatform
	}

	if doorNumber < 1 || doorNumber > 24 {
		return nil, ErrInvalidDoorNumber
	}

	priority = strings.ToLower(strings.TrimSpace(priority))
	if priority == "" {
		priority = "normal"
	}
	if priority != "low" && priority != "normal" && priority != "high" && priority != "critical" {
		return nil, ErrInvalidPriority
	}

	now := time.Now()
	return &Inspection{
		ID:           id,
		PredictionID: predictionID,
		StationCode:  stationCode,
		Platform:     platform,
		DoorNumber:   doorNumber,
		ScheduledFor: strings.TrimSpace(scheduledFor),
		Priority:     priority,
		Status:       "pending",
		Notes:        strings.TrimSpace(notes),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (i *Inspection) UpdateStatus(status, notes string) (*time.Time, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "pending" && status != "in_progress" && status != "completed" && status != "canceled" {
		return nil, ErrInvalidInspectionStatus
	}

	i.Status = status
	if strings.TrimSpace(notes) != "" {
		i.Notes = strings.TrimSpace(notes)
	}

	var completedAt *time.Time
	if status == "completed" {
		now := time.Now()
		completedAt = &now
		i.CompletedAt = completedAt
	}

	i.UpdatedAt = time.Now()
	return completedAt, nil
}

type InspectionRepository interface {
	Create(ctx context.Context, inspection *Inspection) error
	UpdateStatus(ctx context.Context, id string, status, notes string, completedAt *time.Time) error
	GetByID(ctx context.Context, id string) (*Inspection, error)
	List(ctx context.Context, limit, offset int, statusFilter string) ([]*Inspection, error)
}
