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
	uow         port.UnitOfWork
	logger      port.Logger
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
	if delErr := uc.storage.Delete(ctx, storageKey); delErr != nil && uc.logger != nil {
		uc.logger.Warn("falha ao remover arquivo temporario de upload duplicado", "storage_key", storageKey, "error", delErr.Error())
	}
	return existing.ID, true
}

func (uc *IngestMaintenanceUseCase) handleCreateUploadError(ctx context.Context, storageKey, fileHash string, err error) (string, error) {
	if delErr := uc.storage.Delete(ctx, storageKey); delErr != nil && uc.logger != nil {
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

	if err := uc.uploadRepo.CreateUpload(ctx, upload); err != nil {
		return uc.handleCreateUploadError(ctx, storageKey, fileHash, err)
	}

	uc.runner.Run(func() {
		uc.processFileAsync(upload.ID, storageKey, fileName, explicitOrderType)
	})

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

func (uc *IngestMaintenanceUseCase) readAndParseMaintenance(ctx context.Context, storageKey, fileName, uploadID, explicitOrderType string) ([]port.RawMaintenanceRecord, bool) {
	rc, err := uc.storage.Open(ctx, storageKey)
	if err != nil {
		uc.recordFailure(uploadID, "falha ao abrir arquivo", "erro ao registrar falha de abertura do upload", err)
		return nil, false
	}
	defer rc.Close()

	rawRecords, err := uc.maintParser.ParseMaintenance(rc, fileName, explicitOrderType)
	if err != nil {
		uc.recordFailure(uploadID, "falha ao analisar arquivo", "erro ao registrar falha de parsing do upload", err)
		return nil, false
	}

	if len(rawRecords) == 0 {
		uc.recordFailure(uploadID, "arquivo vazio ou sem linhas de dados validas", "erro ao registrar falha de arquivo vazio no upload", nil)
		return nil, false
	}

	return rawRecords, true
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
	upload.RowsProcessed = processed
	upload.RowsInserted = inserted
	upload.Status = "completed"
	upload.CompletedAt = &now
	if err := store.Uploads().UpdateUpload(ctx, upload); err != nil {
		return fmt.Errorf("falha ao atualizar status do upload: %w", err)
	}
	return nil
}

func (uc *IngestMaintenanceUseCase) processFileAsync(uploadID, storageKey, fileName, explicitOrderType string) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	defer func() {
		if delErr := uc.storage.Delete(bgCtx, storageKey); delErr != nil && uc.logger != nil {
			uc.logger.Warn("falha ao remover arquivo temporario", "storage_key", storageKey, "error", delErr.Error())
		}
	}()

	rawRecords, ok := uc.readAndParseMaintenance(bgCtx, storageKey, fileName, uploadID, explicitOrderType)
	if !ok {
		return
	}

	allEvents, processedCount := convertRawMaintenanceEvents(rawRecords, uploadID, fileName)
	const batchSize = 500

	err := uc.uow.Do(bgCtx, func(store port.UnitOfWorkStore) error {
		insertedCount, err := insertMaintenanceBatches(bgCtx, store, allEvents, batchSize)
		if err != nil {
			return err
		}
		return finalizeMaintenanceUpload(bgCtx, store, uploadID, processedCount, insertedCount)
	})

	if err != nil {
		uc.recordFailure(uploadID, err.Error(), "erro ao registrar falha de insercao em lote no upload", nil)
	}
}

func (uc *IngestMaintenanceUseCase) failUpload(uploadID string, reason string) error {
	bgCtx := context.Background()
	now := time.Now()
	upload, err := uc.uploadRepo.GetUploadByID(bgCtx, uploadID)
	if err != nil {
		if uc.logger != nil {
			uc.logger.Error("falha ao obter upload para marcar como failed", "upload_id", uploadID, "reason", reason, "error", err.Error())
		}
		return err
	}
	upload.Status = "failed"
	upload.ErrorLog = reason
	upload.CompletedAt = &now
	if err := uc.uploadRepo.UpdateUpload(bgCtx, upload); err != nil {
		if uc.logger != nil {
			uc.logger.Error("falha ao persistir status failed do upload", "upload_id", uploadID, "reason", reason, "error", err.Error())
		}
		return err
	}
	return nil
}
