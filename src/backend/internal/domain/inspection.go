package domain

import (
	"context"
	"time"
)

type Inspection struct {
	ID           int64      `json:"id"`
	PredictionID *int64     `json:"prediction_id,omitempty"`
	StationCode  string     `json:"station_code"`
	Platform     string     `json:"platform"`
	DoorNumber   int        `json:"door_number"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	AssignedTo   *string    `json:"assigned_to,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	ScheduledFor string     `json:"scheduled_for,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type InspectionRepository interface {
	Create(ctx context.Context, inspection *Inspection) error
	UpdateStatus(ctx context.Context, id int64, status, notes string, completedAt *time.Time) error
	GetByID(ctx context.Context, id int64) (*Inspection, error)
	List(ctx context.Context, limit, offset int, statusFilter string) ([]*Inspection, error)
}
