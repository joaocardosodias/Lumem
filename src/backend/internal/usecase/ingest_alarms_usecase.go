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
	"sync"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/google/uuid"
)

type IngestAlarmsUseCase struct {
	repo domain.AlarmRepository
	wg   *sync.WaitGroup
}

func NewIngestAlarmsUseCase(repo domain.AlarmRepository, wg ...*sync.WaitGroup) *IngestAlarmsUseCase {
	var w *sync.WaitGroup
	if len(wg) > 0 {
		w = wg[0]
	}
	return &IngestAlarmsUseCase{repo: repo, wg: w}
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

	if uc.wg != nil {
		uc.wg.Add(1)
	}
	go func() {
		if uc.wg != nil {
			defer uc.wg.Done()
		}
		uc.processFileAsync(upload.ID, tmpFile.Name())
	}()

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

	tagIdx, hasTag := getFirstIndex(colMap, "tag", "tag_alarme")
	stationIdx, hasStation := getFirstIndex(colMap, "estação", "estacao", "estacao_original", "estacao_extraida")
	doorIdx, hasDoor := getFirstIndex(colMap, "porta", "porta_original", "porta_extraida")
	alarmIdx, hasAlarm := getFirstIndex(colMap, "alarme", "tipo_alarme", "tipo_evento", "tipo_evento_extraido")
	durIdx, hasDur := getFirstIndex(colMap, "tempo_normalizacao_seg", "duracao_calculada_seg", "duracao_seg")
	normIdx, hasNorm := getFirstIndex(colMap, "horário da normalizacao", "horario da normalizacao", "horario_normalizacao", "dt_normalizacao")
	platIdx, hasPlat := getFirstIndex(colMap, "plataforma_extraida", "plataforma")
	timeIdx, hasTime := getFirstIndex(colMap, "horário do alarme", "horario do alarme", "horario_alarme", "dt_alarme")

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
			durSec, _ = strconv.Atoi(strings.TrimSpace(strings.Split(row[durIdx], ".")[0]))
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
