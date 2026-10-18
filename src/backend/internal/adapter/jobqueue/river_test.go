package jobqueue

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestIngestionArgsSerializeJobAndTrace(t *testing.T) {
	oldPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(oldPropagator) })

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	args := newIngestionArgs(ctx, port.IngestionJob{
		Type:       port.IngestionJobAlarms,
		UploadID:   "upload-1",
		StorageKey: "ingestion/file.csv",
		FileName:   "file.csv",
	})
	if args.UploadID != "upload-1" || args.TraceContext["traceparent"] == "" {
		t.Fatalf("job nao preservou dados e trace: %+v", args)
	}
	opts := args.InsertOpts()
	if opts.Queue != QueueName || !opts.UniqueOpts.ByArgs {
		t.Fatalf("opcoes de insercao inesperadas: %+v", opts)
	}
	if args.Kind() != jobKind {
		t.Fatalf("kind inesperado: %s", args.Kind())
	}
}

type processorSpy struct {
	processed []port.IngestionJob
	failed    []port.IngestionJob
	err       error
	failErr   error
}

func (p *processorSpy) ProcessJob(_ context.Context, job port.IngestionJob) error {
	p.processed = append(p.processed, job)
	return p.err
}

func (p *processorSpy) FailJob(_ context.Context, job port.IngestionJob, _ string) error {
	p.failed = append(p.failed, job)
	return p.failErr
}

func TestIngestionWorkerDispatchesAndMarksLastAttempt(t *testing.T) {
	alarms := &processorSpy{err: errors.New("database unavailable")}
	worker := NewIngestionWorker(alarms, &processorSpy{}, nil)
	job := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 5, MaxAttempts: 5},
		Args: IngestionArgs{
			Type:       port.IngestionJobAlarms,
			UploadID:   "upload-2",
			StorageKey: "ingestion/file.csv",
			FileName:   "file.csv",
		},
	}

	err := worker.Work(context.Background(), job)
	if err == nil || len(alarms.processed) != 1 || len(alarms.failed) != 1 {
		t.Fatalf("ultima tentativa nao foi tratada: err=%v processed=%d failed=%d", err, len(alarms.processed), len(alarms.failed))
	}
}

func TestIngestionWorkerCancelsUnknownType(t *testing.T) {
	worker := NewIngestionWorker(&processorSpy{}, &processorSpy{}, nil)
	job := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
		Args:   IngestionArgs{Type: "unknown", UploadID: "upload-3"},
	}

	err := worker.Work(context.Background(), job)
	var cancelErr *river.JobCancelError
	if !errors.As(err, &cancelErr) {
		t.Fatalf("esperava cancelamento permanente, recebeu %v", err)
	}
}

func TestRiverQueueNewAndValidation(t *testing.T) {
	q1 := NewRiverQueue(nil, nil, 0)
	if q1.maxAttempts != 5 {
		t.Fatalf("esperava maxAttempts padrao 5, obteve %d", q1.maxAttempts)
	}
	q2 := NewRiverQueue(nil, nil, 8)
	if q2.maxAttempts != 8 {
		t.Fatalf("esperava maxAttempts 8, obteve %d", q2.maxAttempts)
	}

	var qNil *RiverQueue
	if err := qNil.Enqueue(context.Background(), &domain.FileUpload{}, port.IngestionJob{}); err == nil || !strings.Contains(err.Error(), "cliente River nao configurado") {
		t.Fatalf("esperava erro de cliente nao configurado para nil receiver, obteve %v", err)
	}
	if err := q1.Enqueue(context.Background(), &domain.FileUpload{}, port.IngestionJob{}); err == nil || !strings.Contains(err.Error(), "cliente River nao configurado") {
		t.Fatalf("esperava erro de cliente nao configurado para q1, obteve %v", err)
	}
}

func TestIngestionWorkerSuccess(t *testing.T) {
	alarmsSpy := &processorSpy{}
	maintSpy := &processorSpy{}
	worker := NewIngestionWorker(alarmsSpy, maintSpy, nil)

	alarmsJob := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
		Args:   IngestionArgs{Type: port.IngestionJobAlarms, UploadID: "up-a1"},
	}
	if err := worker.Work(context.Background(), alarmsJob); err != nil {
		t.Fatalf("esperava sucesso para alarms, obteve %v", err)
	}
	if len(alarmsSpy.processed) != 1 {
		t.Fatalf("esperava 1 job de alarmes processado, obteve %d", len(alarmsSpy.processed))
	}

	maintJob := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
		Args:   IngestionArgs{Type: port.IngestionJobMaintenance, UploadID: "up-m1"},
	}
	if err := worker.Work(context.Background(), maintJob); err != nil {
		t.Fatalf("esperava sucesso para maintenance, obteve %v", err)
	}
	if len(maintSpy.processed) != 1 {
		t.Fatalf("esperava 1 job de manutencao processado, obteve %d", len(maintSpy.processed))
	}
}

