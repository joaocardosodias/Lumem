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

type IngestAlarmsUseCase struct {
	alarmRepo   port.AlarmRepository
	uploadRepo  port.UploadRepository
	storage     port.IngestionStorage
	alarmParser port.AlarmParser
	runner      port.JobRunner
	uow         port.UnitOfWork
}

func NewIngestAlarmsUseCase(
	alarmRepo port.AlarmRepository,
	uploadRepo port.UploadRepository,
	storage port.IngestionStorage,
	alarmParser port.AlarmParser,
	runner port.JobRunner,
	uow port.UnitOfWork,
) *IngestAlarmsUseCase {
	return &IngestAlarmsUseCase{
		alarmRepo:   alarmRepo,
		uploadRepo:  uploadRepo,
		storage:     storage,
		alarmParser: alarmParser,
		runner:      runner,
		uow:         uow,
	}
}

func (uc *IngestAlarmsUseCase) StartIngestion(ctx context.Context, fileName string, reader io.Reader) (string, error) {
	storageKey, fileHash, err := uc.storage.Store(ctx, fileName, reader)
	if err != nil {
		return "", err
	}

	existing, err := uc.uploadRepo.GetUploadByHash(ctx, fileHash)
	if err == nil && existing != nil {
		_ = uc.storage.Delete(ctx, storageKey)
		return existing.ID, domain.ErrDuplicateUpload
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	fileType := "alarms_csv"
	if ext == ".xlsx" || ext == ".xls" {
		fileType = "alarms_xlsx"
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
		uc.processFileAsync(upload.ID, storageKey, fileName)
	})

	return upload.ID, nil
}

func (uc *IngestAlarmsUseCase) processFileAsync(uploadID, storageKey, fileName string) {
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

	rawRecords, err := uc.alarmParser.ParseAlarms(rc, fileName)
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
		source = "telemetry"
	}

	var allAlarms []*domain.Alarm
	for _, raw := range rawRecords {
		processedCount++

		alarm, err := domain.NewAlarm(
			uuid.New().String(),
			uploadID,
			raw.Tag,
			raw.StationCode,
			raw.Platform,
			raw.DoorNumber,
			raw.AlarmType,
			raw.DurationSeconds,
			raw.EventTimestamp,
			raw.NormalizationTimestamp,
			source,
		)
		if err != nil {
			continue
		}

		allAlarms = append(allAlarms, alarm)
	}

	insertedCount := 0
	err = uc.uow.Do(bgCtx, func(store port.UnitOfWorkStore) error {
		for i := 0; i < len(allAlarms); i += batchSize {
			end := i + batchSize
			if end > len(allAlarms) {
				end = len(allAlarms)
			}
			chunk := allAlarms[i:end]
			if err := store.Alarms().BatchInsertAlarms(bgCtx, chunk); err != nil {
				return fmt.Errorf("falha ao inserir lote de alarmes: %w", err)
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

func (uc *IngestAlarmsUseCase) failUpload(uploadID string, reason string) error {
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
