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
	queue       port.IngestionJobQueue
	uow         port.UnitOfWork
	logger      port.Logger
}

func NewIngestAlarmsUseCaseWithQueue(
	alarmRepo port.AlarmRepository,
	uploadRepo port.UploadRepository,
	storage port.IngestionStorage,
	alarmParser port.AlarmParser,
	queue port.IngestionJobQueue,
	uow port.UnitOfWork,
	logger port.Logger,
) *IngestAlarmsUseCase {
	return &IngestAlarmsUseCase{
		alarmRepo:   alarmRepo,
		uploadRepo:  uploadRepo,
		storage:     storage,
		alarmParser: alarmParser,
		queue:       queue,
		uow:         uow,
		logger:      logger,
	}
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

	job := port.IngestionJob{
		Type:       port.IngestionJobAlarms,
		UploadID:   upload.ID,
		StorageKey: storageKey,
		FileName:   fileName,
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
			uc.processFileAsync(upload.ID, storageKey, fileName)
		})
	}

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

func (uc *IngestAlarmsUseCase) readAndParseAlarms(ctx context.Context, storageKey, fileName string) ([]port.RawAlarmRecord, error) {
	rc, err := uc.storage.Open(ctx, storageKey)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo: %w", err)
	}
	defer rc.Close()

	rawRecords, err := uc.alarmParser.ParseAlarms(rc, fileName)
	if err != nil {
		return nil, permanentIngestionFailure("falha ao analisar arquivo: %w", err)
	}

	if len(rawRecords) == 0 {
		return nil, permanentIngestionFailure("arquivo vazio ou sem linhas de dados validas")
	}

	return rawRecords, nil
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

func (uc *IngestAlarmsUseCase) deleteStoredFile(ctx context.Context, storageKey string) error {
	if err := uc.storage.Delete(ctx, storageKey); err != nil {
		if uc.logger != nil {
			uc.logger.Warn("falha ao remover arquivo de ingestao", "storage_key", storageKey, "error", err.Error())
		}
		return err
	}
	return nil
}

func (uc *IngestAlarmsUseCase) ProcessJob(ctx context.Context, job port.IngestionJob) error {
	rawRecords, err := uc.readAndParseAlarms(ctx, job.StorageKey, job.FileName)
	if err != nil {
		var permanentErr *permanentIngestionError
		if errors.As(err, &permanentErr) {
			if failErr := uc.failUploadContext(ctx, job.UploadID, err.Error()); failErr != nil {
				return fmt.Errorf("registrar falha permanente do upload: %w", failErr)
			}
			_ = uc.deleteStoredFile(context.Background(), job.StorageKey)
			return nil
		}
		return err
	}

	allAlarms, processedCount := convertRawAlarms(rawRecords, job.UploadID, job.FileName)
	const batchSize = 500

	err = uc.uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		if err := store.Alarms().DeleteByUploadID(ctx, job.UploadID); err != nil {
			return fmt.Errorf("falha ao preparar nova tentativa de alarmes: %w", err)
		}
		insertedCount, err := insertAlarmBatches(ctx, store, allAlarms, batchSize)
		if err != nil {
			return err
		}
		return finalizeAlarmUpload(ctx, store, job.UploadID, processedCount, insertedCount)
	})
	if err != nil {
		return err
	}

	_ = uc.deleteStoredFile(context.Background(), job.StorageKey)
	return nil
}

func (uc *IngestAlarmsUseCase) FailJob(ctx context.Context, job port.IngestionJob, reason string) error {
	if err := uc.failUploadContext(ctx, job.UploadID, reason); err != nil {
		return err
	}
	_ = uc.deleteStoredFile(context.Background(), job.StorageKey)
	return nil
}

func (uc *IngestAlarmsUseCase) processFileAsync(uploadID, storageKey, fileName string) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	job := port.IngestionJob{Type: port.IngestionJobAlarms, UploadID: uploadID, StorageKey: storageKey, FileName: fileName}
	if err := uc.ProcessJob(bgCtx, job); err != nil {
		_ = uc.FailJob(context.Background(), job, err.Error())
	}
}

func (uc *IngestAlarmsUseCase) failUpload(uploadID string, reason string) error {
	return uc.failUploadContext(context.Background(), uploadID, reason)
}

func (uc *IngestAlarmsUseCase) failUploadContext(ctx context.Context, uploadID string, reason string) error {
	now := time.Now()
	upload, err := uc.uploadRepo.GetUploadByID(ctx, uploadID)
	if err != nil {
		if uc.logger != nil {
			uc.logger.Error("falha ao obter upload para marcar como failed", "upload_id", uploadID, "reason", reason, "error", err.Error())
		}
		return err
	}
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
