package domain

import (
	"errors"
	"time"
)

var (
	ErrDoorNotFound                 = errors.New("porta nao encontrada")
	ErrStationRequired              = errors.New("codigo da estacao e obrigatorio")
	ErrNoPredictions                = errors.New("nenhuma rodada de previsao calculada ate o momento")
	ErrNoActiveDoors                = errors.New("nenhuma porta ativa encontrada para inferencia")
	ErrInvalidDoorID                = errors.New("o parametro ID da porta deve ser um numero inteiro positivo")
	ErrPredictionServiceUnavailable = errors.New("servico de predicao indisponivel")
)

type Station struct {
	ID        int
	Code      string
	Name      string
	CreatedAt time.Time
}

type Door struct {
	ID          int
	StationID   int
	StationCode string
	Platform    string
	DoorNumber  int
	SAPTag      string
	Status      string
	CreatedAt   time.Time
}

type PredictionRun struct {
	ID                  string
	HorizonDays         int
	TotalDoorsEvaluated int
	TotalDoorsAtRisk    int
	CreatedAt           time.Time
	Predictions         []PredictionItem
}

type PredictionItem struct {
	ID          string
	RunID       string
	StationCode string
	Platform    string
	DoorNumber  int
	WillFail    bool
}

type StationRiskSummary struct {
	StationCode string
	DoorsAtRisk int
}

type DashboardSummary struct {
	TotalDoors           int
	DoorsAtRisk          int
	PendingInspections   int
	CompletedInspections int
	CriticalStations     []StationRiskSummary
}

type DoorTimeline struct {
	Door            Door
	StationName     string
	WillFail        bool
	RecentAlarms    []*Alarm
	Maintenances    []*MaintenanceEvent
	Inspections     []*Inspection
	LastEvaluatedAt *time.Time
}

type DoorStatusItem struct {
	ID         int
	Platform   string
	DoorNumber int
	SAPTag     string
	Status     string
	WillFail   bool
}