func TestIngestionWorkerIntermediateAttemptDoesNotMarkFailed(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	alarmsSpy := &processorSpy{err: errors.New("falha transiente")}
	worker := NewIngestionWorker(alarmsSpy, &processorSpy{}, logger)

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	job := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 2, MaxAttempts: 5},
		Args:   IngestionArgs{Type: port.IngestionJobAlarms, UploadID: "up-retry"},
	}
	err := worker.Work(ctx, job)
	if err == nil || !strings.Contains(err.Error(), "falha transiente") {
		t.Fatalf("esperava erro transiente, obteve %v", err)
	}
	if len(alarmsSpy.failed) != 0 {
		t.Fatalf("nao deveria ter marcado como falho na tentativa 2: %d", len(alarmsSpy.failed))
	}
	logStr := buf.String()
	if !strings.Contains(logStr, "falha ao processar job de ingestao") || !strings.Contains(logStr, "trace_id") {
		t.Fatalf("esperava log de erro com trace_id, obteve: %s", logStr)
	}
}

func TestIngestionWorkerFailJobError(t *testing.T) {
	alarmsSpy := &processorSpy{err: errors.New("erro original"), failErr: errors.New("banco inacessivel")}
	worker := NewIngestionWorker(alarmsSpy, &processorSpy{}, nil)

	job := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 5, MaxAttempts: 5},
		Args:   IngestionArgs{Type: port.IngestionJobAlarms, UploadID: "up-last-fail"},
	}
	err := worker.Work(context.Background(), job)
	if err == nil || !strings.Contains(err.Error(), "marcar upload como falho") {
		t.Fatalf("esperava erro combinado de falha, obteve %v", err)
	}
}

func TestIngestionWorkerNilProcessors(t *testing.T) {
	worker := NewIngestionWorker(nil, nil, nil)
	jobA := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
		Args:   IngestionArgs{Type: port.IngestionJobAlarms, UploadID: "up-nil-a"},
	}
	errA := worker.Work(context.Background(), jobA)
	var cancelErrA *river.JobCancelError
	if !errors.As(errA, &cancelErrA) {
		t.Fatalf("esperava cancelamento para alarms nulo, obteve %v", errA)
	}

	jobM := &river.Job[IngestionArgs]{
		JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 5},
		Args:   IngestionArgs{Type: port.IngestionJobMaintenance, UploadID: "up-nil-m"},
	}
	errM := worker.Work(context.Background(), jobM)
	var cancelErrM *river.JobCancelError
	if !errors.As(errM, &cancelErrM) {
		t.Fatalf("esperava cancelamento para maintenance nulo, obteve %v", errM)
	}
}

func TestNewInsertClient(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("falha ao abrir sql.DB: %v", err)
	}
	defer db.Close()

	client, err := NewInsertClient(db, nil)
	if err != nil {
		t.Fatalf("esperava sucesso ao criar insert client: %v", err)
	}
	if client == nil {
		t.Fatal("esperava client nao nulo")
	}
}

func TestNewWorkerClient(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("falha ao parsear configuracao de pool: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("falha ao criar pool: %v", err)
	}
	defer pool.Close()

	worker := NewIngestionWorker(nil, nil, nil)
	client, err := NewWorkerClient(pool, worker, WorkerConfig{MaxWorkers: 0})
	if err != nil {
		t.Fatalf("esperava sucesso ao criar worker client: %v", err)
	}
	if client == nil {
		t.Fatal("esperava worker client nao nulo")
	}

	clientExplicitWorkers, err := NewWorkerClient(pool, worker, WorkerConfig{MaxWorkers: 4})
	if err != nil || clientExplicitWorkers == nil {
		t.Fatalf("esperava sucesso com workers explicitos: %v", err)
	}
}

func TestMigrateAndMigrateSQL(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("falha ao parsear configuracao de pool: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("falha ao criar pool: %v", err)
	}
	defer pool.Close()

	_ = Migrate(context.Background(), pool, nil)

	db, err := sql.Open("pgx", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("falha ao abrir sql.DB: %v", err)
	}
	defer db.Close()

	_ = MigrateSQL(context.Background(), db, nil)
}

func TestRiverQueueEnqueueBeginTxError(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://invalid:pass@127.0.0.1:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("falha ao abrir sql.DB: %v", err)
	}
	defer db.Close()

	client, err := NewInsertClient(db, nil)
	if err != nil {
		t.Fatalf("falha ao criar insert client: %v", err)
	}

	q := NewRiverQueue(db, client, 5)
	err = q.Enqueue(context.Background(), &domain.FileUpload{ID: "up-tx-fail"}, port.IngestionJob{})
	if err == nil || !strings.Contains(err.Error(), "iniciar transacao de agendamento") {
		t.Fatalf("esperava erro de conexao na transacao, obteve %v", err)
	}
}

