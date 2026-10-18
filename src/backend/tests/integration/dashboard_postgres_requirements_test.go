package integration_test

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
)

func TestRF10_PostgresIndicatorsAndInspectionDeduplication(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL absent; aggregation and inspection deduplication not executed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	exec := func(query string) {
		t.Helper()
		if _, err := tx.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	// Temporary tables shadow public tables on this connection; rollback removes fixtures.
	for _, q := range []string{
		`CREATE TEMP TABLE doors (status text) ON COMMIT DROP`,
		`CREATE TEMP TABLE stations (code text) ON COMMIT DROP`,
		`CREATE TEMP TABLE prediction_runs (id text, total_doors_at_risk integer, is_experiment boolean, published_at timestamptz) ON COMMIT DROP`,
		`CREATE TEMP TABLE predictions (id text, run_id text, station_code text, will_fail boolean) ON COMMIT DROP`,
		`CREATE TEMP TABLE inspections (id text PRIMARY KEY, prediction_id text, station_code text, platform text, door_number integer, status text, notes text, completed_at timestamptz, created_at timestamptz DEFAULT now(), updated_at timestamptz DEFAULT now()) ON COMMIT DROP`,
		`CREATE UNIQUE INDEX rf_open_inspection ON inspections (station_code,platform,door_number) WHERE status IN ('pending','in_progress')`,
	} {
		exec(q)
	}
	repo := repository.NewPostgresDashboardRepository(tx)
	empty, err := repo.GetSummary(ctx)
	if err != nil || !reflect.DeepEqual(empty, &domain.DashboardSummary{}) {
		t.Fatalf("empty summary: %+v err=%v", empty, err)
	}
	for _, q := range []string{
		`INSERT INTO doors SELECT 'active' FROM generate_series(1,8)`,
		`INSERT INTO doors VALUES ('inactive')`,
		`INSERT INTO stations VALUES ('FAL'),('PIN'),('BUT')`,
		`INSERT INTO prediction_runs VALUES ('old',9,FALSE,'2026-10-01'),('current',4,FALSE,'2026-10-02'),('experiment',7,TRUE,'2026-10-03'),('unpublished',8,FALSE,NULL)`,
		`INSERT INTO predictions VALUES ('1','current','FAL',TRUE),('2','current','FAL',TRUE),('3','current','FAL',TRUE),('4','current','PIN',TRUE),('5','current','BUT',FALSE),('6','old','BUT',TRUE),('7','experiment','BUT',TRUE),('8','unpublished','BUT',TRUE)`,
		`INSERT INTO inspections(id,station_code,platform,door_number,status) VALUES ('p1','PIN','P1',1,'pending'),('p2','PIN','P1',2,'pending'),('p3','PIN','P1',3,'pending'),('c1','PIN','P1',4,'completed'),('c2','PIN','P1',5,'completed'),('i1','PIN','P1',6,'in_progress'),('x1','PIN','P1',7,'canceled')`,
	} {
		exec(q)
	}
	want := &domain.DashboardSummary{TotalDoors: 8, DoorsAtRisk: 4, PendingInspections: 3, CompletedInspections: 2, CriticalStations: []domain.StationRiskSummary{{StationCode: "FAL", DoorsAtRisk: 3}, {StationCode: "PIN", DoorsAtRisk: 1}}}
	got, err := repo.GetSummary(ctx)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("aggregate summary=%+v err=%v want=%+v", got, err, want)
	}
	insRepo := repository.NewPostgresInspectionRepository(tx)
	for index, status := range []string{"pending", "in_progress"} {
		firstPrediction, laterPrediction := "first-prediction", "later-prediction"
		first := &domain.Inspection{PredictionID: &firstPrediction, StationCode: "FAL", Platform: "P2", DoorNumber: 23 + index, Status: status, Notes: "Original"}
		if err := insRepo.Create(ctx, first); err != nil {
			t.Fatal(err)
		}
		duplicate := &domain.Inspection{PredictionID: &laterPrediction, StationCode: "FAL", Platform: "P2", DoorNumber: 23 + index, Status: "pending", Notes: "New"}
		if err := insRepo.Create(ctx, duplicate); err != nil {
			t.Fatal(err)
		}
		if duplicate.ID != first.ID || duplicate.Status != status || duplicate.PredictionID == nil || *duplicate.PredictionID != firstPrediction || duplicate.Notes != "Original" {
			t.Fatalf("open inspection not reused: %+v", duplicate)
		}
		now := time.Now()
		if err := insRepo.UpdateStatus(ctx, first.ID, "completed", "Done", &now); err != nil {
			t.Fatal(err)
		}
		next := &domain.Inspection{PredictionID: &laterPrediction, StationCode: "FAL", Platform: "P2", DoorNumber: 23 + index, Status: "pending"}
		if err := insRepo.Create(ctx, next); err != nil {
			t.Fatal(err)
		}
		if next.ID == first.ID || next.PredictionID == nil || *next.PredictionID != laterPrediction {
			t.Fatal("new inspection was not created after completion")
		}
	}
}
