package usecase

import (
	"context"
	"fmt"

	"github.com/g04/backend/internal/usecase/port"
)

// consumeBatches propagates database/cancellation errors unchanged so River can
// retry them. Only parsing/validation errors are permanent ingestion failures.
func consumeBatches[T any](ctx context.Context, stream func(func(T) error) error, insert func([]T) (int, error)) (int, int, error) {
	const batchSize = 500
	batch := make([]T, 0, batchSize)
	processed, inserted := 0, 0
	var callbackErr error
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		n, err := insert(batch)
		if err != nil {
			return err
		}
		inserted += n
		clear(batch)
		batch = batch[:0]
		return nil
	}
	err := stream(func(record T) error {
		if callbackErr != nil {
			return callbackErr
		}
		if callbackErr = ctx.Err(); callbackErr != nil {
			return callbackErr
		}
		processed++
		batch = append(batch, record)
		if len(batch) == batchSize {
			callbackErr = flush()
		}
		return callbackErr
	})
	if callbackErr != nil {
		return processed, inserted, callbackErr
	}
	if ctx.Err() != nil {
		return processed, inserted, ctx.Err()
	}
	if err != nil {
		return processed, inserted, permanentIngestionFailure("falha ao analisar arquivo: %w", err)
	}
	if processed == 0 {
		return 0, 0, permanentIngestionFailure("arquivo vazio ou sem linhas de dados validas")
	}
	err = flush()
	return processed, inserted, err
}

// CleanupPendingError asks the queue to retry finalization without consuming
// ingestion attempts or changing an already persisted terminal status.
type CleanupPendingError struct{ Cause error }

func (e *CleanupPendingError) Error() string {
	return fmt.Sprintf("limpeza de upload pendente: %v", e.Cause)
}
func (e *CleanupPendingError) Unwrap() error { return e.Cause }

func terminalUpload(ctx context.Context, repo port.UploadRepository, id string) (bool, error) {
	upload, err := repo.GetUploadByID(ctx, id)
	if err != nil {
		return false, err
	}
	if upload == nil {
		return false, fmt.Errorf("upload %s nao encontrado", id)
	}
	return upload.Status == "completed" || upload.Status == "failed", nil
}
