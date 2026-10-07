package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type IngestMaintenanceUseCase struct {
	maintRepo   port.MaintenanceRepository
	uploadRepo  port.UploadRepository
	storage     port.IngestionStorage
	maintParser port.MaintenanceParser
	runner      port.JobRunner
	queue       port.IngestionJobQueue
	uow         port.UnitOfWork
	logger      port.Logger
}

func NewIngestMaintenanceUseCaseWithQueue(
	maintRepo port.MaintenanceRepository,
	uploadRepo port.UploadRepository,
	storage port.IngestionStorage,
	maintParser port.MaintenanceParser,
	queue port.IngestionJobQueue,
	uow port.UnitOfWork,
	logger port.Logger,
) *IngestMaintenanceUseCase {
	return &IngestMaintenanceUseCase{
		maintRepo:   maintRepo,
		uploadRepo:  uploadRepo,
		storage:     storage,
		maintParser: maintParser,
		queue:       queue,
		uow:         uow,
		logger:      logger,
	}
}

func NewIngestMaintenanceUseCase(
	maintRepo port.MaintenanceRepository,
	uploadRepo port.UploadRepository,
	storage port.IngestionStorage,
	maintParser port.MaintenanceParser,
	runner port.JobRunner,
	uow port.UnitOfWork,
	logger port.Logger,
) *IngestMaintenanceUseCase {
	return &IngestMaintenanceUseCase{
		maintRepo:   maintRepo,
		uploadRepo:  uploadRepo,
		storage:     storage,
		maintParser: maintParser,
		runner:      runner,
		uow:         uow,
		logger:      logger,
	}
}

func resolveMaintenanceFileType(explicitOrderType string) string {
	if explicitOrderType != "" {
		return "maintenance_" + strings.ToLower(explicitOrderType)
	}
	return "maintenance"
}

func (uc *IngestMaintenanceUseCase) checkDuplicateHash(ctx context.Context, fileHash, storageKey string) (string, bool) {
	existing, err := uc.uploadRepo.GetUploadByHash(ctx, fileHash)
	if err != nil || existing == nil {
		return "", false
	}
	if delErr := uc.cleanupStoredFile(ctx, storageKey); delErr != nil && uc.logger != nil {
		uc.logger.Warn("falha ao remover arquivo temporario de upload duplicado", "storage_key", storageKey, "error", delErr.Error())
	}
	return existing.ID, true
}

func (uc *IngestMaintenanceUseCase) handleCreateUploadError(ctx context.Context, storageKey, fileHash string, err error) (string, error) {
	if delErr := uc.cleanupStoredFile(ctx, storageKey); delErr != nil && uc.logger != nil {
		uc.logger.Warn("falha ao remover arquivo temporario apos erro ao criar upload", "storage_key", storageKey, "error", delErr.Error())
	}
	if errors.Is(err, domain.ErrDuplicateUpload) {
		existingUpload, getErr := uc.uploadRepo.GetUploadByHash(ctx, fileHash)
		if getErr == nil && existingUpload != nil {
			return existingUpload.ID, domain.ErrDuplicateUpload
		}
		return "", domain.ErrDuplicateUpload
	}
	return "", err
}

func (uc *IngestMaintenanceUseCase) StartIngestion(ctx context.Context, fileName string, reader io.Reader, explicitOrderType string) (string, error) {
	storageKey, fileHash, err := uc.storage.Store(ctx, fileName, reader)
	if err != nil {
		return "", err
	}

	if existingID, isDup := uc.checkDuplicateHash(ctx, fileHash, storageKey); isDup {
		return existingID, domain.ErrDuplicateUpload
	}

	upload := &domain.FileUpload{
		ID:       uuid.New().String(),
		FileName: fileName,
		FileType: resolveMaintenanceFileType(explicitOrderType),
		FileHash: fileHash,
		Status:   "processing",
	}

	job := port.IngestionJob{
		Type:       port.IngestionJobMaintenance,
		UploadID:   upload.ID,
		StorageKey: storageKey,
		FileName:   fileName,
		OrderType:  explicitOrderType,
	}
	if uc.queue != nil {
		if err := uc.queue.Enqueue(ctx, upload, job); err != nil {
			return uc.handleCreateUploadError(ctx, storageKey, fileHash, err)
		}
	} else {
		if err := uc.uploadRepo.CreateUpload(ctx, upload); err != nil {
			return uc.handleCreateUploadError(ctx, storageKey, fileHash, err)
		}
		uc.runner.Run(func() {
			uc.processFileAsync(upload.ID, storageKey, fileName, explicitOrderType)
		})
	}

	return upload.ID, nil
}

