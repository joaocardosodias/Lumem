package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/g04/backend/internal/domain"
)

var _ domain.DashboardRepository = (*SQLiteDashboardRepository)(nil)

type SQLiteDashboardRepository struct {
	db *sql.DB
}

func NewSQLiteDashboardRepository(db *sql.DB) *SQLiteDashboardRepository {
	return &SQLiteDashboardRepository{db: db}
}

func (r *SQLiteDashboardRepository) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	summary := &domain.DashboardSummary{
		CriticalStations: make([]domain.StationRiskSummary, 0),
	}

	_ = r.db.QueryRowContext(ctx, `SELECT count(*) FROM doors`).Scan(&summary.TotalDoors)

	var latestRunID int64
	err := r.db.QueryRowContext(ctx, `SELECT id, total_doors_at_risk FROM prediction_runs ORDER BY id DESC LIMIT 1`).
		Scan(&latestRunID, &summary.DoorsAtRisk)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	_ = r.db.QueryRowContext(ctx, `SELECT count(*) FROM inspections WHERE status = 'pending'`).
		Scan(&summary.PendingInspections)

	_ = r.db.QueryRowContext(ctx, `SELECT count(*) FROM inspections WHERE status = 'completed'`).
		Scan(&summary.CompletedInspections)

	if latestRunID > 0 {
		rows, err := r.db.QueryContext(ctx, `
			SELECT station_code, count(*)
			FROM predictions
			WHERE run_id = ? AND will_fail = 1
			GROUP BY station_code
			ORDER BY count(*) DESC
		`, latestRunID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var s domain.StationRiskSummary
				if err := rows.Scan(&s.StationCode, &s.DoorsAtRisk); err == nil {
					summary.CriticalStations = append(summary.CriticalStations, s)
				}
			}
		}
	}

	return summary, nil
}

