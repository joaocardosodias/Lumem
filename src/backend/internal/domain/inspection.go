package domain

import (
	"context"
	"time"
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

type InspectionRepository interface {
	Create(ctx context.Context, inspection *Inspection) error
	UpdateStatus(ctx context.Context, id string, status, notes string, completedAt *time.Time) error
	GetByID(ctx context.Context, id string) (*Inspection, error)
	List(ctx context.Context, limit, offset int, statusFilter string) ([]*Inspection, error)
}
