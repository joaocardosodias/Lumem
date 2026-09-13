package usecase

import "time"

type StationRiskOutputDTO struct {
	StationCode string
	DoorsAtRisk int
}

type DashboardSummaryOutputDTO struct {
	TotalDoors           int
	DoorsAtRisk          int
	PendingInspections   int
	CompletedInspections int
	CriticalStations     []StationRiskOutputDTO
}

type TimelineAlarmOutputDTO struct {
	ID                     string
	Tag                    string
	AlarmType              string
	DurationSeconds        int
	EventTimestamp         string
	NormalizationTimestamp string
}

type TimelineMaintenanceOutputDTO struct {
	ID             string
	OrderType      string
	SAPOrderNumber string
	StartTimestamp string
	DurationHours  float64
	Description    string
	Symptom        string
	Source         string
}

type TimelineInspectionOutputDTO struct {
	ID          string
	Status      string
	Notes       string
	CompletedAt *time.Time
	CreatedAt   time.Time
}

type DoorTimelineOutputDTO struct {
	DoorID          int
	StationName     string
	Platform        string
	DoorNumber      int
	SAPTag          string
	Status          string
	WillFail        bool
	LastEvaluatedAt *time.Time
	RecentAlarms    []TimelineAlarmOutputDTO
	Maintenances    []TimelineMaintenanceOutputDTO
	Inspections     []TimelineInspectionOutputDTO
}

type DoorStatusOutputDTO struct {
	ID         int
	Platform   string
	DoorNumber int
	SAPTag     string
	Status     string
	WillFail   bool
}

type PredictionItemOutputDTO struct {
	ID          string
	RunID       string
	StationCode string
	Platform    string
	DoorNumber  int
	WillFail    bool
	Probability *float64
}

type PredictionRunOutputDTO struct {
	ID                  string
	HorizonDays         int
	ThresholdUsed       *float64
	ModelVersion        string
	IsExperiment        bool
	PublishedAt         *time.Time
	TotalDoorsEvaluated int
	TotalDoorsAtRisk    int
	AverageProbability  *float64
	P95Probability      *float64
	MaximumProbability  *float64
	EstimatedPrecision  *float64
	EstimatedRecall     *float64
	EstimatedF1         *float64
	CreatedAt           time.Time
	Predictions         []PredictionItemOutputDTO
}

type DoorPredictionHistoryOutputDTO struct {
	ID            string
	RunID         string
	StationCode   string
	Platform      string
	DoorNumber    int
	WillFail      bool
	Probability   *float64
	HorizonDays   int
	ThresholdUsed *float64
	ModelVersion  string
	CreatedAt     time.Time
	WindowEndsAt  time.Time
}
