package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func testUserRepo(t *testing.T, ctx context.Context, db *sql.DB) {
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

	newUser := &domain.User{
		ID:           uuid.New().String(),
		Name:         "Novo Usuario Teste",
		Email:        "novo_" + uuid.New().String() + "@motiva.com.br",
		PasswordHash: "hashed",
		Role:         "operator",
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := userRepo.Create(ctx, newUser); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = userRepo.GetByEmail(ctx, "inexistente@motiva.com.br")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("esperava ErrUserNotFound")
	}
	_, err = userRepo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("esperava ErrUserNotFound")
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := userRepo.GetByEmail(canceledCtx, "any@test.com"); err == nil {
		t.Fatalf("esperava erro com context cancelado em GetByEmail")
	}
	if _, err := userRepo.GetByID(canceledCtx, "any-id"); err == nil {
		t.Fatalf("esperava erro com context cancelado em GetByID")
	}
}

func testAlarmRepo(t *testing.T, ctx context.Context, db *sql.DB) string {
	alarmRepo := repository.NewPostgresAlarmRepository(db)
	uploadID := uuid.New().String()
	upload := &domain.FileUpload{
		ID:            uploadID,
		FileName:      "telemetria_test.xlsx",
		FileType:      "alarms",
		FileHash:      "hash_" + uuid.New().String(),
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

	byHash, err := alarmRepo.GetUploadByHash(ctx, upload.FileHash)
	if err != nil || byHash.ID != uploadID {
		t.Fatalf("GetUploadByHash failed: %v", err)
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

	dupUpload := &domain.FileUpload{
		ID:       uuid.New().String(),
		FileName: "dup.xlsx",
		FileType: "alarms",
		FileHash: upload.FileHash,
	}
	if err := alarmRepo.CreateUpload(ctx, dupUpload); !errors.Is(err, domain.ErrDuplicateUpload) {
		t.Fatalf("esperava ErrDuplicateUpload, obteve %v", err)
	}

	_, err = alarmRepo.GetUploadByID(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrUploadNotFound) {
		t.Fatalf("esperava ErrUploadNotFound")
	}

	if err := alarmRepo.BatchInsertAlarms(ctx, nil); err != nil {
		t.Fatalf("esperava nil para batch vazio de alarmes")
	}

	testAlarmRepoEdgeCases(t, ctx, db, uploadID, alarms)

	return uploadID
}

func testAlarmRepoEdgeCases(t *testing.T, ctx context.Context, db *sql.DB, uploadID string, alarms []*domain.Alarm) {
	alarmRepo := repository.NewPostgresAlarmRepository(db)

	alarmNoID := &domain.FileUpload{
		FileName: "no-id.xlsx",
		FileType: "alarms",
		FileHash: "hash_" + uuid.New().String(),
	}
	if err := alarmRepo.CreateUpload(ctx, alarmNoID); err != nil {
		t.Fatalf("CreateUpload without ID failed: %v", err)
	}
	if alarmNoID.ID == "" {
		t.Fatalf("esperava ID gerado")
	}

	alarmsMulti := []*domain.Alarm{
		{
			UploadID:        uploadID,
			StationCode:     "PIN",
			Platform:        "P1",
			DoorNumber:      1,
			AlarmType:       "Falha 1",
			DurationSeconds: 2,
			EventTimestamp:  "2026-09-01T10:01:00Z",
			Source:          "spreadsheet",
		},
		{
			UploadID:        uploadID,
			StationCode:     "PIN",
			Platform:        "P1",
			DoorNumber:      1,
			AlarmType:       "Falha 2",
			DurationSeconds: 3,
			EventTimestamp:  "2026-09-01T10:02:00Z",
			Source:          "spreadsheet",
		},
	}
	if err := alarmRepo.BatchInsertAlarms(ctx, alarmsMulti); err != nil {
		t.Fatalf("BatchInsertAlarms multi failed: %v", err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := alarmRepo.CreateUpload(canceledCtx, &domain.FileUpload{}); err == nil {
		t.Fatalf("esperava erro em CreateUpload cancelado")
	}
	if _, err := alarmRepo.GetUploadByID(canceledCtx, uploadID); err == nil {
		t.Fatalf("esperava erro em GetUploadByID cancelado")
	}
	if _, err := alarmRepo.GetUploadByHash(canceledCtx, "nonexistent"); err == nil {
		t.Fatalf("esperava erro em GetUploadByHash cancelado")
	}
	if err := alarmRepo.BatchInsertAlarms(canceledCtx, alarms); err == nil {
		t.Fatalf("esperava erro em BatchInsertAlarms cancelado")
	}
}

func testMaintenanceRepo(t *testing.T, ctx context.Context, db *sql.DB, uploadID string) {
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

	maintSingle := &domain.MaintenanceEvent{
		ID:                   uuid.New().String(),
		UploadID:             uploadID,
		SAPOrderNumber:       "ORD-MANUAL-001",
		OrderType:            "ZM03",
		StationCode:          "PIN",
		InstallationLocation: "PLATAFORMA 1",
		StartTimestamp:       "2026-09-01T12:00:00Z",
		DurationHours:        1.0,
		Description:          "Criacao manual de teste",
		Source:               "manual",
		CreatedAt:            time.Now().UTC(),
	}
	if err := maintRepo.Create(ctx, maintSingle); err != nil {
		t.Fatalf("maintRepo.Create failed: %v", err)
	}

	list, err := maintRepo.List(ctx, 10, 0)
	if err != nil || len(list) == 0 {
		t.Fatalf("maintRepo.List failed: %v", err)
	}

	maintNoID := &domain.MaintenanceEvent{
		OrderType:      "ZM01",
		StartTimestamp: "2026-09-01T10:00:00Z",
		DurationHours:  1.0,
		Source:         "manual",
	}
	if err := maintRepo.Create(ctx, maintNoID); err != nil {
		t.Fatalf("maintRepo.Create without ID failed: %v", err)
	}
	if maintNoID.ID == "" {
		t.Fatalf("esperava ID gerado")
	}

	if _, err := maintRepo.List(ctx, -1, -1); err != nil {
		t.Fatalf("maintRepo.List negative bounds failed: %v", err)
	}

	if err := maintRepo.BatchInsertMaintenance(ctx, nil); err != nil {
		t.Fatalf("esperava nil para batch vazio de manutencoes")
	}

	maintMulti := []*domain.MaintenanceEvent{
		{
			UploadID:       uploadID,
			SAPOrderNumber: "ORD-M1",
			OrderType:      "ZM01",
			StationCode:    "PIN",
			StartTimestamp: "2026-09-01T10:00:00Z",
			DurationHours:  1.0,
			Source:         "spreadsheet",
		},
		{
			UploadID:       uploadID,
			SAPOrderNumber: "ORD-M2",
			OrderType:      "ZM01",
			StationCode:    "PIN",
			StartTimestamp: "2026-09-01T11:00:00Z",
			DurationHours:  1.0,
			Source:         "spreadsheet",
		},
	}
	if err := maintRepo.BatchInsertMaintenance(ctx, maintMulti); err != nil {
		t.Fatalf("BatchInsertMaintenance multi failed: %v", err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := maintRepo.BatchInsertMaintenance(canceledCtx, maints); err == nil {
		t.Fatalf("esperava erro em BatchInsertMaintenance cancelado")
	}
	if _, err := maintRepo.List(canceledCtx, 10, 0); err == nil {
		t.Fatalf("esperava erro em List cancelado")
	}
}

func testCleanupUploads(t *testing.T, ctx context.Context, db *sql.DB, uploadID string) {
	alarmRepo := repository.NewPostgresAlarmRepository(db)
	maintRepo := repository.NewPostgresMaintenanceRepository(db)

	if err := alarmRepo.DeleteByUploadID(ctx, uploadID); err != nil {
		t.Fatalf("alarmRepo.DeleteByUploadID failed: %v", err)
	}
	if err := maintRepo.DeleteByUploadID(ctx, uploadID); err != nil {
		t.Fatalf("maintRepo.DeleteByUploadID failed: %v", err)
	}
}

func testPredictionUOW(t *testing.T, ctx context.Context, db *sql.DB) (string, string) {
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
	err := uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		_ = store.Inspections()
		_ = store.Maintenances()
		_ = store.Alarms()
		_ = store.Uploads()
		return store.Predictions().SaveRun(ctx, run, []domain.PredictionItem{item})
	})
	if err != nil {
		t.Fatalf("uow.Do failed: %v", err)
	}
	return runID, predID
}

func testInspectionRepo(t *testing.T, ctx context.Context, db *sql.DB, predID string) {
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
	if fetchedInsp.PredictionID == nil || *fetchedInsp.PredictionID != predID {
		t.Fatalf("esperava PredictionID %s, obteve %v", predID, fetchedInsp.PredictionID)
	}

	list, err := inspRepo.List(ctx, 10, 0, "pending")
	if err != nil || len(list) == 0 {
		t.Fatalf("inspRepo.List failed: %v", err)
	}
	if list[0].PredictionID == nil || *list[0].PredictionID != predID {
		t.Fatalf("esperava PredictionID %s na listagem, obteve %v", predID, list[0].PredictionID)
	}

	now := time.Now().UTC()
	if err := inspRepo.UpdateStatus(ctx, inspID, "completed", "Finalizado com sucesso", &now); err != nil {
		t.Fatalf("inspRepo.UpdateStatus failed: %v", err)
	}

	testInspectionRepoEdgeCases(t, ctx, db, inspID)
}

func testInspectionRepoEdgeCases(t *testing.T, ctx context.Context, db *sql.DB, inspID string) {
	inspRepo := repository.NewPostgresInspectionRepository(db)

	inspNoID := &domain.Inspection{
		StationCode: "PIN",
		Platform:    "P1",
		DoorNumber:  1,
		Priority:    "high",
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := inspRepo.Create(ctx, inspNoID); err != nil {
		t.Fatalf("inspRepo.Create without ID failed: %v", err)
	}
	if inspNoID.ID == "" {
		t.Fatalf("esperava ID gerado")
	}

	_, err := inspRepo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrInspectionNotFound) {
		t.Fatalf("esperava ErrInspectionNotFound")
	}

	err = inspRepo.UpdateStatus(ctx, "00000000-0000-0000-0000-000000000000", "completed", "notes", nil)
	if !errors.Is(err, domain.ErrInspectionNotFound) {
		t.Fatalf("esperava ErrInspectionNotFound ao atualizar id inexistente")
	}

	if _, err := inspRepo.List(ctx, -1, -1, ""); err != nil {
		t.Fatalf("inspRepo.List negative bounds and empty filter failed: %v", err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := inspRepo.UpdateStatus(canceledCtx, inspID, "completed", "", nil); err == nil {
		t.Fatalf("esperava erro em UpdateStatus cancelado")
	}
	if _, err := inspRepo.GetByID(canceledCtx, inspID); err == nil {
		t.Fatalf("esperava erro em GetByID cancelado")
	}
	if _, err := inspRepo.List(canceledCtx, 10, 0, ""); err == nil {
		t.Fatalf("esperava erro em List cancelado")
	}
}

func testDashboardRepo(t *testing.T, ctx context.Context, db *sql.DB) {
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

	timeline145, err := dashRepo.GetDoorTimeline(ctx, 145)
	if err != nil {
		t.Fatalf("dashRepo.GetDoorTimeline 145 failed: %v", err)
	}
	if timeline145.Door.ID != 145 {
		t.Fatalf("unexpected door id: %d", timeline145.Door.ID)
	}

	stationDoors, err := dashRepo.GetStationDoorsStatus(ctx, "PIN")
	if err != nil || len(stationDoors) == 0 {
		t.Fatalf("dashRepo.GetStationDoorsStatus failed: %v", err)
	}

	predRepo := repository.NewPostgresPredictionRepository(db)
	doors, err := predRepo.ListDoorsForInference(ctx)
	if err != nil || len(doors) == 0 {
		t.Fatalf("predRepo.ListDoorsForInference failed: %v", err)
	}

	latestRun, err := predRepo.GetLatestRun(ctx)
	if err != nil || latestRun == nil {
		t.Fatalf("predRepo.GetLatestRun failed: %v", err)
	}

	_, err = dashRepo.GetDoorTimeline(ctx, 999999)
	if !errors.Is(err, domain.ErrDoorNotFound) {
		t.Fatalf("esperava ErrDoorNotFound para porta inexistente")
	}

	testDashboardRepoCachedAndCanceled(t, ctx, dashRepo)
}

func testDashboardRepoCachedAndCanceled(t *testing.T, ctx context.Context, dashRepo *repository.PostgresDashboardRepository) {
	cachedRepo := repository.NewCachedDashboardRepository(dashRepo, 10*time.Minute)
	if _, err := cachedRepo.GetSummary(ctx); err != nil {
		t.Fatalf("cachedRepo.GetSummary failed: %v", err)
	}
	if _, err := cachedRepo.GetDoorTimeline(ctx, 1); err != nil {
		t.Fatalf("cachedRepo.GetDoorTimeline failed: %v", err)
	}
	if _, err := cachedRepo.GetStationDoorsStatus(ctx, "PIN"); err != nil {
		t.Fatalf("cachedRepo.GetStationDoorsStatus failed: %v", err)
	}
	cachedRepo.Invalidate()

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := dashRepo.GetSummary(canceledCtx); err == nil {
		t.Fatalf("esperava erro em GetSummary cancelado")
	}
	if _, err := dashRepo.GetDoorTimeline(canceledCtx, 1); err == nil {
		t.Fatalf("esperava erro em GetDoorTimeline cancelado")
	}
	if _, err := dashRepo.GetStationDoorsStatus(canceledCtx, "PIN"); err == nil {
		t.Fatalf("esperava erro em GetStationDoorsStatus cancelado")
	}
}

func testPredictionRepoExtra(t *testing.T, ctx context.Context, db *sql.DB) {
	predRepo := repository.NewPostgresPredictionRepository(db)

	_, _ = db.ExecContext(ctx, "DELETE FROM inspections; DELETE FROM predictions; DELETE FROM prediction_runs")
	if _, err := predRepo.GetLatestRun(ctx); !errors.Is(err, domain.ErrNoPredictions) {
		t.Fatalf("esperava ErrNoPredictions")
	}

	runNoID := &domain.PredictionRun{
		HorizonDays:         7,
		TotalDoorsEvaluated: 1,
		TotalDoorsAtRisk:    1,
	}
	itemNoID := domain.PredictionItem{
		StationCode: "PIN",
		Platform:    "P1",
		DoorNumber:  1,
		WillFail:    true,
	}
	if err := predRepo.SaveRun(ctx, runNoID, []domain.PredictionItem{itemNoID}); err != nil {
		t.Fatalf("predRepo.SaveRun without IDs failed: %v", err)
	}
	if runNoID.ID == "" {
		t.Fatalf("esperava run ID gerado")
	}

	runWithItems, err := predRepo.GetLatestRun(ctx)
	if err != nil || len(runWithItems.Predictions) == 0 {
		t.Fatalf("esperava run com items preenchidos, obteve %+v", runWithItems)
	}

	dupItemID := uuid.New().String()
	runFail := &domain.PredictionRun{ID: uuid.New().String(), HorizonDays: 7}
	itemsFail := []domain.PredictionItem{
		{ID: dupItemID, StationCode: "PIN", Platform: "P1", DoorNumber: 1},
		{ID: dupItemID, StationCode: "PIN", Platform: "P1", DoorNumber: 1},
	}
	if err := predRepo.SaveRun(ctx, runFail, itemsFail); err == nil {
		t.Fatalf("esperava erro de item duplicado em SaveRun")
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := predRepo.ListDoorsForInference(canceledCtx); err == nil {
		t.Fatalf("esperava erro em ListDoorsForInference cancelado")
	}
	if err := predRepo.SaveRun(canceledCtx, runNoID, []domain.PredictionItem{itemNoID}); err == nil {
		t.Fatalf("esperava erro em SaveRun cancelado")
	}
	if _, err := predRepo.GetLatestRun(canceledCtx); err == nil {
		t.Fatalf("esperava erro em GetLatestRun cancelado")
	}
}

func testUOWRollback(t *testing.T, ctx context.Context, db *sql.DB) {
	uow := repository.NewPostgresUnitOfWork(db)
	err := uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		return errors.New("simulated error for rollback")
	})
	if err == nil || err.Error() != "simulated error for rollback" {
		t.Fatalf("esperava erro de rollback")
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := uow.Do(canceledCtx, func(store port.UnitOfWorkStore) error {
		return nil
	}); err == nil {
		t.Fatalf("esperava erro em uow.Do com context cancelado")
	}

	commitErr := uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		_ = store.Alarms().BatchInsertAlarms(ctx, []*domain.Alarm{
			{ID: uuid.New().String(), UploadID: "00000000-0000-0000-0000-000000000000"},
		})
		return nil
	})
	if commitErr == nil {
		t.Fatalf("esperava erro ao confirmar transacao com estado abortado")
	}
}

func runTestMigrations(t *testing.T, db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("failed to create migrate driver: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "postgres", driver)
	if err != nil {
		t.Fatalf("failed to init migrations: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("failed to apply migrations: %v", err)
	}
}

func TestPostgresRepositories(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5439/postgres?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("cannot connect to postgres: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping postgres repository tests, database unreachable: %v", err)
	}

	runTestMigrations(t, db)
	testUserRepo(t, ctx, db)
	uploadID := testAlarmRepo(t, ctx, db)
	testMaintenanceRepo(t, ctx, db, uploadID)
	_, predID := testPredictionUOW(t, ctx, db)
	testInspectionRepo(t, ctx, db, predID)
	testDashboardRepo(t, ctx, db)
	testPredictionRepoExtra(t, ctx, db)
	testCleanupUploads(t, ctx, db, uploadID)
	testUOWRollback(t, ctx, db)
}
