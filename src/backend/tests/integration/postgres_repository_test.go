package integration_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresRepositories(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping postgres repository tests")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open postgres connection: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping postgres: %v", err)
	}

	userRepo := repository.NewPostgresUserRepository(db)
	user, err := userRepo.GetByEmail(ctx, "operador@motiva.com.br")
	if err != nil {
		t.Fatalf("expected seeded user from migration 2, got: %v", err)
	}
	if user.Email != "operador@motiva.com.br" {
		t.Fatalf("unexpected user email: %s", user.Email)
	}
	userByID, err := userRepo.GetByID(ctx, user.ID)
	if err != nil || userByID.ID != user.ID {
		t.Fatalf("failed to get user by id: %v", err)
	}

	alarmRepo := repository.NewPostgresAlarmRepository(db)
	uploadID := uuid.New().String()
	upload := &domain.FileUpload{
		ID:            uploadID,
		FileName:      "telemetria_test.xlsx",
		FileType:      "alarms",
		FileHash:      "hash_123456",
		RowsProcessed: 1,
		RowsInserted:  1,
		Status:        "processing",
		CreatedAt:     time.Now().UTC(),
	}
	if err := alarmRepo.CreateUpload(ctx, upload); err != nil {
		t.Fatalf("CreateUpload failed: %v", err)
	}

	completedAt := time.Now().UTC()
	upload.Status = "completed"
	upload.CompletedAt = &completedAt
	if err := alarmRepo.UpdateUpload(ctx, upload); err != nil {
		t.Fatalf("UpdateUpload failed: %v", err)
	}

	fetchedUpload, err := alarmRepo.GetUploadByID(ctx, uploadID)
	if err != nil || fetchedUpload.Status != "completed" {
		t.Fatalf("GetUploadByID failed: %v", err)
	}

	alarmID := uuid.New().String()
	alarms := []*domain.Alarm{
		{
			ID:              alarmID,
			UploadID:        uploadID,
			Tag:             "SP04-6-PSD-PIN-P1-PP01",
			StationCode:     "PIN",
			Platform:        "P1",
			DoorNumber:      1,
			AlarmType:       "Obstaculo",
			DurationSeconds: 4,
			EventTimestamp:  "2026-09-01T10:00:00Z",
			Source:          "spreadsheet",
		},
	}
	if err := alarmRepo.BatchInsertAlarms(ctx, alarms); err != nil {
		t.Fatalf("BatchInsertAlarms failed: %v", err)
	}

	if err := alarmRepo.ReconcileStaleUploads(ctx); err != nil {
		t.Fatalf("ReconcileStaleUploads failed: %v", err)
	}

	maintRepo := repository.NewPostgresMaintenanceRepository(db)
	maintID := uuid.New().String()
	maints := []*domain.MaintenanceEvent{
		{
			ID:                   maintID,
			UploadID:             uploadID,
			SAPOrderNumber:       "ORD-TEST-001",
			OrderType:            "ZM03",
			StationCode:          "PIN",
			InstallationLocation: "PLATAFORMA 1",
			StartTimestamp:       "2026-09-01T11:00:00Z",
			DurationHours:        2.0,
			Description:          "Inspecao preventiva geral",
			Source:               "spreadsheet",
		},
	}
	if err := maintRepo.BatchInsertMaintenance(ctx, maints); err != nil {
		t.Fatalf("BatchInsertMaintenance failed: %v", err)
	}

	uow := repository.NewPostgresUnitOfWork(db)
	runID := uuid.New().String()
	predID := uuid.New().String()
	run := &domain.PredictionRun{
		ID:                  runID,
		HorizonDays:         7,
		TotalDoorsEvaluated: 1,
		TotalDoorsAtRisk:    1,
	}
	item := domain.PredictionItem{
		ID:          predID,
		RunID:       runID,
		StationCode: "PIN",
		Platform:    "P1",
		DoorNumber:  1,
		WillFail:    true,
	}
	err = uow.Do(ctx, func(store domain.UnitOfWorkStore) error {
		return store.Predictions().SaveRun(ctx, run, []domain.PredictionItem{item})
	})
	if err != nil {
		t.Fatalf("uow.Do failed: %v", err)
	}

	inspRepo := repository.NewPostgresInspectionRepository(db)
	inspID := uuid.New().String()
	insp := &domain.Inspection{
		ID:           inspID,
		PredictionID: &predID,
		StationCode:  "PIN",
		Platform:     "P1",
		DoorNumber:   1,
		Status:       "pending",
		Priority:     "high",
		Notes:        "Inspecao gerada automaticamente pelo teste",
		ScheduledFor: "2026-09-10",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := inspRepo.Create(ctx, insp); err != nil {
		t.Fatalf("inspRepo.Create failed: %v", err)
	}

	fetchedInsp, err := inspRepo.GetByID(ctx, inspID)
	if err != nil || fetchedInsp.Status != "pending" {
		t.Fatalf("inspRepo.GetByID failed: %v", err)
	}

	list, err := inspRepo.List(ctx, 10, 0, "pending")
	if err != nil || len(list) == 0 {
		t.Fatalf("inspRepo.List failed: %v", err)
	}

	now := time.Now().UTC()
	if err := inspRepo.UpdateStatus(ctx, inspID, "completed", "Finalizado com sucesso", &now); err != nil {
		t.Fatalf("inspRepo.UpdateStatus failed: %v", err)
	}

	dashRepo := repository.NewPostgresDashboardRepository(db)
	summary, err := dashRepo.GetSummary(ctx)
	if err != nil {
		t.Fatalf("dashRepo.GetSummary failed: %v", err)
	}
	if summary.TotalDoors == 0 {
		t.Fatalf("expected doors from migration 8, got 0")
	}

	timeline, err := dashRepo.GetDoorTimeline(ctx, 1)
	if err != nil {
		t.Fatalf("dashRepo.GetDoorTimeline failed: %v", err)
	}
	if timeline.Door.ID != 1 {
		t.Fatalf("unexpected door id: %d", timeline.Door.ID)
	}

	stationDoors, err := dashRepo.GetStationDoorsStatus(ctx, "PIN")
	if err != nil || len(stationDoors) == 0 {
		t.Fatalf("dashRepo.GetStationDoorsStatus failed: %v", err)
	}
}
