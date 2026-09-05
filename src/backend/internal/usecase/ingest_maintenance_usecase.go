package usecase

import (
	"context"
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
}

func NewIngestMaintenanceUseCase(
	maintRepo port.MaintenanceRepository,
	uploadRepo port.UploadRepository,
	storage port.IngestionStorage,
	maintParser port.MaintenanceParser,
	runner port.JobRunner,
) *IngestMaintenanceUseCase {
	return &IngestMaintenanceUseCase{
		maintRepo:   maintRepo,
		uploadRepo:  uploadRepo,
		storage:     storage,
		maintParser: maintParser,
		runner:      runner,
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
	var batch []*domain.MaintenanceEvent
	processedCount := 0
	insertedCount := 0
	source := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if source == "" {
		source = "maintenance"
	}

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

		batch = append(batch, event)

		if len(batch) >= batchSize {
			if err := uc.maintRepo.BatchInsertMaintenance(bgCtx, batch); err != nil {
				_ = uc.maintRepo.DeleteByUploadID(bgCtx, uploadID)
				_ = uc.failUpload(uploadID, "falha ao inserir lote de manutencao: "+err.Error())
				return
			}
			insertedCount += len(batch)
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := uc.maintRepo.BatchInsertMaintenance(bgCtx, batch); err != nil {
			_ = uc.maintRepo.DeleteByUploadID(bgCtx, uploadID)
			_ = uc.failUpload(uploadID, "falha ao inserir lote final de manutencao: "+err.Error())
			return
		}
		insertedCount += len(batch)
	}

	now := time.Now()
	upload, err := uc.uploadRepo.GetUploadByID(bgCtx, uploadID)
	if err == nil {
		upload.RowsProcessed = processedCount
		upload.RowsInserted = insertedCount
		upload.Status = "completed"
		upload.CompletedAt = &now
		if err := uc.uploadRepo.UpdateUpload(bgCtx, upload); err != nil {
			_ = uc.failUpload(uploadID, "falha ao atualizar status do upload: "+err.Error())
		}
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
