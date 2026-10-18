package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
)

func TestAlarm_Validation(t *testing.T) {
	validParams := domain.CreateAlarmParams{
		ID:              "alarm-1",
		Tag:             "TAG-01",
		StationCode:     "FAL",
		Platform:        "P1",
		DoorNumber:      1,
		AlarmType:       "Falha",
		DurationSeconds: 10,
		EventTimestamp:  "2026-09-01 10:00:00",
		Source:          "telemetry",
	}

	alarm, err := domain.NewAlarm(validParams)
	if err != nil || alarm == nil {
		t.Fatalf("expected valid alarm, got error: %v", err)
	}

	p := validParams
	p.Source = ""
	alarmDefSource, err := domain.NewAlarm(p)
	if err != nil || alarmDefSource.Source != "telemetry" {
		t.Fatalf("expected default telemetry source")
	}

	p = validParams
	p.Tag = ""
	if _, err := domain.NewAlarm(p); !errors.Is(err, domain.ErrAlarmTagRequired) {
		t.Fatalf("expected ErrAlarmTagRequired")
	}

	p = validParams
	p.StationCode = ""
	if _, err := domain.NewAlarm(p); !errors.Is(err, domain.ErrAlarmStationRequired) {
		t.Fatalf("expected ErrAlarmStationRequired")
	}

	p = validParams
	p.Platform = "P3"
	if _, err := domain.NewAlarm(p); !errors.Is(err, domain.ErrInvalidAlarmPlatform) {
		t.Fatalf("expected ErrInvalidAlarmPlatform")
	}

	p = validParams
	p.DoorNumber = 0
	if _, err := domain.NewAlarm(p); !errors.Is(err, domain.ErrInvalidAlarmDoorNumber) {
		t.Fatalf("expected ErrInvalidAlarmDoorNumber for 0")
	}

	p.DoorNumber = 25
	if _, err := domain.NewAlarm(p); !errors.Is(err, domain.ErrInvalidAlarmDoorNumber) {
		t.Fatalf("expected ErrInvalidAlarmDoorNumber for 25")
	}

	p = validParams
	p.AlarmType = ""
	if _, err := domain.NewAlarm(p); !errors.Is(err, domain.ErrAlarmTypeRequired) {
		t.Fatalf("expected ErrAlarmTypeRequired")
	}

	p = validParams
	p.EventTimestamp = ""
	if _, err := domain.NewAlarm(p); !errors.Is(err, domain.ErrAlarmTimestampRequired) {
		t.Fatalf("expected ErrAlarmTimestampRequired")
	}
}

func TestInspection_CreationValidation(t *testing.T) {
	validParams := domain.CreateInspectionParams{
		ID:          "insp-1",
		StationCode: "PIN",
		Platform:    "P2",
		DoorNumber:  5,
		Notes:       "test notes",
	}

	insp, err := domain.NewInspection(validParams)
	if err != nil || insp == nil {
		t.Fatalf("expected valid inspection, got %v", err)
	}

	p := validParams
	p.StationCode = ""
	if _, err := domain.NewInspection(p); !errors.Is(err, domain.ErrStationCodeRequired) {
		t.Fatalf("expected ErrStationCodeRequired")
	}

	p = validParams
	p.Platform = "P4"
	if _, err := domain.NewInspection(p); !errors.Is(err, domain.ErrInvalidPlatform) {
		t.Fatalf("expected ErrInvalidPlatform")
	}

	p = validParams
	p.DoorNumber = 0
	if _, err := domain.NewInspection(p); !errors.Is(err, domain.ErrInvalidDoorNumber) {
		t.Fatalf("expected ErrInvalidDoorNumber for 0")
	}

	p.DoorNumber = 25
	if _, err := domain.NewInspection(p); !errors.Is(err, domain.ErrInvalidDoorNumber) {
		t.Fatalf("expected ErrInvalidDoorNumber for 25")
	}
}

