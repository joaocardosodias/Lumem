package domain

import (
	"context"
	"time"
)

type Station struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Door struct {
	ID          int       `json:"id"`
	StationID   int       `json:"station_id"`
	StationCode string    `json:"station_code,omitempty"`
	Platform    string    `json:"platform"`
	DoorNumber  int       `json:"door_number"`
	SAPTag      string    `json:"sap_tag"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type PredictionRun struct {
	ID                  int64            `json:"id"`
	HorizonDays         int              `json:"horizon_days"`
	TotalDoorsEvaluated int              `json:"total_doors_evaluated"`
	TotalDoorsAtRisk    int              `json:"total_doors_at_risk"`
	CreatedAt           time.Time        `json:"created_at"`
	Predictions         []PredictionItem `json:"predictions,omitempty"`
}

type PredictionItem struct {
	ID          int64  `json:"id"`
	RunID       int64  `json:"run_id"`
	StationCode string `json:"station_code"`
	Platform    string `json:"platform"`
	DoorNumber  int    `json:"door_number"`
	WillFail    bool   `json:"will_fail"`
}

type StationRiskSummary struct {
	StationCode string `json:"station_code"`
	DoorsAtRisk int    `json:"doors_at_risk"`
}

type DashboardSummary struct {
	TotalDoors           int                  `json:"total_doors"`
	DoorsAtRisk          int                  `json:"doors_at_risk"`
	PendingInspections   int                  `json:"pending_inspections"`
	CompletedInspections int                  `json:"completed_inspections"`
	CriticalStations     []StationRiskSummary `json:"critical_stations"`
}

type DoorTimeline struct {
	Door            Door                `json:"door"`
	StationName     string              `json:"station_name"`
	WillFail        bool                `json:"will_fail"`
	RecentAlarms    []*Alarm            `json:"recent_alarms"`
	Maintenances    []*MaintenanceEvent `json:"maintenances"`
	Inspections     []*Inspection       `json:"inspections"`
	LastEvaluatedAt *time.Time          `json:"last_evaluated_at,omitempty"`
}

type DoorStatusItem struct {
	ID         int    `json:"id"`
	Platform   string `json:"platform"`
	DoorNumber int    `json:"door_number"`
	SAPTag     string `json:"sap_tag"`
	Status     string `json:"status"`
	WillFail   bool   `json:"will_fail"`
}

type DashboardRepository interface {
	GetSummary(ctx context.Context) (*DashboardSummary, error)
	GetDoorTimeline(ctx context.Context, doorID int) (*DoorTimeline, error)
	GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*DoorStatusItem, error)
}

type PredictionRepository interface {
	SaveRun(ctx context.Context, run *PredictionRun, items []PredictionItem) error
	GetLatestRun(ctx context.Context) (*PredictionRun, error)
	ListDoorsForInference(ctx context.Context) ([]Door, error)
}

type MLClient interface {
	PredictBatch(ctx context.Context, doors []Door) ([]PredictionItem, error)
}
