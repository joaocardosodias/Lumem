package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrOrderTypeRequired          = errors.New("o tipo da ordem (order_type) e obrigatorio")
	ErrMaintenanceStationRequired = errors.New("o codigo da estacao (station_code) e obrigatorio")
	ErrStartTimestampRequired     = errors.New("a data de inicio (start_timestamp) e obrigatoria")
	ErrMaintenanceNotFound        = errors.New("registro de manutencao nao encontrado")
)

type MaintenanceEvent struct {
	ID                   string
	UploadID             string
	SAPOrderNumber       string
	OrderType            string
	StationCode          string
	InstallationLocation string
	StartTimestamp       string
	EndTimestamp         string
	DurationHours        float64
	Description          string
	Symptom              string
	Cause                string
	ActionTaken          string
	Source               string
	CreatedAt            time.Time
}

type CreateMaintenanceParams struct {
	ID                   string
	SAPOrderNumber       string
	OrderType            string
	StationCode          string
	InstallationLocation string
	StartTimestamp       string
	EndTimestamp         string
	DurationHours        float64
	Description          string
	Symptom              string
	Cause                string
	ActionTaken          string
	Source               string
}

func NewMaintenanceEvent(p CreateMaintenanceParams) (*MaintenanceEvent, error) {
	orderType := strings.TrimSpace(strings.ToUpper(p.OrderType))
	if orderType == "" {
		return nil, ErrOrderTypeRequired
	}

	stationCode := strings.TrimSpace(strings.ToUpper(p.StationCode))
	if stationCode == "" && p.InstallationLocation != "" {
		parts := strings.Split(p.InstallationLocation, "-")
		if len(parts) >= 4 {
			stationCode = strings.TrimSpace(strings.ToUpper(parts[3]))
		}
	}
	if stationCode == "" {
		return nil, ErrMaintenanceStationRequired
	}

	startTimestamp := strings.TrimSpace(p.StartTimestamp)
	if startTimestamp == "" {
		return nil, ErrStartTimestampRequired
	}

	return &MaintenanceEvent{
		ID:                   p.ID,
		SAPOrderNumber:       strings.TrimSpace(p.SAPOrderNumber),
		OrderType:            orderType,
		StationCode:          stationCode,
		InstallationLocation: strings.TrimSpace(p.InstallationLocation),
		StartTimestamp:       startTimestamp,
		EndTimestamp:         strings.TrimSpace(p.EndTimestamp),
		DurationHours:        p.DurationHours,
		Description:          strings.TrimSpace(p.Description),
		Symptom:              strings.TrimSpace(p.Symptom),
		Cause:                strings.TrimSpace(p.Cause),
		ActionTaken:          strings.TrimSpace(p.ActionTaken),
		Source:               p.Source,
		CreatedAt:            time.Now(),
	}, nil
}
