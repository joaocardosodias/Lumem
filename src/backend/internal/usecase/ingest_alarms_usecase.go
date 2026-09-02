package usecase

import (
	"context"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/g04/backend/internal/domain"
)

type IngestAlarmsUseCase struct {
	repo domain.AlarmRepository
}

func NewIngestAlarmsUseCase(repo domain.AlarmRepository) *IngestAlarmsUseCase {
	return &IngestAlarmsUseCase{repo: repo}
}

func (uc *IngestAlarmsUseCase) StartIngestion(ctx context.Context, fileName string, reader io.Reader) (int64, error) {
	tmpFile, err := os.CreateTemp("", "alarms_upload_*.csv")
	if err != nil {
		return 0, err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		os.Remove(tmpFile.Name())
		return 0, err
	}

	upload := &domain.FileUpload{
		FileName: fileName,
		FileType: "alarms_csv",
		Status:   "processing",
	}

	if err := uc.repo.CreateUpload(ctx, upload); err != nil {
		os.Remove(tmpFile.Name())
		return 0, err
	}

	go uc.processFileAsync(upload.ID, tmpFile.Name())

	return upload.ID, nil
}

func (uc *IngestAlarmsUseCase) GetStatus(ctx context.Context, uploadID int64) (*domain.FileUpload, error) {
	return uc.repo.GetUploadByID(ctx, uploadID)
}

func (uc *IngestAlarmsUseCase) processFileAsync(uploadID int64, filePath string) {
	defer os.Remove(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		uc.failUpload(uploadID, err.Error())
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.ReuseRecord = true

	headers, err := reader.Read()
	if err != nil {
		uc.failUpload(uploadID, "erro ao ler cabecalho csv: "+err.Error())
		return
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[h] = i
	}

	tagIdx, hasTag := colMap["TAG"]
	timeIdx, hasTime := colMap["Horário do Alarme"]
	stationIdx, hasStation := colMap["Estação"]
	alarmTypeIdx, hasType := colMap["Alarme"]
	normTimeIdx, hasNormTime := colMap["Horário da Normalizacao"]
	durIdx, hasDur := colMap["tempo_normalizacao_seg"]
	platIdx, hasPlat := colMap["plataforma_extraida"]
	doorIdx, hasDoor := colMap["porta_extraida"]

	if !hasTag || !hasTime || !hasStation || !hasType {
		uc.failUpload(uploadID, "colunas obrigatorias ausentes no csv")
		return
	}

	const batchSize = 500
	var batch []*domain.Alarm
	processedCount := 0
	insertedCount := 0

	bgCtx := context.Background()

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		processedCount++

		alarm := &domain.Alarm{
			UploadID:       uploadID,
			Tag:            record[tagIdx],
			EventTimestamp: record[timeIdx],
			StationCode:    record[stationIdx],
			AlarmType:      record[alarmTypeIdx],
			Source:         "spreadsheet",
		}

		if hasNormTime && normTimeIdx < len(record) {
			alarm.NormalizationTimestamp = record[normTimeIdx]
		}

		if hasDur && durIdx < len(record) {
			if dur, err := strconv.ParseFloat(record[durIdx], 64); err == nil {
				alarm.DurationSeconds = dur
			}
		}

		if hasPlat && platIdx < len(record) {
			alarm.Platform = record[platIdx]
		}

		if hasDoor && doorIdx < len(record) {
			if d, err := strconv.ParseFloat(record[doorIdx], 64); err == nil {
				alarm.DoorNumber = int(d)
			}
		}

		batch = append(batch, alarm)

		if len(batch) >= batchSize {
			if err := uc.repo.BatchInsertAlarms(bgCtx, batch); err != nil {
				log.Printf("erro ao inserir lote: %v", err)
			} else {
				insertedCount += len(batch)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := uc.repo.BatchInsertAlarms(bgCtx, batch); err != nil {
			log.Printf("erro ao inserir lote final: %v", err)
		} else {
			insertedCount += len(batch)
		}
	}

	now := time.Now()
	upload, err := uc.repo.GetUploadByID(bgCtx, uploadID)
	if err == nil {
		upload.RowsProcessed = processedCount
		upload.RowsInserted = insertedCount
		upload.Status = "completed"
		upload.CompletedAt = &now
		_ = uc.repo.UpdateUpload(bgCtx, upload)
	}
}

func (uc *IngestAlarmsUseCase) failUpload(uploadID int64, reason string) {
	bgCtx := context.Background()
	now := time.Now()
	upload, err := uc.repo.GetUploadByID(bgCtx, uploadID)
	if err == nil {
		upload.Status = "failed"
		upload.ErrorLog = reason
		upload.CompletedAt = &now
		_ = uc.repo.UpdateUpload(bgCtx, upload)
	}
}