func TestInspection_StatusTransitions(t *testing.T) {
	validParams := domain.CreateInspectionParams{
		ID:          "insp-1",
		StationCode: "PIN",
		Platform:    "P2",
		DoorNumber:  5,
		Notes:       "test notes",
	}
	insp, _ := domain.NewInspection(validParams)

	if _, err := insp.UpdateStatus("invalid", ""); !errors.Is(err, domain.ErrInvalidInspectionStatus) {
		t.Fatalf("expected ErrInvalidInspectionStatus")
	}

	if _, err := insp.UpdateStatus("in_progress", "working"); err != nil {
		t.Fatalf("expected valid transition to in_progress: %v", err)
	}

	completedAt, err := insp.UpdateStatus("completed", "finished")
	if err != nil || completedAt == nil {
		t.Fatalf("expected valid transition to completed: %v", err)
	}

	secondCompletedAt, err := insp.UpdateStatus("completed", "already done")
	if err != nil || secondCompletedAt == nil {
		t.Fatalf("expected idempotent completed update: %v", err)
	}

	if _, err := insp.UpdateStatus("in_progress", "reopened"); err != nil {
		t.Fatalf("expected completed to in_progress transition: %v", err)
	}

	if _, err := insp.UpdateStatus("pending", "back to pending"); err != nil {
		t.Fatalf("expected in_progress to pending transition: %v", err)
	}

	if _, err := insp.UpdateStatus("canceled", "aborted"); err != nil {
		t.Fatalf("expected pending to canceled transition: %v", err)
	}

	if _, err := insp.UpdateStatus("in_progress", "try re-open"); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition from canceled")
	}

	emptyStatusInsp := &domain.Inspection{}
	if _, err := emptyStatusInsp.UpdateStatus("pending", "init"); err != nil {
		t.Fatalf("expected valid transition from empty status: %v", err)
	}

	if _, err := emptyStatusInsp.UpdateStatus("pending", "same"); err != nil {
		t.Fatalf("expected valid transition to same status: %v", err)
	}

	unknownStatusInsp := &domain.Inspection{Status: "unknown"}
	if _, err := unknownStatusInsp.UpdateStatus("completed", "test"); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition from unknown status")
	}

	completedInsp := &domain.Inspection{Status: "completed"}
	if _, err := completedInsp.UpdateStatus("canceled", "test"); !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition from completed to canceled")
	}
}

func TestMaintenanceEvent_Validation(t *testing.T) {
	validParams := domain.CreateMaintenanceParams{
		ID:                   "maint-1",
		OrderType:            "ZM01",
		StationCode:          "FAL",
		InstallationLocation: "SP04-6-PSD-FAL-P1-PP04",
		StartTimestamp:       "2026-09-01 12:00:00",
		Description:          "descricao",
	}

	m, err := domain.NewMaintenanceEvent(validParams)
	if err != nil || m == nil {
		t.Fatalf("expected valid maintenance event, got %v", err)
	}

	p := validParams
	p.StationCode = ""
	p.InstallationLocation = "SP04-6-PSD-PIN-P2-PP08"
	mFromLoc, err := domain.NewMaintenanceEvent(p)
	if err != nil || mFromLoc.StationCode != "PIN" {
		t.Fatalf("expected station PIN extracted from location, got: %+v", mFromLoc)
	}

	p.InstallationLocation = "INVALID"
	if _, err := domain.NewMaintenanceEvent(p); !errors.Is(err, domain.ErrMaintenanceStationRequired) {
		t.Fatalf("expected ErrMaintenanceStationRequired for invalid location")
	}

	p = validParams
	p.OrderType = ""
	if _, err := domain.NewMaintenanceEvent(p); !errors.Is(err, domain.ErrOrderTypeRequired) {
		t.Fatalf("expected ErrOrderTypeRequired")
	}

	p = validParams
	p.StartTimestamp = ""
	if _, err := domain.NewMaintenanceEvent(p); !errors.Is(err, domain.ErrStartTimestampRequired) {
		t.Fatalf("expected ErrStartTimestampRequired")
	}
}

func TestEvents(t *testing.T) {
	now := time.Now()

	e1 := domain.DoorRiskDetectedEvent{
		RunID:       "run-1",
		StationCode: "FAL",
		Platform:    "P1",
		DoorNumber:  4,
		Timestamp:   now,
	}
	if e1.EventName() != "DoorRiskDetected" || !e1.OccurredAt().Equal(now) {
		t.Fatalf("unexpected DoorRiskDetectedEvent")
	}

	e2 := domain.PredictionRunCompletedEvent{
		RunID:          "run-1",
		HorizonDays:    7,
		TotalEvaluated: 10,
		TotalAtRisk:    2,
		Timestamp:      now,
	}
	if e2.EventName() != "PredictionRunCompleted" || !e2.OccurredAt().Equal(now) {
		t.Fatalf("unexpected PredictionRunCompletedEvent")
	}

	e3 := domain.FileUploadCompletedEvent{
		UploadID:      "up-1",
		FileType:      "alarms",
		RowsProcessed: 100,
		RowsInserted:  90,
		Timestamp:     now,
	}
	if e3.EventName() != "FileUploadCompleted" || !e3.OccurredAt().Equal(now) {
		t.Fatalf("unexpected FileUploadCompletedEvent")
	}
}
