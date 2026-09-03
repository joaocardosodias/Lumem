package usecase

type StationRiskOutputDTO struct {
	StationCode string `json:"station_code"`
	DoorsAtRisk int    `json:"doors_at_risk"`
}

type DashboardSummaryOutputDTO struct {
	TotalDoors           int                    `json:"total_doors"`
	DoorsAtRisk          int                    `json:"doors_at_risk"`
	PendingInspections   int                    `json:"pending_inspections"`
	CompletedInspections int                    `json:"completed_inspections"`
	CriticalStations     []StationRiskOutputDTO `json:"critical_stations"`
}

type TimelineAlarmOutputDTO struct {
	ID                     int64  `json:"id"`
	Tag                    string `json:"tag"`
	AlarmType              string `json:"alarm_type"`
	DurationSeconds        int    `json:"duration_seconds"`
	EventTimestamp         string `json:"event_timestamp"`
	NormalizationTimestamp string `json:"normalization_timestamp,omitempty"`
}

type TimelineMaintenanceOutputDTO struct {
	ID             int64   `json:"id"`
	OrderType      string  `json:"order_type"`
	SAPOrderNumber string  `json:"sap_order_number,omitempty"`
	StartTimestamp string  `json:"start_timestamp"`
	DurationHours  float64 `json:"duration_hours,omitempty"`
	Description    string  `json:"description,omitempty"`
	Symptom        string  `json:"symptom,omitempty"`
	Source         string  `json:"source"`
}

type TimelineInspectionOutputDTO struct {
	ID           int64   `json:"id"`
	Status       string  `json:"status"`
	Priority     string  `json:"priority"`
	Notes        string  `json:"notes,omitempty"`
	ScheduledFor string  `json:"scheduled_for,omitempty"`
	CompletedAt  *string `json:"completed_at,omitempty"`
}

type DoorTimelineOutputDTO struct {
	DoorID          int                            `json:"door_id"`
	StationName     string                         `json:"station_name"`
	Platform        string                         `json:"platform"`
	DoorNumber      int                            `json:"door_number"`
	SAPTag          string                         `json:"sap_tag"`
	Status          string                         `json:"status"`
	WillFail        bool                           `json:"will_fail"`
	LastEvaluatedAt *string                        `json:"last_evaluated_at,omitempty"`
	RecentAlarms    []TimelineAlarmOutputDTO       `json:"recent_alarms"`
	Maintenances    []TimelineMaintenanceOutputDTO `json:"maintenances"`
	Inspections     []TimelineInspectionOutputDTO  `json:"inspections"`
}

type DoorStatusOutputDTO struct {
	ID         int    `json:"id"`
	Platform   string `json:"platform"`
	DoorNumber int    `json:"door_number"`
	SAPTag     string `json:"sap_tag"`
	Status     string `json:"status"`
	WillFail   bool   `json:"will_fail"`
}

type PredictionItemOutputDTO struct {
	ID          int64  `json:"id"`
	StationCode string `json:"station_code"`
	Platform    string `json:"platform"`
	DoorNumber  int    `json:"door_number"`
	WillFail    bool   `json:"will_fail"`
}

type PredictionRunOutputDTO struct {
	ID                  int64                     `json:"id"`
	HorizonDays         int                       `json:"horizon_days"`
	TotalDoorsEvaluated int                       `json:"total_doors_evaluated"`
	TotalDoorsAtRisk    int                       `json:"total_doors_at_risk"`
	CreatedAt           string                    `json:"created_at"`
	Predictions         []PredictionItemOutputDTO `json:"predictions,omitempty"`
}
