package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
)

type IngestAlarmsUseCase struct {
	repo domain.AlarmRepository
}

func NewIngestAlarmsUseCase(repo domain.AlarmRepository) *IngestAlarmsUseCase {
	return &IngestAlarmsUseCase{repo: repo}
}

func (uc *IngestAlarmsUseCase) StartIngestion(ctx context.Context, fileName string, reader io.Reader) (string, error) {
	tmpFile, err := os.CreateTemp("", "alarm_upload_*.csv")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if _, err := io.Copy(tmpFile, tee); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))

	existing, err := uc.repo.GetUploadByHash(ctx, hashStr)
	if err == nil && existing != nil {
		os.Remove(tmpFile.Name())
		return existing.ID, domain.ErrDuplicateUpload
	}

	upload := &domain.FileUpload{
		ID:       uuid.New().String(),
		FileName: fileName,
		FileType: "alarms_csv",
		FileHash: hashStr,
		Status:   "processing",
	}

	if err := uc.repo.CreateUpload(ctx, upload); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	go uc.processFileAsync(upload.ID, tmpFile.Name())

	return upload.ID, nil
}

func (uc *IngestAlarmsUseCase) processFileAsync(uploadID string, filePath string) {
	defer os.Remove(filePath)

	file, err := os.Open(filePath)
	if err != nil {
		uc.failUpload(uploadID, "falha ao abrir arquivo temporario: "+err.Error())
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true

	headers, err := reader.Read()
	if err != nil {
		uc.failUpload(uploadID, "falha ao ler cabecalho: "+err.Error())
		return
	}

	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	tagIdx, hasTag := colMap["tag"]
	stationIdx, hasStation := colMap["estação"]
	if !hasStation {
		stationIdx, hasStation = colMap["estacao"]
	}
	doorIdx, hasDoor := colMap["porta"]
	alarmIdx, hasAlarm := colMap["alarme"]
	durIdx, hasDur := colMap["tempo_normalizacao_seg"]
	normIdx, hasNorm := colMap["horário da normalizacao"]
	if !hasNorm {
		normIdx, hasNorm = colMap["horario da normalizacao"]
	}
	platIdx, hasPlat := colMap["plataforma_extraida"]
	timeIdx, hasTime := colMap["horário do alarme"]
	if !hasTime {
		timeIdx, hasTime = colMap["horario do alarme"]
	}

	if !hasTag || !hasStation || !hasDoor || !hasAlarm || !hasTime {
		uc.failUpload(uploadID, "colunas obrigatorias ausentes no arquivo CSV de alarmes")
		return
	}

	const batchSize = 500
	var batch []*domain.Alarm
	processedCount := 0
	insertedCount := 0
	bgCtx := context.Background()

	for {
		row, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		processedCount++

		doorNum, _ := strconv.Atoi(strings.TrimSpace(strings.Split(row[doorIdx], ".")[0]))
		durSec := 0
		if hasDur && durIdx < len(row) {
			durSec, _ = strconv.Atoi(strings.TrimSpace(row[durIdx]))
		}

		platform := ""
		if hasPlat && platIdx < len(row) {
			platform = strings.TrimSpace(row[platIdx])
		}

		normTime := ""
		if hasNorm && normIdx < len(row) {
			normTime = strings.TrimSpace(row[normIdx])
		}

		alarm := &domain.Alarm{
			ID:                     uuid.New().String(),
			UploadID:               uploadID,
			Tag:                    strings.TrimSpace(row[tagIdx]),
			StationCode:            strings.TrimSpace(row[stationIdx]),
			Platform:               platform,
			DoorNumber:             doorNum,
			AlarmType:              strings.TrimSpace(row[alarmIdx]),
			DurationSeconds:        durSec,
			EventTimestamp:         strings.TrimSpace(row[timeIdx]),
			NormalizationTimestamp: normTime,
			Source:                 "csv",
			CreatedAt:              time.Now(),
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

func (uc *IngestAlarmsUseCase) failUpload(uploadID string, reason string) {
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

func (uc *IngestAlarmsUseCase) GetStatus(ctx context.Context, uploadID string) (*UploadStatusOutputDTO, error) {
	u, err := uc.repo.GetUploadByID(ctx, uploadID)
	if err != nil {
		return nil, err
	}

	var completedStr *string
	if u.CompletedAt != nil {
		s := u.CompletedAt.Format("2006-01-02 15:04:05")
		completedStr = &s
	}

	return &UploadStatusOutputDTO{
		ID:            u.ID,
		FileName:      u.FileName,
		FileType:      u.FileType,
		Status:        u.Status,
		RowsProcessed: u.RowsProcessed,
		RowsInserted:  u.RowsInserted,
		ErrorLog:      u.ErrorLog,
		CreatedAt:     u.CreatedAt.Format("2006-01-02 15:04:05"),
		CompletedAt:   completedStr,
	}, nil
}