type fakeRiverInserter struct {
	err      error
	rollback bool
	args     IngestionArgs
	opts     river.InsertOpts
}

func (f *fakeRiverInserter) InsertTx(_ context.Context, tx *sql.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	f.args = args.(IngestionArgs)
	f.opts = *opts
	if f.rollback {
		_ = tx.Rollback()
	}
	if f.err != nil {
		return nil, f.err
	}
	return &rivertype.JobInsertResult{}, nil
}

func openRiverTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5439/postgres?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("nao foi possivel abrir o PostgreSQL de teste: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		t.Skipf("PostgreSQL de teste indisponivel: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRiverQueueEnqueueTransaction(t *testing.T) {
	db := openRiverTestDB(t)
	db.SetMaxOpenConns(1)
	_, err := db.Exec(`
		CREATE TEMP TABLE file_uploads (
			id UUID PRIMARY KEY,
			file_name TEXT NOT NULL,
			file_type TEXT NOT NULL,
			file_hash TEXT UNIQUE,
			status TEXT,
			rows_processed INTEGER,
			rows_inserted INTEGER,
			error_log TEXT
		)
	`)
	if err != nil {
		t.Fatalf("criar tabela temporaria de uploads: %v", err)
	}

	newUpload := func(id, hash string) *domain.FileUpload {
		return &domain.FileUpload{
			ID:       id,
			FileName: "alarmes.csv",
			FileType: "alarms",
			FileHash: hash,
			Status:   "queued",
		}
	}
	job := port.IngestionJob{
		Type:       port.IngestionJobAlarms,
		UploadID:   "10000000-0000-0000-0000-000000000001",
		StorageKey: "ingestion/alarmes.csv",
		FileName:   "alarmes.csv",
	}

	successInserter := &fakeRiverInserter{}
	queue := &RiverQueue{db: db, client: successInserter, maxAttempts: 7}
	if err := queue.Enqueue(context.Background(), newUpload(job.UploadID, "hash-success"), job); err != nil {
		t.Fatalf("enfileirar upload: %v", err)
	}
	if successInserter.args.UploadID != job.UploadID || successInserter.opts.MaxAttempts != 7 {
		t.Fatalf("argumentos do job inesperados: args=%+v opts=%+v", successInserter.args, successInserter.opts)
	}
	var persisted int
	if err := db.QueryRow("SELECT COUNT(*) FROM file_uploads WHERE id = $1", job.UploadID).Scan(&persisted); err != nil || persisted != 1 {
		t.Fatalf("upload nao foi confirmado na transacao: count=%d err=%v", persisted, err)
	}

	duplicate := newUpload("10000000-0000-0000-0000-000000000002", "hash-success")
	if err := queue.Enqueue(context.Background(), duplicate, job); !errors.Is(err, domain.ErrDuplicateUpload) {
		t.Fatalf("esperava erro de upload duplicado, obteve %v", err)
	}

	insertFailure := &fakeRiverInserter{err: errors.New("river indisponivel")}
	queue.client = insertFailure
	failedUpload := newUpload("10000000-0000-0000-0000-000000000003", "hash-insert-error")
	if err := queue.Enqueue(context.Background(), failedUpload, job); err == nil || !strings.Contains(err.Error(), "inserir job River") {
		t.Fatalf("esperava erro de insercao River, obteve %v", err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM file_uploads WHERE id = $1", failedUpload.ID).Scan(&persisted); err != nil || persisted != 0 {
		t.Fatalf("upload deveria sofrer rollback: count=%d err=%v", persisted, err)
	}

	commitFailure := &fakeRiverInserter{rollback: true}
	queue.client = commitFailure
	commitUpload := newUpload("10000000-0000-0000-0000-000000000004", "hash-commit-error")
	if err := queue.Enqueue(context.Background(), commitUpload, job); err == nil || !strings.Contains(err.Error(), "confirmar upload e job") {
		t.Fatalf("esperava erro de commit, obteve %v", err)
	}
}

func TestRiverMigrationsSuccess(t *testing.T) {
	db := openRiverTestDB(t)
	ctx := context.Background()
	if err := MigrateSQL(ctx, db, nil); err != nil {
		t.Fatalf("migracao River via database/sql: %v", err)
	}

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5439/postgres?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir pool pgx: %v", err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool, nil); err != nil {
		t.Fatalf("migracao River via pgx: %v", err)
	}
}
