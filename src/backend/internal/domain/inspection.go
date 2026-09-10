package domain

import (
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
	ErrInvalidStatusTransition = errors.New("transicao de status invalida")
)

type Inspection struct {
	ID           string
	PredictionID *string
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

type CreateInspectionParams struct {
	ID           string
	PredictionID *string
	StationCode  string
	Platform     string
	DoorNumber   int
	ScheduledFor string
	Priority     string
	Notes        string
}

func NewInspection(p CreateInspectionParams) (*Inspection, error) {
	stationCode := strings.TrimSpace(strings.ToUpper(p.StationCode))
	if stationCode == "" {
		return nil, ErrStationCodeRequired
	}

	platform := strings.TrimSpace(strings.ToUpper(p.Platform))
	if platform != "P1" && platform != "P2" {
		return nil, ErrInvalidPlatform
	}

	if p.DoorNumber < 1 || p.DoorNumber > 24 {
		return nil, ErrInvalidDoorNumber
	}

	priority := strings.ToLower(strings.TrimSpace(p.Priority))
	if priority == "" {
		priority = "normal"
	}
	if priority != "low" && priority != "normal" && priority != "high" && priority != "critical" {
		return nil, ErrInvalidPriority
	}

	now := time.Now()
	return &Inspection{
		ID:           p.ID,
		PredictionID: p.PredictionID,
		StationCode:  stationCode,
		Platform:     platform,
		DoorNumber:   p.DoorNumber,
		ScheduledFor: strings.TrimSpace(p.ScheduledFor),
		Priority:     priority,
		Status:       "pending",
		Notes:        strings.TrimSpace(p.Notes),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func canTransitionInspectionStatus(current, next string) bool {
	if current == "" || current == next {
		return true
	}
	switch current {
	case "pending":
		return next == "in_progress" || next == "completed" || next == "canceled"
	case "in_progress":
		return next == "completed" || next == "pending" || next == "canceled"
	case "completed":
		return next == "in_progress" || next == "pending"
	case "canceled":
		return false
	default:
		return false
	}
}

func (i *Inspection) UpdateStatus(status, notes string) (*time.Time, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "pending" && status != "in_progress" && status != "completed" && status != "canceled" {
		return nil, ErrInvalidInspectionStatus
	}

	if !canTransitionInspectionStatus(i.Status, status) {
		return nil, ErrInvalidStatusTransition
	}

	i.Status = status
	if strings.TrimSpace(notes) != "" {
		i.Notes = strings.TrimSpace(notes)
	}

	if status == "completed" {
		if i.CompletedAt == nil {
			now := time.Now()
			i.CompletedAt = &now
		}
	} else {
		i.CompletedAt = nil
	}

	i.UpdatedAt = time.Now()
	return i.CompletedAt, nil
}
