package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type lateInvalidParser struct{ fail bool }

func (p lateInvalidParser) StreamAlarms(_ context.Context, _ io.Reader, _ string, yield func(port.RawAlarmRecord) error) error {
	for i := 0; i < 501; i++ {
		if err := yield(port.RawAlarmRecord{Tag: "PSD-FAL-P1", StationCode: "FAL", Platform: "P1", DoorNumber: 1, AlarmType: "Falha", EventTimestamp: "2026-09-01 10:00:00"}); err != nil {
			return err
		}
	}
	if p.fail {
		return errors.New("malformed row after first batch")
	}
	return nil
}
func (p lateInvalidParser) StreamMaintenance(_ context.Context, _ io.Reader, _, _ string, yield func(port.RawMaintenanceRecord) error) error {
	for i := 0; i < 501; i++ {
		if err := yield(port.RawMaintenanceRecord{SAPOrderNumber: uuid.NewString(), OrderType: "ZM01", StationCode: "FAL", StartTimestamp: "2026-09-01 10:00:00", DurationHours: 1}); err != nil {
			return err
		}
	}
	if p.fail {
		return errors.New("malformed row after first batch")
	}
	return nil
}

func TestStreamingIngestionCommitsAllBatchesOrRollsBack(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	runTestMigrations(t, db)
	for _, kind := range []port.IngestionJobType{port.IngestionJobAlarms, port.IngestionJobMaintenance} {
		for _, fail := range []bool{false, true} {
			name := string(kind) + "/success"
			if fail {
				name = string(kind) + "/rollback"
			}
			t.Run(name, func(t *testing.T) {
				repo := repository.NewPostgresAlarmRepository(db)
				id := uuid.NewString()
				if err := repo.CreateUpload(ctx, &domain.FileUpload{ID: id, FileName: "file.csv", FileType: string(kind), FileHash: uuid.NewString(), Status: "processing"}); err != nil {
					t.Fatal(err)
				}
				raw := storage.NewLocalTempStorage()
				key, _, err := raw.Store(ctx, "file.csv", strings.NewReader("fixture"))
				if err != nil {
					t.Fatal(err)
				}
				defer raw.Delete(context.Background(), key)
				parser := lateInvalidParser{fail: fail}
				var uc interface {
					ProcessJob(context.Context, port.IngestionJob) error
				}
				table := "telemetry_alarms"
				if kind == port.IngestionJobAlarms {
					uc = usecase.NewIngestAlarmsUseCaseWithQueue(repo, repo, raw, parser, nil, repository.NewPostgresUnitOfWork(db), nil)
				} else {
					table = "maintenance_events"
					uc = usecase.NewIngestMaintenanceUseCaseWithQueue(repository.NewPostgresMaintenanceRepository(db), repo, raw, parser, nil, repository.NewPostgresUnitOfWork(db), nil)
				}
				if err := uc.ProcessJob(ctx, port.IngestionJob{Type: kind, UploadID: id, StorageKey: key, FileName: "file.csv"}); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(key); !os.IsNotExist(err) {
					t.Fatalf("temporary file survived ingestion finalization: %v", err)
				}
				var count int
				// table is selected only from the two constants above.
				if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+table+" WHERE upload_id=$1", id).Scan(&count); err != nil {
					t.Fatal(err)
				}
				upload, err := repo.GetUploadByID(ctx, id)
				if err != nil {
					t.Fatal(err)
				}
				if fail {
					if count != 0 || upload.Status != "failed" {
						t.Fatalf("partial data survived rollback: count=%d status=%s", count, upload.Status)
					}
				} else {
					if count != 501 || upload.Status != "completed" || upload.RowsInserted != 501 {
						t.Fatalf("batches not committed: count=%d upload=%+v", count, upload)
					}
				}
			})
		}
	}
}