func (r *SQLiteDashboardRepository) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	var timeline domain.DoorTimeline
	timeline.RecentAlarms = make([]*domain.Alarm, 0)
	timeline.Maintenances = make([]*domain.MaintenanceEvent, 0)
	timeline.Inspections = make([]*domain.Inspection, 0)

	var stationCode string
	err := r.db.QueryRowContext(ctx, `
		SELECT d.id, d.station_id, d.platform, d.door_number, d.sap_tag, d.status, s.code, s.name
		FROM doors d
		JOIN stations s ON d.station_id = s.id
		WHERE d.id = ?
		LIMIT 1
	`, doorID).Scan(
		&timeline.Door.ID,
		&timeline.Door.StationID,
		&timeline.Door.Platform,
		&timeline.Door.DoorNumber,
		&timeline.Door.SAPTag,
		&timeline.Door.Status,
		&stationCode,
		&timeline.StationName,
	)
	if err != nil {
		return nil, err
	}

	var latestRunID int64
	var runCreatedAtStr string
	err = r.db.QueryRowContext(ctx, `
		SELECT id, created_at FROM prediction_runs ORDER BY id DESC LIMIT 1
	`).Scan(&latestRunID, &runCreatedAtStr)
	if err == nil && latestRunID > 0 {
		var willFailInt int
		_ = r.db.QueryRowContext(ctx, `
			SELECT will_fail FROM predictions
			WHERE run_id = ? AND station_code = ? AND platform = ? AND door_number = ?
			LIMIT 1
		`, latestRunID, stationCode, timeline.Door.Platform, timeline.Door.DoorNumber).Scan(&willFailInt)
		timeline.WillFail = (willFailInt == 1)

		t := parseTime(runCreatedAtStr)
		if !t.IsZero() {
			timeline.LastEvaluatedAt = &t
		}
	}

	alarmRows, err := r.db.QueryContext(ctx, `
		SELECT id, upload_id, tag, station_code, platform, door_number,
		       alarm_type, duration_seconds, event_timestamp, IFNULL(normalization_timestamp, ''), source, created_at
		FROM telemetry_alarms
		WHERE station_code = ? AND door_number = ?
		ORDER BY id DESC
		LIMIT 15
	`, stationCode, timeline.Door.DoorNumber)
	if err == nil {
		defer alarmRows.Close()
		for alarmRows.Next() {
			var a domain.Alarm
			var createdAtStr string
			if err := alarmRows.Scan(
				&a.ID, &a.UploadID, &a.Tag, &a.StationCode, &a.Platform, &a.DoorNumber,
				&a.AlarmType, &a.DurationSeconds, &a.EventTimestamp, &a.NormalizationTimestamp,
				&a.Source, &createdAtStr,
			); err == nil {
				a.CreatedAt = parseTime(createdAtStr)
				timeline.RecentAlarms = append(timeline.RecentAlarms, &a)
			}
		}
	}

	maintRows, err := r.db.QueryContext(ctx, `
		SELECT id, IFNULL(upload_id, 0), IFNULL(sap_order_number, ''), order_type,
		       IFNULL(station_code, ''), IFNULL(installation_location, ''),
		       start_timestamp, IFNULL(end_timestamp, ''), duration_hours,
		       IFNULL(description, ''), IFNULL(symptom, ''), IFNULL(cause, ''),
		       IFNULL(action_taken, ''), source, created_at
		FROM maintenance_events
		WHERE station_code = ? OR installation_location LIKE '%' || ? || '%'
		ORDER BY id DESC
		LIMIT 10
	`, stationCode, timeline.Door.SAPTag)
	if err == nil {
		defer maintRows.Close()
		for maintRows.Next() {
			var m domain.MaintenanceEvent
			var createdAtStr string
			if err := maintRows.Scan(
				&m.ID, &m.UploadID, &m.SAPOrderNumber, &m.OrderType,
				&m.StationCode, &m.InstallationLocation,
				&m.StartTimestamp, &m.EndTimestamp, &m.DurationHours,
				&m.Description, &m.Symptom, &m.Cause,
				&m.ActionTaken, &m.Source, &createdAtStr,
			); err == nil {
				m.CreatedAt = parseTime(createdAtStr)
				timeline.Maintenances = append(timeline.Maintenances, &m)
			}
		}
	}

	insRows, err := r.db.QueryContext(ctx, `
		SELECT id, prediction_id, station_code, platform, door_number,
		       status, priority, assigned_to, IFNULL(notes, ''), IFNULL(scheduled_for, ''),
		       completed_at, created_at, updated_at
		FROM inspections
		WHERE station_code = ? AND platform = ? AND door_number = ?
		ORDER BY id DESC
		LIMIT 5
	`, stationCode, timeline.Door.Platform, timeline.Door.DoorNumber)
	if err == nil {
		defer insRows.Close()
		for insRows.Next() {
			var ins domain.Inspection
			var completedAtStr sql.NullString
			var createdAtStr, updatedAtStr string
			if err := insRows.Scan(
				&ins.ID, &ins.PredictionID, &ins.StationCode, &ins.Platform, &ins.DoorNumber,
				&ins.Status, &ins.Priority, &ins.AssignedTo, &ins.Notes, &ins.ScheduledFor,
				&completedAtStr, &createdAtStr, &updatedAtStr,
			); err == nil {
				ins.CreatedAt = parseTime(createdAtStr)
				ins.UpdatedAt = parseTime(updatedAtStr)
				if completedAtStr.Valid {
					t := parseTime(completedAtStr.String)
					if !t.IsZero() {
						ins.CompletedAt = &t
					}
				}
				timeline.Inspections = append(timeline.Inspections, &ins)
			}
		}
	}

	return &timeline, nil
}

func (r *SQLiteDashboardRepository) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	var latestRunID int64
	_ = r.db.QueryRowContext(ctx, `SELECT id FROM prediction_runs ORDER BY id DESC LIMIT 1`).Scan(&latestRunID)

	query := `
		SELECT d.id, d.platform, d.door_number, d.sap_tag,
		       CASE 
		           WHEN EXISTS (
		               SELECT 1 FROM inspections i 
		               WHERE i.station_code = s.code AND i.platform = d.platform AND i.door_number = d.door_number 
		                 AND i.status IN ('pending', 'in_progress')
		           ) THEN 'under_inspection'
		           WHEN IFNULL(p.will_fail, 0) = 1 THEN 'at_risk'
		           ELSE 'healthy'
		       END AS operational_status,
		       CASE WHEN IFNULL(p.will_fail, 0) = 1 THEN 1 ELSE 0 END AS will_fail
		FROM doors d
		JOIN stations s ON d.station_id = s.id
		LEFT JOIN predictions p ON p.run_id = ? AND p.station_code = s.code AND p.platform = d.platform AND p.door_number = d.door_number
		WHERE s.code = ?
		ORDER BY d.platform ASC, d.door_number ASC
	`

	rows, err := r.db.QueryContext(ctx, query, latestRunID, stationCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*domain.DoorStatusItem, 0, 48)
	for rows.Next() {
		var item domain.DoorStatusItem
		var willFailInt int
		if err := rows.Scan(
			&item.ID,
			&item.Platform,
			&item.DoorNumber,
			&item.SAPTag,
			&item.Status,
			&willFailInt,
		); err != nil {
			return nil, err
		}
		item.WillFail = (willFailInt == 1)
		items = append(items, &item)
	}

	return items, rows.Err()
}
