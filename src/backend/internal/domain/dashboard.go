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
	ErrExperimentNotFound           = errors.New("experimento de predicao nao encontrado")
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
	Predictions         []PredictionItem
}

type PredictionItem struct {
	ID          string
	RunID       string
	StationCode string
	Platform    string
	DoorNumber  int
	WillFail    bool
	Probability *float64
}

type DoorPredictionHistoryItem struct {
	PredictionItem
	HorizonDays   int
	ThresholdUsed *float64
	ModelVersion  string
	CreatedAt     time.Time
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
