package domain

import (
	"context"
	"time"
)

type Event interface {
	EventName() string
	OccurredAt() time.Time
}

type EventHandler func(ctx context.Context, event Event) error

type EventBus interface {
	Publish(ctx context.Context, event Event)
	Subscribe(eventName string, handler EventHandler)
}

type DoorRiskDetectedEvent struct {
	RunID       string    `json:"run_id"`
	StationCode string    `json:"station_code"`
	Platform    string    `json:"platform"`
	DoorNumber  int       `json:"door_number"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e DoorRiskDetectedEvent) EventName() string {
	return "DoorRiskDetected"
}

func (e DoorRiskDetectedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

type PredictionRunCompletedEvent struct {
	RunID          string    `json:"run_id"`
	HorizonDays    int       `json:"horizon_days"`
	TotalEvaluated int       `json:"total_evaluated"`
	TotalAtRisk    int       `json:"total_at_risk"`
	Timestamp      time.Time `json:"timestamp"`
}

func (e PredictionRunCompletedEvent) EventName() string {
	return "PredictionRunCompleted"
}

func (e PredictionRunCompletedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

type FileUploadCompletedEvent struct {
	UploadID      string    `json:"upload_id"`
	FileType      string    `json:"file_type"`
	RowsProcessed int       `json:"rows_processed"`
	RowsInserted  int       `json:"rows_inserted"`
	Timestamp     time.Time `json:"timestamp"`
}

func (e FileUploadCompletedEvent) EventName() string {
	return "FileUploadCompleted"
}

func (e FileUploadCompletedEvent) OccurredAt() time.Time {
	return e.Timestamp
}
