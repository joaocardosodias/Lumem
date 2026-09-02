package domain

import (
	"context"
	"time"
)

type FileUpload struct {
	ID            int64      `json:"id"`
	FileName      string     `json:"file_name"`
	FileType      string     `json:"file_type"`
	RowsProcessed int        `json:"rows_processed"`
	RowsInserted  int        `json:"rows_inserted"`
	Status        string     `json:"status"`
	ErrorLog      string     `json:"error_log,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type Alarm struct {
	ID                     int64     `json:"id"`
	UploadID               int64     `json:"upload_id"`
	Tag                    string    `json:"tag"`
	StationCode            string    `json:"station_code"`
	Platform               string    `json:"platform"`
	DoorNumber             int       `json:"door_number"`
	AlarmType              string    `json:"alarm_type"`
	DurationSeconds        float64   `json:"duration_seconds"`
	EventTimestamp         string    `json:"event_timestamp"`
	NormalizationTimestamp string    `json:"normalization_timestamp"`
	Source                 string    `json:"source"`
	CreatedAt              time.Time `json:"created_at"`
}

type AlarmRepository interface {
	CreateUpload(ctx context.Context, upload *FileUpload) error
	UpdateUpload(ctx context.Context, upload *FileUpload) error
	GetUploadByID(ctx context.Context, id int64) (*FileUpload, error)
	BatchInsertAlarms(ctx context.Context, alarms []*Alarm) error
}
