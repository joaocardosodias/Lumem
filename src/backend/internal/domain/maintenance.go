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

func NewMaintenanceEvent(id, sapOrderNumber, orderType, stationCode, installationLocation, startTimestamp, endTimestamp string, durationHours float64, description, symptom, cause, actionTaken, source string) (*MaintenanceEvent, error) {
	orderType = strings.TrimSpace(strings.ToUpper(orderType))
	if orderType == "" {
		return nil, ErrOrderTypeRequired
	}

	stationCode = strings.TrimSpace(strings.ToUpper(stationCode))
	if stationCode == "" && installationLocation != "" {
		parts := strings.Split(installationLocation, "-")
		if len(parts) >= 4 {
			stationCode = strings.TrimSpace(strings.ToUpper(parts[3]))
		}
	}
	if stationCode == "" {
		return nil, ErrMaintenanceStationRequired
	}

	startTimestamp = strings.TrimSpace(startTimestamp)
	if startTimestamp == "" {
		return nil, ErrStartTimestampRequired
	}

	return &MaintenanceEvent{
		ID:                   id,
		SAPOrderNumber:       strings.TrimSpace(sapOrderNumber),
		OrderType:            orderType,
		StationCode:          stationCode,
		InstallationLocation: strings.TrimSpace(installationLocation),
		StartTimestamp:       startTimestamp,
		EndTimestamp:         strings.TrimSpace(endTimestamp),
		DurationHours:        durationHours,
		Description:          strings.TrimSpace(description),
		Symptom:              strings.TrimSpace(symptom),
		Cause:                strings.TrimSpace(cause),
		ActionTaken:          strings.TrimSpace(actionTaken),
		Source:               source,
		CreatedAt:            time.Now(),
	}, nil
}
