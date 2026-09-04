package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidCSVFormat = errors.New("formato de arquivo CSV invalido")
	ErrDuplicateUpload  = errors.New("arquivo duplicado detectado (idempotencia sha-256)")
)

type FileUpload struct {
	ID            string     `json:"id"`
	FileName      string     `json:"file_name"`
	FileType      string     `json:"file_type"`
	FileHash      string     `json:"file_hash,omitempty"`
	RowsProcessed int        `json:"rows_processed"`
	RowsInserted  int        `json:"rows_inserted"`
	Status        string     `json:"status"`
	ErrorLog      string     `json:"error_log,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type Alarm struct {
	ID                     string    `json:"id"`
	UploadID               string    `json:"upload_id"`
	Tag                    string    `json:"tag"`
	StationCode            string    `json:"station_code"`
	Platform               string    `json:"platform"`
	DoorNumber             int       `json:"door_number"`
	AlarmType              string    `json:"alarm_type"`
	DurationSeconds        int       `json:"duration_seconds"`
	EventTimestamp         string    `json:"event_timestamp"`
	NormalizationTimestamp string    `json:"normalization_timestamp,omitempty"`
	Source                 string    `json:"source"`
	CreatedAt              time.Time `json:"created_at"`
}

type AlarmRepository interface {
	CreateUpload(ctx context.Context, upload *FileUpload) error
	UpdateUpload(ctx context.Context, upload *FileUpload) error
	GetUploadByID(ctx context.Context, id string) (*FileUpload, error)
	GetUploadByHash(ctx context.Context, hash string) (*FileUpload, error)
	BatchInsertAlarms(ctx context.Context, alarms []*Alarm) error
	ReconcileStaleUploads(ctx context.Context) error
}
