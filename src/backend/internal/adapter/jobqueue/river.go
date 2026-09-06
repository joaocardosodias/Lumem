package jobqueue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	QueueName = "ingestion"
	jobKind   = "process_ingestion_upload"
)

type IngestionArgs struct {
	Type         port.IngestionJobType `json:"type"`
	UploadID     string                `json:"upload_id" river:"unique"`
	StorageKey   string                `json:"storage_key"`
	FileName     string                `json:"file_name"`
	OrderType    string                `json:"order_type,omitempty"`
	TraceContext map[string]string     `json:"trace_context,omitempty"`
}

func (IngestionArgs) Kind() string { return jobKind }

func (IngestionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:      QueueName,
		UniqueOpts: river.UniqueOpts{ByArgs: true},
	}
}

type RiverQueue struct {
	db          *sql.DB
	client      riverTxInserter
	maxAttempts int
}

type riverTxInserter interface {
	InsertTx(ctx context.Context, tx *sql.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

var _ port.IngestionJobQueue = (*RiverQueue)(nil)

func NewRiverQueue(db *sql.DB, client *river.Client[*sql.Tx], maxAttempts int) *RiverQueue {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &RiverQueue{db: db, client: client, maxAttempts: maxAttempts}
}

func NewInsertClient(db *sql.DB, logger *slog.Logger) (*river.Client[*sql.Tx], error) {
	return river.NewClient(riverdatabasesql.New(db), &river.Config{Logger: logger})
}

func (q *RiverQueue) Enqueue(ctx context.Context, upload *domain.FileUpload, job port.IngestionJob) error {
	if q == nil || q.db == nil || q.client == nil {
		return errors.New("cliente River nao configurado")
	}

	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transacao de agendamento: %w", err)
	}
	defer tx.Rollback()

	if err := repository.NewPostgresAlarmRepository(tx).CreateUpload(ctx, upload); err != nil {
		return err
	}

	args := newIngestionArgs(ctx, job)
	insertOpts := args.InsertOpts()
	insertOpts.MaxAttempts = q.maxAttempts
	if _, err := q.client.InsertTx(ctx, tx, args, &insertOpts); err != nil {
		return fmt.Errorf("inserir job River: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar upload e job: %w", err)
	}
	return nil
}

func newIngestionArgs(ctx context.Context, job port.IngestionJob) IngestionArgs {
	traceContext := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(traceContext))
	return IngestionArgs{
		Type:         job.Type,
		UploadID:     job.UploadID,
		StorageKey:   job.StorageKey,
		FileName:     job.FileName,
		OrderType:    job.OrderType,
		TraceContext: traceContext,
	}
}

type IngestionProcessor interface {
	ProcessJob(ctx context.Context, job port.IngestionJob) error
	FailJob(ctx context.Context, job port.IngestionJob, reason string) error
}

type IngestionWorker struct {
	river.WorkerDefaults[IngestionArgs]
	alarms      IngestionProcessor
	maintenance IngestionProcessor
	logger      *slog.Logger
}

func NewIngestionWorker(alarms, maintenance IngestionProcessor, logger *slog.Logger) *IngestionWorker {
	return &IngestionWorker{alarms: alarms, maintenance: maintenance, logger: logger}
}

func (w *IngestionWorker) Work(ctx context.Context, riverJob *river.Job[IngestionArgs]) error {
	args := riverJob.Args
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(args.TraceContext))
	ctx, span := otel.Tracer("g04/backend/jobqueue").Start(ctx, "ingestion.process",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "river"),
			attribute.String("messaging.destination.name", QueueName),
			attribute.String("ingestion.type", string(args.Type)),
			attribute.String("ingestion.upload_id", args.UploadID),
			attribute.Int("messaging.message.retry.count", riverJob.Attempt-1),
		),
	)
	defer span.End()

	job := port.IngestionJob{
		Type:         args.Type,
		UploadID:     args.UploadID,
		StorageKey:   args.StorageKey,
		FileName:     args.FileName,
		OrderType:    args.OrderType,
		TraceContext: args.TraceContext,
	}

	processor, err := w.processorFor(args.Type)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return river.JobCancel(err)
	}

	err = processor.ProcessJob(ctx, job)
	if err == nil {
		return nil
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	if w.logger != nil {
		fields := []any{
			slog.String("upload_id", args.UploadID),
			slog.String("type", string(args.Type)),
			slog.Int("attempt", riverJob.Attempt),
			slog.Int("max_attempts", riverJob.MaxAttempts),
			slog.String("error", err.Error()),
		}
		spanContext := trace.SpanContextFromContext(ctx)
		if spanContext.IsValid() {
			fields = append(fields,
				slog.String("trace_id", spanContext.TraceID().String()),
				slog.String("span_id", spanContext.SpanID().String()),
			)
		}
		w.logger.Error("falha ao processar job de ingestao", fields...)
	}

	if riverJob.Attempt >= riverJob.MaxAttempts {
		failCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		reason := fmt.Sprintf("processamento falhou apos %d tentativas: %v", riverJob.Attempt, err)
		if failErr := processor.FailJob(failCtx, job, reason); failErr != nil {
			return errors.Join(err, fmt.Errorf("marcar upload como falho: %w", failErr))
		}
	}
	return err
}

func (w *IngestionWorker) processorFor(jobType port.IngestionJobType) (IngestionProcessor, error) {
	switch jobType {
	case port.IngestionJobAlarms:
		if w.alarms != nil {
			return w.alarms, nil
		}
	case port.IngestionJobMaintenance:
		if w.maintenance != nil {
			return w.maintenance, nil
		}
	}
	return nil, fmt.Errorf("tipo de ingestao desconhecido: %q", jobType)
}

type WorkerConfig struct {
	MaxWorkers int
	PollOnly   bool
	Logger     *slog.Logger
}

func NewWorkerClient(pool *pgxpool.Pool, worker *IngestionWorker, cfg WorkerConfig) (*river.Client[pgx.Tx], error) {
	maxWorkers := cfg.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 2
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		JobTimeout:      15 * time.Minute,
		Logger:          cfg.Logger,
		PollOnly:        cfg.PollOnly,
		Queues:          map[string]river.QueueConfig{QueueName: {MaxWorkers: maxWorkers}},
		SoftStopTimeout: 20 * time.Second,
		Workers:         workers,
	})
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), &rivermigrate.Config{Logger: logger})
	if err != nil {
		return fmt.Errorf("criar migrador River: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("aplicar migracoes River: %w", err)
	}
	return nil
}

func MigrateSQL(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	migrator, err := rivermigrate.New(riverdatabasesql.New(db), &rivermigrate.Config{Logger: logger})
	if err != nil {
		return fmt.Errorf("criar migrador River: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("aplicar migracoes River: %w", err)
	}
	return nil
}
