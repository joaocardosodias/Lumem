package jobqueue

import (
	"context"
	"log/slog"
	"time"

	"github.com/g04/backend/internal/usecase/port"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const cleanupQueueName = "ingestion_cleanup"

type CleanupArgs struct {
	StorageKey string `json:"storage_key" river:"unique"`
}

func (CleanupArgs) Kind() string { return "delete_ingestion_file" }
func (CleanupArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: cleanupQueueName, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type cleanupInserter interface {
	Insert(context.Context, river.JobArgs, *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// CleanupStorage persists deletion intent before acknowledging it. The actual
// deletion belongs to a dedicated worker and survives API/worker restarts.
type CleanupStorage struct {
	port.IngestionStorage
	client cleanupInserter
}

func NewCleanupStorage(storage port.IngestionStorage, client cleanupInserter) *CleanupStorage {
	return &CleanupStorage{IngestionStorage: storage, client: client}
}
func (s *CleanupStorage) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, err := s.client.Insert(ctx, CleanupArgs{StorageKey: key}, nil)
	return err
}

type CleanupWorker struct {
	river.WorkerDefaults[CleanupArgs]
	storage port.IngestionStorage
	logger  *slog.Logger
}

func (w *CleanupWorker) Work(ctx context.Context, job *river.Job[CleanupArgs]) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := w.storage.Delete(ctx, job.Args.StorageKey); err != nil {
		if w.logger != nil {
			w.logger.Error("limpeza de arquivo pendente; nova tentativa em 5 minutos", "storage_key", job.Args.StorageKey, "error", err)
		}
		// Snoozing keeps the durable job pending without exhausting ingestion retries.
		return river.JobSnooze(5 * time.Minute)
	}
	return nil
}
