package domain

import (
	"time"
)

type Event interface {
	EventName() string
	OccurredAt() time.Time
}

type DoorRiskDetectedEvent struct {
	RunID       string
	StationCode string
	Platform    string
	DoorNumber  int
	Timestamp   time.Time
}

func (e DoorRiskDetectedEvent) EventName() string {
	return "DoorRiskDetected"
}

func (e DoorRiskDetectedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

type PredictionRunCompletedEvent struct {
	RunID          string
	HorizonDays    int
	TotalEvaluated int
	TotalAtRisk    int
	Timestamp      time.Time
}

func (e PredictionRunCompletedEvent) EventName() string {
	return "PredictionRunCompleted"
}

func (e PredictionRunCompletedEvent) OccurredAt() time.Time {
	return e.Timestamp
}

type FileUploadCompletedEvent struct {
	UploadID      string
	FileType      string
	RowsProcessed int
	RowsInserted  int
	Timestamp     time.Time
}

func (e FileUploadCompletedEvent) EventName() string {
	return "FileUploadCompleted"
}

func (e FileUploadCompletedEvent) OccurredAt() time.Time {
	return e.Timestamp
}
