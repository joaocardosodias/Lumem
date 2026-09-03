package domain

import (
	"context"
	"time"
)

type MaintenanceEvent struct {
	ID                   int64     `json:"id"`
	UploadID             int64     `json:"upload_id,omitempty"`
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

type MaintenanceRepository interface {
	BatchInsertMaintenance(ctx context.Context, events []*MaintenanceEvent) error
	Create(ctx context.Context, event *MaintenanceEvent) error
	List(ctx context.Context, limit, offset int) ([]*MaintenanceEvent, error)
}
