package jobqueue

import (
	"context"
	"errors"
	"testing"

	"github.com/g04/backend/internal/usecase/port"
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
}

type processorSpy struct {
	processed []port.IngestionJob
	failed    []port.IngestionJob
	err       error
}

func (p *processorSpy) ProcessJob(_ context.Context, job port.IngestionJob) error {
	p.processed = append(p.processed, job)
	return p.err
}

func (p *processorSpy) FailJob(_ context.Context, job port.IngestionJob, _ string) error {
	p.failed = append(p.failed, job)
	return nil
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
