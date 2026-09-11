package jobqueue

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type fakeCleanupInserter struct {
	key string
	err error
}

func (f *fakeCleanupInserter) Insert(ctx context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing deadline")
	}
	f.key = args.(CleanupArgs).StorageKey
	return &rivertype.JobInsertResult{}, f.err
}

type failingCleanupStorage struct{ err error }

func (*failingCleanupStorage) Store(context.Context, string, io.Reader) (string, string, error) {
	panic("unexpected Store")
}
func (*failingCleanupStorage) Open(context.Context, string) (io.ReadCloser, error) {
	panic("unexpected Open")
}
func (s *failingCleanupStorage) Delete(ctx context.Context, _ string) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("missing deadline")
	}
	return s.err
}

func TestCleanupIsDurableAndIndependentOfCanceledRequest(t *testing.T) {
	client := &fakeCleanupInserter{}
	wrapped := NewCleanupStorage(&failingCleanupStorage{}, client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := wrapped.Delete(ctx, "file.csv"); err != nil || client.key != "file.csv" {
		t.Fatalf("failed to persist cleanup: %v", err)
	}
	client.err = errors.New("database unavailable")
	if err := wrapped.Delete(ctx, "file.csv"); !errors.Is(err, client.err) {
		t.Fatalf("must not acknowledge lost cleanup: %v", err)
	}
}

func TestCleanupRetriesWithoutExhaustingAttempts(t *testing.T) {
	s := &failingCleanupStorage{err: errors.New("S3 unavailable")}
	worker := &CleanupWorker{storage: s}
	job := &river.Job[CleanupArgs]{JobRow: &rivertype.JobRow{Attempt: 25, MaxAttempts: 25}, Args: CleanupArgs{StorageKey: "file.csv"}}
	var snooze *river.JobSnoozeError
	if err := worker.Work(context.Background(), job); !errors.As(err, &snooze) {
		t.Fatalf("must remain pending after storage failure: %v", err)
	}
	s.err = nil
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatal(err)
	}
}

func TestIngestionCleanupFailureDoesNotFailUploadOnLastAttempt(t *testing.T) {
	processor := &processorSpy{err: &usecase.CleanupPendingError{Cause: errors.New("queue unavailable")}}
	worker := NewIngestionWorker(processor, nil, nil)
	job := &river.Job[IngestionArgs]{JobRow: &rivertype.JobRow{Attempt: 5, MaxAttempts: 5}, Args: IngestionArgs{Type: "alarms"}}
	var snooze *river.JobSnoozeError
	if err := worker.Work(context.Background(), job); !errors.As(err, &snooze) || len(processor.failed) != 0 {
		t.Fatalf("cleanup must not fail ingestion: %v", err)
	}
	processor.err = errors.New("ingestion failure")
	processor.failErr = &usecase.CleanupPendingError{Cause: errors.New("queue unavailable")}
	if err := worker.Work(context.Background(), job); !errors.As(err, &snooze) {
		t.Fatalf("failed upload cleanup must be retried: %v", err)
	}
}

func TestCleanupJobSurvivesProducerAndRunsInSeparateWorker(t *testing.T) {
	db := openRiverTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := MigrateSQL(ctx, db, nil); err != nil {
		t.Fatal(err)
	}
	producer, err := NewInsertClient(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := storage.NewLocalTempStorage()
	key, _, err := raw.Store(ctx, "cleanup.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Delete(context.Background(), key)
	wrapped := NewCleanupStorage(raw, producer)
	if err := wrapped.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if err := wrapped.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM river_job WHERE kind = 'delete_ingestion_file' AND args->>'storage_key' = $1", key).Scan(&count); err != nil || count != 1 {
		t.Fatalf("cleanup must be persisted and deduplicated: %d %v", count, err)
	}
	if _, err := os.Stat(key); err != nil {
		t.Fatalf("producer must leave deletion to worker: %v", err)
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5439/postgres?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	consumer, err := NewWorkerClient(pool, NewIngestionWorker(nil, nil, nil), WorkerConfig{CleanupStorage: raw, PollOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := consumer.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := consumer.Stop(stopCtx); err != nil {
			t.Error(err)
		}
	}()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatal("separate worker did not delete persisted object")
			return
		case <-ticker.C:
			if _, err := os.Stat(key); errors.Is(err, os.ErrNotExist) {
				return
			}
		}
	}
}
