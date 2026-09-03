package domain

import (
	"context"
	"time"
)

type MaintenanceEvent struct {
	ID                   int64     `json:"id"`
	UploadID             int64     `json:"upload_id"`
	SAPOrderNumber       string    `json:"sap_order_number"`
	OrderType            string    `json:"order_type"`
	StationCode          string    `json:"station_code"`
	InstallationLocation string    `json:"installation_location"`
	StartTimestamp       string    `json:"start_timestamp"`
	EndTimestamp         string    `json:"end_timestamp"`
	DurationHours        float64   `json:"duration_hours"`
	Description          string    `json:"description"`
	Symptom              string    `json:"symptom"`
	Cause                string    `json:"cause"`
	ActionTaken          string    `json:"action_taken"`
	Source               string    `json:"source"`
	CreatedAt            time.Time `json:"created_at"`
}

type MaintenanceRepository interface {
	BatchInsertMaintenance(ctx context.Context, events []*MaintenanceEvent) error
}
