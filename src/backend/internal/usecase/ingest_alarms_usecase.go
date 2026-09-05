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

type IngestAlarmsUseCase struct {
	alarmRepo   port.AlarmRepository
	uploadRepo  port.UploadRepository
	storage     port.IngestionStorage
	alarmParser port.AlarmParser
	runner      port.JobRunner
	uow         port.UnitOfWork
	logger      port.Logger
}

func NewIngestAlarmsUseCase(
	alarmRepo port.AlarmRepository,
	uploadRepo port.UploadRepository,
	storage port.IngestionStorage,
	alarmParser port.AlarmParser,
	runner port.JobRunner,
	uow port.UnitOfWork,
	logger port.Logger,
) *IngestAlarmsUseCase {
	return &IngestAlarmsUseCase{
		alarmRepo:   alarmRepo,
		uploadRepo:  uploadRepo,
		storage:     storage,
		alarmParser: alarmParser,
		runner:      runner,
		uow:         uow,
		logger:      logger,
	}
}

func resolveAlarmFileType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == ".xlsx" || ext == ".xls" {
		return "alarms_xlsx"
	}
	return "alarms_csv"
}

func (uc *IngestAlarmsUseCase) checkDuplicateHash(ctx context.Context, fileHash, storageKey string) (string, bool) {
	existing, err := uc.uploadRepo.GetUploadByHash(ctx, fileHash)
	if err != nil || existing == nil {
		return "", false
	}
	if delErr := uc.storage.Delete(ctx, storageKey); delErr != nil && uc.logger != nil {
		uc.logger.Warn("falha ao remover arquivo temporario de upload duplicado", "storage_key", storageKey, "error", delErr.Error())
	}
	return existing.ID, true
}

func (uc *IngestAlarmsUseCase) handleCreateUploadError(ctx context.Context, storageKey, fileHash string, err error) (string, error) {
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

func (uc *IngestAlarmsUseCase) StartIngestion(ctx context.Context, fileName string, reader io.Reader) (string, error) {
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
		FileType: resolveAlarmFileType(fileName),
		FileHash: fileHash,
		Status:   "processing",
	}

	if err := uc.uploadRepo.CreateUpload(ctx, upload); err != nil {
		return uc.handleCreateUploadError(ctx, storageKey, fileHash, err)
	}

	uc.runner.Run(func() {
		uc.processFileAsync(upload.ID, storageKey, fileName)
	})

	return upload.ID, nil
}

func (uc *IngestAlarmsUseCase) recordFailure(uploadID, reason, logMsg string, cause error) {
	fullReason := reason
	if cause != nil {
		fullReason = reason + ": " + cause.Error()
	}
	if failErr := uc.failUpload(uploadID, fullReason); failErr != nil && uc.logger != nil {
		uc.logger.Error(logMsg, "upload_id", uploadID, "error", failErr.Error())
	}
}

func (uc *IngestAlarmsUseCase) readAndParseAlarms(ctx context.Context, storageKey, fileName, uploadID string) ([]port.RawAlarmRecord, bool) {
	rc, err := uc.storage.Open(ctx, storageKey)
	if err != nil {
		uc.recordFailure(uploadID, "falha ao abrir arquivo", "erro ao registrar falha de abertura do upload", err)
		return nil, false
	}
	defer rc.Close()

	rawRecords, err := uc.alarmParser.ParseAlarms(rc, fileName)
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

func convertRawAlarms(rawRecords []port.RawAlarmRecord, uploadID, fileName string) ([]*domain.Alarm, int) {
	source := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if source == "" {
		source = "telemetry"
	}

	alarms := make([]*domain.Alarm, 0, len(rawRecords))
	for _, raw := range rawRecords {
		alarm, err := domain.NewAlarm(domain.CreateAlarmParams{
			ID:                     uuid.New().String(),
			UploadID:               uploadID,
			Tag:                    raw.Tag,
			StationCode:            raw.StationCode,
			Platform:               raw.Platform,
			DoorNumber:             raw.DoorNumber,
			AlarmType:              raw.AlarmType,
			DurationSeconds:        raw.DurationSeconds,
			EventTimestamp:         raw.EventTimestamp,
			NormalizationTimestamp: raw.NormalizationTimestamp,
			Source:                 source,
		})
		if err != nil {
			continue
		}
		alarms = append(alarms, alarm)
	}
	return alarms, len(rawRecords)
}

func insertAlarmBatches(ctx context.Context, store port.UnitOfWorkStore, alarms []*domain.Alarm, batchSize int) (int, error) {
	inserted := 0
	for i := 0; i < len(alarms); i += batchSize {
		end := i + batchSize
		if end > len(alarms) {
			end = len(alarms)
		}
		chunk := alarms[i:end]
		if err := store.Alarms().BatchInsertAlarms(ctx, chunk); err != nil {
			return inserted, fmt.Errorf("falha ao inserir lote de alarmes: %w", err)
		}
		inserted += len(chunk)
	}
	return inserted, nil
}

func finalizeAlarmUpload(ctx context.Context, store port.UnitOfWorkStore, uploadID string, processed, inserted int) error {
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

func (uc *IngestAlarmsUseCase) processFileAsync(uploadID, storageKey, fileName string) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	defer func() {
		if delErr := uc.storage.Delete(bgCtx, storageKey); delErr != nil && uc.logger != nil {
			uc.logger.Warn("falha ao remover arquivo temporario", "storage_key", storageKey, "error", delErr.Error())
		}
	}()

	rawRecords, ok := uc.readAndParseAlarms(bgCtx, storageKey, fileName, uploadID)
	if !ok {
		return
	}

	allAlarms, processedCount := convertRawAlarms(rawRecords, uploadID, fileName)
	const batchSize = 500

	err := uc.uow.Do(bgCtx, func(store port.UnitOfWorkStore) error {
		insertedCount, err := insertAlarmBatches(bgCtx, store, allAlarms, batchSize)
		if err != nil {
			return err
		}
		return finalizeAlarmUpload(bgCtx, store, uploadID, processedCount, insertedCount)
	})

	if err != nil {
		uc.recordFailure(uploadID, err.Error(), "erro ao registrar falha de insercao em lote no upload", nil)
	}
}

func (uc *IngestAlarmsUseCase) failUpload(uploadID string, reason string) error {
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

func (uc *IngestAlarmsUseCase) GetStatus(ctx context.Context, uploadID string) (*UploadStatusOutputDTO, error) {
	u, err := uc.uploadRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		return nil, err
	}

	return &UploadStatusOutputDTO{
		ID:            u.ID,
		FileName:      u.FileName,
		FileType:      u.FileType,
		Status:        u.Status,
		RowsProcessed: u.RowsProcessed,
		RowsInserted:  u.RowsInserted,
		ErrorLog:      u.ErrorLog,
		CreatedAt:     u.CreatedAt,
		CompletedAt:   u.CompletedAt,
	}, nil
}