func (uc *IngestMaintenanceUseCase) recordFailure(uploadID, reason, logMsg string, cause error) {
	fullReason := reason
	if cause != nil {
		fullReason = reason + ": " + cause.Error()
	}
	if failErr := uc.failUpload(uploadID, fullReason); failErr != nil && uc.logger != nil {
		uc.logger.Error(logMsg, "upload_id", uploadID, "error", failErr.Error())
	}
}

func convertRawMaintenanceEvents(rawRecords []port.RawMaintenanceRecord, uploadID, fileName string) ([]*domain.MaintenanceEvent, int) {
	source := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if source == "" {
		source = "maintenance"
	}

	allEvents := make([]*domain.MaintenanceEvent, 0, len(rawRecords))
	for _, raw := range rawRecords {
		event, err := domain.NewMaintenanceEvent(domain.CreateMaintenanceParams{
			ID:                   uuid.New().String(),
			SAPOrderNumber:       raw.SAPOrderNumber,
			OrderType:            raw.OrderType,
			StationCode:          raw.StationCode,
			InstallationLocation: raw.InstallationLocation,
			StartTimestamp:       raw.StartTimestamp,
			EndTimestamp:         raw.EndTimestamp,
			DurationHours:        raw.DurationHours,
			Description:          raw.Description,
			Symptom:              raw.Symptom,
			Cause:                raw.Cause,
			ActionTaken:          raw.ActionTaken,
			Source:               source,
		})
		if err != nil {
			continue
		}
		event.UploadID = uploadID
		allEvents = append(allEvents, event)
	}
	return allEvents, len(rawRecords)
}

func insertMaintenanceBatches(ctx context.Context, store port.UnitOfWorkStore, events []*domain.MaintenanceEvent, batchSize int) (int, error) {
	inserted := 0
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		chunk := events[i:end]
		if err := store.Maintenances().BatchInsertMaintenance(ctx, chunk); err != nil {
			return inserted, fmt.Errorf("falha ao inserir lote de manutencao: %w", err)
		}
		inserted += len(chunk)
	}
	return inserted, nil
}

func finalizeMaintenanceUpload(ctx context.Context, store port.UnitOfWorkStore, uploadID string, processed, inserted int) error {
	now := time.Now()
	upload, err := store.Uploads().GetUploadByID(ctx, uploadID)
	if err != nil {
		return fmt.Errorf("falha ao obter upload: %w", err)
	}
	updated := *upload
	upload = &updated
	upload.RowsProcessed = processed
	upload.RowsInserted = inserted
	upload.Status = "completed"
	upload.CompletedAt = &now
	if err := store.Uploads().UpdateUpload(ctx, upload); err != nil {
		return fmt.Errorf("falha ao atualizar status do upload: %w", err)
	}
	return nil
}

func (uc *IngestMaintenanceUseCase) deleteStoredFile(ctx context.Context, storageKey string) error {
	if err := uc.cleanupStoredFile(ctx, storageKey); err != nil {
		if uc.logger != nil {
			uc.logger.Warn("falha ao remover arquivo de ingestao", "storage_key", storageKey, "error", err.Error())
		}
		return err
	}
	return nil
}

