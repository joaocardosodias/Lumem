package port

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

type InspectionRepository interface {
	Create(ctx context.Context, ins *domain.Inspection) error
	UpdateStatus(ctx context.Context, id string, status, notes string, completedAt *time.Time) error
	GetByID(ctx context.Context, id string) (*domain.Inspection, error)
	List(ctx context.Context, limit, offset int, statusFilter string) ([]*domain.Inspection, error)
}

type MaintenanceRepository interface {
	BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error
	DeleteByUploadID(ctx context.Context, uploadID string) error
	Create(ctx context.Context, event *domain.MaintenanceEvent) error
	List(ctx context.Context, limit, offset int) ([]*domain.MaintenanceEvent, error)
}

type AlarmFilter struct {
	StationCode string
	Platform    string
	DoorNumber  int
	AlarmType   string
	Tag         string
	Limit       int
	Offset      int
}

type AlarmRepository interface {
	BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error
	DeleteByUploadID(ctx context.Context, uploadID string) error
	List(ctx context.Context, filter AlarmFilter) ([]*domain.Alarm, error)
}

type UploadRepository interface {
	CreateUpload(ctx context.Context, upload *domain.FileUpload) error
	UpdateUpload(ctx context.Context, upload *domain.FileUpload) error
	GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error)
	GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error)
	ReconcileStaleUploads(ctx context.Context) error
}

type DashboardRepository interface {
	GetSummary(ctx context.Context) (*domain.DashboardSummary, error)
	GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error)
	GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error)
}

type PredictionRepository interface {
	SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error
	GetLatestRun(ctx context.Context) (*domain.PredictionRun, error)
	ListDoorsForInference(ctx context.Context) ([]domain.Door, error)
}
