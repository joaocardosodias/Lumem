package repository

import (
	"context"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.PredictionRepository = (*SQLitePredictionRepository)(nil)

type SQLitePredictionRepository struct {
	db DBExecutor
}

func NewSQLitePredictionRepository(db DBExecutor) *SQLitePredictionRepository {
	return &SQLitePredictionRepository{db: db}
}

func (r *SQLitePredictionRepository) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.station_id, s.code, d.platform, d.door_number, d.sap_tag, d.status, d.created_at
		FROM doors d
		JOIN stations s ON d.station_id = s.id
		WHERE d.status = 'active'
		ORDER BY d.station_id ASC, d.platform ASC, d.door_number ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var doors []domain.Door
	for rows.Next() {
		var d domain.Door
		var createdAtStr string
		if err := rows.Scan(&d.ID, &d.StationID, &d.StationCode, &d.Platform, &d.DoorNumber, &d.SAPTag, &d.Status, &createdAtStr); err != nil {
			return nil, err
		}
		d.CreatedAt = parseTime(createdAtStr)
		doors = append(doors, d)
	}
	return doors, rows.Err()
}

func (r *SQLitePredictionRepository) SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	now := time.Now()
	run.CreatedAt = now

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO prediction_runs (horizon_days, total_doors_evaluated, total_doors_at_risk, created_at)
		VALUES (?, ?, ?, ?)
	`, run.HorizonDays, run.TotalDoorsEvaluated, run.TotalDoorsAtRisk, now.Format("2006-01-02 15:04:05"))
	if err != nil {
		return err
	}

	runID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	run.ID = runID

	stmt, err := r.db.PrepareContext(ctx, `
		INSERT INTO predictions (run_id, station_code, platform, door_number, will_fail, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		willFailInt := 0
		if item.WillFail {
			willFailInt = 1
		}

		_, err := stmt.ExecContext(ctx,
			runID,
			item.StationCode,
			item.Platform,
			item.DoorNumber,
			willFailInt,
			now.Format("2006-01-02 15:04:05"),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *SQLitePredictionRepository) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	var run domain.PredictionRun
	var createdAtStr string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, horizon_days, total_doors_evaluated, total_doors_at_risk, created_at
		FROM prediction_runs
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&run.ID, &run.HorizonDays, &run.TotalDoorsEvaluated, &run.TotalDoorsAtRisk, &createdAtStr)
	if err != nil {
		return nil, err
	}
	run.CreatedAt = parseTime(createdAtStr)

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, run_id, station_code, platform, door_number, will_fail
		FROM predictions
		WHERE run_id = ? AND will_fail = 1
		ORDER BY station_code ASC, platform ASC, door_number ASC
	`, run.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p domain.PredictionItem
			var willFailInt int
			if err := rows.Scan(&p.ID, &p.RunID, &p.StationCode, &p.Platform, &p.DoorNumber, &willFailInt); err == nil {
				p.WillFail = (willFailInt == 1)
				run.Predictions = append(run.Predictions, p)
			}
		}
	}

	return &run, nil
}