func (uc *IngestMaintenanceUseCase) ProcessJob(ctx context.Context, job port.IngestionJob) error {
	terminal, err := terminalUpload(ctx, uc.uploadRepo, job.UploadID)
	if err != nil {
		return err
	}
	if terminal {
		return uc.cleanupJob(ctx, job.StorageKey)
	}
	rc, err := uc.storage.Open(ctx, job.StorageKey)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo: %w", err)
	}
	// Close the input before final cleanup; Windows cannot delete an open file.
	err = func() error {
		defer rc.Close()
		return uc.uow.Do(ctx, func(store port.UnitOfWorkStore) error {
			if err := store.Maintenances().DeleteByUploadID(ctx, job.UploadID); err != nil {
				return fmt.Errorf("falha ao preparar nova tentativa de manutencao: %w", err)
			}
			processed, inserted, err := consumeBatches(ctx,
				func(yield func(port.RawMaintenanceRecord) error) error {
					return uc.maintParser.StreamMaintenance(ctx, rc, job.FileName, job.OrderType, yield)
				},
				func(batch []port.RawMaintenanceRecord) (int, error) {
					records, _ := convertRawMaintenanceEvents(batch, job.UploadID, job.FileName)
					return insertMaintenanceBatches(ctx, store, records, 500)
				})
			if err != nil {
				return err
			}
			return finalizeMaintenanceUpload(ctx, store, job.UploadID, processed, inserted)
		})
	}()
	if err != nil {
		var permanentErr *permanentIngestionError
		if !errors.As(err, &permanentErr) {
			return err
		}
		if failErr := uc.failUploadContext(ctx, job.UploadID, err.Error()); failErr != nil {
			return fmt.Errorf("registrar falha permanente do upload: %w", failErr)
		}
	}
	return uc.cleanupJob(ctx, job.StorageKey)
}

func (uc *IngestMaintenanceUseCase) cleanupJob(ctx context.Context, storageKey string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := uc.deleteStoredFile(cleanupCtx, storageKey); err != nil {
		return &CleanupPendingError{Cause: err}
	}
	return nil
}

func (uc *IngestMaintenanceUseCase) FailJob(ctx context.Context, job port.IngestionJob, reason string) error {
	if err := uc.failUploadContext(ctx, job.UploadID, reason); err != nil {
		return err
	}
	return uc.cleanupJob(ctx, job.StorageKey)
}

func (uc *IngestMaintenanceUseCase) processFileAsync(uploadID, storageKey, fileName, explicitOrderType string) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	job := port.IngestionJob{Type: port.IngestionJobMaintenance, UploadID: uploadID, StorageKey: storageKey, FileName: fileName, OrderType: explicitOrderType}
	if err := uc.ProcessJob(bgCtx, job); err != nil {
		_ = uc.FailJob(context.Background(), job, err.Error())
	}
}

func (uc *IngestMaintenanceUseCase) failUpload(uploadID string, reason string) error {
	return uc.failUploadContext(context.Background(), uploadID, reason)
}

func (uc *IngestMaintenanceUseCase) failUploadContext(ctx context.Context, uploadID string, reason string) error {
	now := time.Now()
	upload, err := uc.uploadRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		if uc.logger != nil {
			uc.logger.Error("falha ao obter upload para marcar como failed", "upload_id", uploadID, "reason", reason, "error", err.Error())
		}
		return err
	}
	if upload.Status == "completed" || upload.Status == "failed" {
		return nil
	}
	updated := *upload
	upload = &updated
	upload.Status = "failed"
	upload.ErrorLog = reason
	upload.CompletedAt = &now
	if err := uc.uploadRepo.UpdateUpload(ctx, upload); err != nil {
		if uc.logger != nil {
			uc.logger.Error("falha ao persistir status failed do upload", "upload_id", uploadID, "reason", reason, "error", err.Error())
		}
		return err
	}
	return nil
}

func (uc *IngestMaintenanceUseCase) cleanupStoredFile(ctx context.Context, storageKey string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return uc.storage.Delete(cleanupCtx, storageKey)
}
