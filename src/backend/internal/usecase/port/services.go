package port

import (
	"context"
	"io"
	"time"

	"github.com/g04/backend/internal/domain"
)

type MLClient interface {
	PredictBatch(ctx context.Context, doors []domain.Door) ([]domain.PredictionItem, error)
}

type EventHandler func(ctx context.Context, event domain.Event) error

type EventBus interface {
	Publish(ctx context.Context, event domain.Event)
	PublishSync(ctx context.Context, event domain.Event) error
	Subscribe(eventName string, handler EventHandler)
}

type UnitOfWorkStore interface {
	Predictions() PredictionRepository
	Inspections() InspectionRepository
	Maintenances() MaintenanceRepository
	Alarms() AlarmRepository
	Uploads() UploadRepository
}

type UnitOfWork interface {
	Do(ctx context.Context, fn func(store UnitOfWorkStore) error) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

type TokenGenerator interface {
	Generate(claims map[string]any, duration time.Duration) (string, error)
}

type IngestionStorage interface {
	Store(ctx context.Context, name string, r io.Reader) (storageKey string, hash string, err error)
	Open(ctx context.Context, storageKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, storageKey string) error
}

type RawAlarmRecord struct {
	Tag                    string
	StationCode            string
	Platform               string
	DoorNumber             int
	AlarmType              string
	DurationSeconds        int
	EventTimestamp         string
	NormalizationTimestamp string
}

type AlarmParser interface {
	StreamAlarms(ctx context.Context, r io.Reader, format string, yield func(RawAlarmRecord) error) error
}

type RawMaintenanceRecord struct {
	SAPOrderNumber       string
	OrderType            string
	InstallationLocation string
	StationCode          string
	StartTimestamp       string
	EndTimestamp         string
	DurationHours        float64
	Description          string
	Symptom              string
	Cause                string
	ActionTaken          string
}

type MaintenanceParser interface {
	StreamMaintenance(ctx context.Context, r io.Reader, format string, defaultOrderType string, yield func(RawMaintenanceRecord) error) error
}

type JobRunner interface {
	Run(job func())
}

type IngestionJobType string

const (
	IngestionJobAlarms      IngestionJobType = "alarms"
	IngestionJobMaintenance IngestionJobType = "maintenance"
)

type IngestionJob struct {
	Type         IngestionJobType
	UploadID     string
	StorageKey   string
	FileName     string
	OrderType    string
	TraceContext map[string]string
}

type IngestionJobQueue interface {
	Enqueue(ctx context.Context, upload *domain.FileUpload, job IngestionJob) error
}

type Logger interface {
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Info(msg string, args ...any)
}
