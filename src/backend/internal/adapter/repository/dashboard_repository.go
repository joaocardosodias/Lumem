package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

var _ port.DashboardRepository = (*PostgresDashboardRepository)(nil)

type PostgresDashboardRepository struct {
	db DBExecutor
}

func NewPostgresDashboardRepository(db DBExecutor) *PostgresDashboardRepository {
	return &PostgresDashboardRepository{db: db}
}

func (r *PostgresDashboardRepository) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	var totalDoors int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM doors WHERE status = 'active'").Scan(&totalDoors)
	if err != nil {
		return nil, err
	}

	var doorsAtRisk int
	_ = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(total_doors_at_risk, 0)
		FROM prediction_runs
		ORDER BY created_at DESC LIMIT 1
	`).Scan(&doorsAtRisk)

	var pendingInspections, completedInspections int
	_ = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM inspections WHERE status = 'pending'").Scan(&pendingInspections)
	_ = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM inspections WHERE status = 'completed'").Scan(&completedInspections)

	rows, err := r.db.QueryContext(ctx, `
		SELECT s.code, COUNT(p.id)
		FROM predictions p
		JOIN prediction_runs pr ON p.run_id = pr.id
		JOIN stations s ON p.station_code = s.code
		WHERE p.will_fail = TRUE AND pr.id = (SELECT id FROM prediction_runs ORDER BY created_at DESC LIMIT 1)
		GROUP BY s.code
		ORDER BY COUNT(p.id) DESC
		LIMIT 5
	`)
	var criticalStations []domain.StationRiskSummary
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item domain.StationRiskSummary
			if err := rows.Scan(&item.StationCode, &item.DoorsAtRisk); err == nil {
				criticalStations = append(criticalStations, item)
			}
		}
	}

	return &domain.DashboardSummary{
		TotalDoors:           totalDoors,
		DoorsAtRisk:          doorsAtRisk,
		PendingInspections:   pendingInspections,
		CompletedInspections: completedInspections,
		CriticalStations:     criticalStations,
	}, nil
}

func (r *PostgresDashboardRepository) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	var door domain.Door
	var stationName string
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT d.id, d.station_id, s.code, s.name, d.platform, d.door_number, d.sap_tag, d.status, d.created_at
		FROM doors d
		JOIN stations s ON d.station_id = s.id
		WHERE d.id = $1
	`, doorID).Scan(
		&door.ID,
		&door.StationID,
		&door.StationCode,
		&stationName,
		&door.Platform,
		&door.DoorNumber,
		&door.SAPTag,
		&door.Status,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDoorNotFound
		}
		return nil, err
	}
	door.CreatedAt = createdAt

	var willFail bool
	var lastEvalAt *time.Time
	_ = r.db.QueryRowContext(ctx, `
		SELECT p.will_fail, pr.created_at
		FROM predictions p
		JOIN prediction_runs pr ON p.run_id = pr.id
		WHERE p.station_code = $1 AND p.platform = $2 AND p.door_number = $3
		  AND p.run_id = (SELECT id FROM prediction_runs ORDER BY created_at DESC LIMIT 1)
		LIMIT 1
	`, door.StationCode, door.Platform, door.DoorNumber).Scan(&willFail, &lastEvalAt)

	var alarms []*domain.Alarm
	alarmRows, err := r.db.QueryContext(ctx, `
		SELECT id, tag, alarm_type, duration_seconds, event_timestamp, COALESCE(normalization_timestamp, '')
		FROM telemetry_alarms
		WHERE door_number = $1 AND station_code = $2
		ORDER BY event_timestamp DESC
		LIMIT 20
	`, door.DoorNumber, door.StationCode)
	if err == nil {
		defer alarmRows.Close()
		for alarmRows.Next() {
			var a domain.Alarm
			if err := alarmRows.Scan(&a.ID, &a.Tag, &a.AlarmType, &a.DurationSeconds, &a.EventTimestamp, &a.NormalizationTimestamp); err == nil {
				alarms = append(alarms, &a)
			}
		}
	}

	var maints []*domain.MaintenanceEvent
	maintRows, err := r.db.QueryContext(ctx, `
		SELECT id, COALESCE(sap_order_number, ''), order_type, start_timestamp, duration_hours,
		       COALESCE(description, ''), COALESCE(symptom, ''), source
		FROM maintenance_events
		WHERE station_code = $1
		ORDER BY start_timestamp DESC
		LIMIT 10
	`, door.StationCode)
	if err == nil {
		defer maintRows.Close()
		for maintRows.Next() {
			var m domain.MaintenanceEvent
			if err := maintRows.Scan(&m.ID, &m.SAPOrderNumber, &m.OrderType, &m.StartTimestamp, &m.DurationHours, &m.Description, &m.Symptom, &m.Source); err == nil {
				maints = append(maints, &m)
			}
		}
	}

	var inspections []*domain.Inspection
	insRows, err := r.db.QueryContext(ctx, `
		SELECT id, priority, status, COALESCE(notes, ''), COALESCE(scheduled_for, ''), completed_at
		FROM inspections
		WHERE station_code = $1 AND platform = $2 AND door_number = $3
		ORDER BY created_at DESC
		LIMIT 10
	`, door.StationCode, door.Platform, door.DoorNumber)
	if err == nil {
		defer insRows.Close()
		for insRows.Next() {
			var i domain.Inspection
			var schedStr, notesStr string
			var compAt *time.Time
			if err := insRows.Scan(&i.ID, &i.Priority, &i.Status, &notesStr, &schedStr, &compAt); err == nil {
				i.Notes = notesStr
				i.ScheduledFor = schedStr
				i.CompletedAt = compAt
				inspections = append(inspections, &i)
			}
		}
	}

	return &domain.DoorTimeline{
		Door:            door,
		StationName:     stationName,
		WillFail:        willFail,
		RecentAlarms:    alarms,
		Maintenances:    maints,
		Inspections:     inspections,
		LastEvaluatedAt: lastEvalAt,
	}, nil
}

func (r *PostgresDashboardRepository) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.platform, d.door_number, d.sap_tag, d.status,
		       COALESCE((
		           SELECT p.will_fail
		           FROM predictions p
		           WHERE p.station_code = s.code AND p.platform = d.platform AND p.door_number = d.door_number
		             AND p.run_id = (SELECT id FROM prediction_runs ORDER BY created_at DESC LIMIT 1)
		           LIMIT 1
		       ), FALSE) as will_fail
		FROM doors d
		JOIN stations s ON d.station_id = s.id
		WHERE s.code = $1
		ORDER BY d.platform ASC, d.door_number ASC
	`, stationCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.DoorStatusItem
	for rows.Next() {
		var item domain.DoorStatusItem
		if err := rows.Scan(&item.ID, &item.Platform, &item.DoorNumber, &item.SAPTag, &item.Status, &item.WillFail); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}

	return list, nil
}
