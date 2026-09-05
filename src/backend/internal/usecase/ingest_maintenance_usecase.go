package usecase

import (
	"context"
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
}

func NewIngestMaintenanceUseCase(
	maintRepo port.MaintenanceRepository,
	uploadRepo port.UploadRepository,
	storage port.IngestionStorage,
	maintParser port.MaintenanceParser,
	runner port.JobRunner,
	uow port.UnitOfWork,
) *IngestMaintenanceUseCase {
	return &IngestMaintenanceUseCase{
		maintRepo:   maintRepo,
		uploadRepo:  uploadRepo,
		storage:     storage,
		maintParser: maintParser,
		runner:      runner,
		uow:         uow,
	}
}

func (uc *IngestMaintenanceUseCase) StartIngestion(ctx context.Context, fileName string, reader io.Reader, explicitOrderType string) (string, error) {
	storageKey, fileHash, err := uc.storage.Store(ctx, fileName, reader)
	if err != nil {
		return "", err
	}

	existing, err := uc.uploadRepo.GetUploadByHash(ctx, fileHash)
	if err == nil && existing != nil {
		_ = uc.storage.Delete(ctx, storageKey)
		return existing.ID, domain.ErrDuplicateUpload
	}

	fileType := "maintenance"
	if explicitOrderType != "" {
		fileType = "maintenance_" + strings.ToLower(explicitOrderType)
	}

	upload := &domain.FileUpload{
		ID:       uuid.New().String(),
		FileName: fileName,
		FileType: fileType,
		FileHash: fileHash,
		Status:   "processing",
	}

	if err := uc.uploadRepo.CreateUpload(ctx, upload); err != nil {
		_ = uc.storage.Delete(ctx, storageKey)
		return "", err
	}

	uc.runner.Run(func() {
		uc.processFileAsync(upload.ID, storageKey, fileName, explicitOrderType)
	})

	return upload.ID, nil
}

func (uc *IngestMaintenanceUseCase) processFileAsync(uploadID, storageKey, fileName, explicitOrderType string) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	defer func() {
		_ = uc.storage.Delete(bgCtx, storageKey)
	}()

	rc, err := uc.storage.Open(bgCtx, storageKey)
	if err != nil {
		_ = uc.failUpload(uploadID, "falha ao abrir arquivo: "+err.Error())
		return
	}
	defer rc.Close()

	rawRecords, err := uc.maintParser.ParseMaintenance(rc, fileName, explicitOrderType)
	if err != nil {
		_ = uc.failUpload(uploadID, "falha ao analisar arquivo: "+err.Error())
		return
	}

	if len(rawRecords) == 0 {
		_ = uc.failUpload(uploadID, "arquivo vazio ou sem linhas de dados validas")
		return
	}

	const batchSize = 500
	processedCount := 0
	source := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if source == "" {
		source = "maintenance"
	}

	var allEvents []*domain.MaintenanceEvent
	for _, raw := range rawRecords {
		processedCount++

		event, err := domain.NewMaintenanceEvent(
			uuid.New().String(),
			raw.SAPOrderNumber,
			raw.OrderType,
			raw.StationCode,
			raw.InstallationLocation,
			raw.StartTimestamp,
			raw.EndTimestamp,
			raw.DurationHours,
			raw.Description,
			raw.Symptom,
			raw.Cause,
			raw.ActionTaken,
			source,
		)
		if err != nil {
			continue
		}
		event.UploadID = uploadID

		allEvents = append(allEvents, event)
	}

	insertedCount := 0
	err = uc.uow.Do(bgCtx, func(store port.UnitOfWorkStore) error {
		for i := 0; i < len(allEvents); i += batchSize {
			end := i + batchSize
			if end > len(allEvents) {
				end = len(allEvents)
			}
			chunk := allEvents[i:end]
			if err := store.Maintenances().BatchInsertMaintenance(bgCtx, chunk); err != nil {
				return fmt.Errorf("falha ao inserir lote de manutencao: %w", err)
			}
			insertedCount += len(chunk)
		}

		now := time.Now()
		upload, err := store.Uploads().GetUploadByID(bgCtx, uploadID)
		if err != nil {
			return fmt.Errorf("falha ao obter upload: %w", err)
		}
		upload.RowsProcessed = processedCount
		upload.RowsInserted = insertedCount
		upload.Status = "completed"
		upload.CompletedAt = &now
		if err := store.Uploads().UpdateUpload(bgCtx, upload); err != nil {
			return fmt.Errorf("falha ao atualizar status do upload: %w", err)
		}
		return nil
	})

	if err != nil {
		_ = uc.failUpload(uploadID, err.Error())
		return
	}
}

func (uc *IngestMaintenanceUseCase) failUpload(uploadID string, reason string) error {
	bgCtx := context.Background()
	now := time.Now()
	upload, err := uc.uploadRepo.GetUploadByID(bgCtx, uploadID)
	if err != nil {
		return err
	}
	upload.Status = "failed"
	upload.ErrorLog = reason
	upload.CompletedAt = &now
	return uc.uploadRepo.UpdateUpload(bgCtx, upload)
}
