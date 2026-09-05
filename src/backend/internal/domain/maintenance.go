package domain

import (
	"context"
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
	ID                   string    `json:"id"`
	UploadID             string    `json:"upload_id,omitempty"`
	SAPOrderNumber       string    `json:"sap_order_number,omitempty"`
	OrderType            string    `json:"order_type"`
	StationCode          string    `json:"station_code"`
	InstallationLocation string    `json:"installation_location,omitempty"`
	StartTimestamp       string    `json:"start_timestamp"`
	EndTimestamp         string    `json:"end_timestamp,omitempty"`
	DurationHours        float64   `json:"duration_hours,omitempty"`
	Description          string    `json:"description,omitempty"`
	Symptom              string    `json:"symptom,omitempty"`
	Cause                string    `json:"cause,omitempty"`
	ActionTaken          string    `json:"action_taken,omitempty"`
	Source               string    `json:"source"`
	CreatedAt            time.Time `json:"created_at"`
}

func NewMaintenanceEvent(id, sapOrderNumber, orderType, stationCode, installationLocation, startTimestamp, endTimestamp string, durationHours float64, description, symptom, cause, actionTaken, source string) (*MaintenanceEvent, error) {
	orderType = strings.TrimSpace(strings.ToUpper(orderType))
	if orderType == "" {
		return nil, ErrOrderTypeRequired
	}

	stationCode = strings.TrimSpace(strings.ToUpper(stationCode))
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

type MaintenanceRepository interface {
	BatchInsertMaintenance(ctx context.Context, events []*MaintenanceEvent) error
	Create(ctx context.Context, event *MaintenanceEvent) error
	List(ctx context.Context, limit, offset int) ([]*MaintenanceEvent, error)
}
