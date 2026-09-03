package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/g04/backend/internal/domain"
)

type IngestMaintenanceUseCase struct {
	maintRepo  domain.MaintenanceRepository
	uploadRepo domain.AlarmRepository
	parser     domain.SpreadsheetParser
}

func NewIngestMaintenanceUseCase(
	maintRepo domain.MaintenanceRepository,
	uploadRepo domain.AlarmRepository,
	parser domain.SpreadsheetParser,
) *IngestMaintenanceUseCase {
	return &IngestMaintenanceUseCase{
		maintRepo:  maintRepo,
		uploadRepo: uploadRepo,
		parser:     parser,
	}
}

func (uc *IngestMaintenanceUseCase) StartIngestion(ctx context.Context, fileName string, reader io.Reader, explicitOrderType string) (int64, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	pattern := "maint_upload_*" + ext
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return 0, err
	}
	defer tmpFile.Close()

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if _, err := io.Copy(tmpFile, tee); err != nil {
		os.Remove(tmpFile.Name())
		return 0, err
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))

	existing, err := uc.uploadRepo.GetUploadByHash(ctx, hashStr)
	if err == nil && existing != nil {
		os.Remove(tmpFile.Name())
		return existing.ID, domain.ErrDuplicateUpload
	}

	fileType := "maintenance"
	if explicitOrderType != "" {
		fileType = "maintenance_" + strings.ToLower(explicitOrderType)
	}

	upload := &domain.FileUpload{
		FileName: fileName,
		FileType: fileType,
		FileHash: hashStr,
		Status:   "processing",
	}

	if err := uc.uploadRepo.CreateUpload(ctx, upload); err != nil {
		os.Remove(tmpFile.Name())
		return 0, err
	}

	go uc.processFileAsync(upload.ID, tmpFile.Name(), ext, explicitOrderType)

	return upload.ID, nil
}

func (uc *IngestMaintenanceUseCase) processFileAsync(uploadID int64, filePath, ext, explicitOrderType string) {
	defer os.Remove(filePath)

	rows, err := uc.parser.ReadRows(filePath, ext)
	if err != nil {
		uc.failUpload(uploadID, err.Error())
		return
	}

	if len(rows) < 2 {
		uc.failUpload(uploadID, "arquivo vazio ou sem linhas de dados")
		return
	}

	headers := rows[0]
	colMap := make(map[string]int)
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	orderIdx, hasOrder := getFirstIndex(colMap, "ordem")
	typeIdx, hasType := getFirstIndex(colMap, "tipo de ordem", "tipo_de_nota", "tipo_de_ordem")
	locIdx, hasLoc := getFirstIndex(colMap, "local de instalação", "local_de_instalacao")
	stationIdx, hasStation := getFirstIndex(colMap, "estacao", "estação", "centro_de_trabalho_responsavel")
	startIdx, hasStart := getFirstIndex(colMap, "inicio_avaria_datetime", "dt_inicio", "data-base do início")
	endIdx, hasEnd := getFirstIndex(colMap, "fim_avaria_datetime", "dt_fim", "data-base do fim")
	durIdx, hasDur := getFirstIndex(colMap, "duracao_avaria_horas", "duracao_h")
	descIdx, hasDesc := getFirstIndex(colMap, "descricao", "texto breve", "texto_longo_da_ordem")
	symptomIdx, hasSymptom := getFirstIndex(colMap, "sintoma")
	causeIdx, hasCause := getFirstIndex(colMap, "causa")
	actionIdx, hasAction := getFirstIndex(colMap, "acao", "ação")

	if !hasStart {
		uc.failUpload(uploadID, "coluna obrigatoria de data de inicio nao encontrada")
		return
	}

	const batchSize = 500
	var batch []*domain.MaintenanceEvent
	processedCount := 0
	insertedCount := 0
	bgCtx := context.Background()

	for _, row := range rows[1:] {
		processedCount++

		orderType := explicitOrderType
		if orderType == "" && hasType && typeIdx < len(row) {
			orderType = strings.TrimSpace(row[typeIdx])
		}
		if orderType == "" {
			orderType = "MANUTENCAO"
		}

		event := &domain.MaintenanceEvent{
			UploadID:  uploadID,
			OrderType: orderType,
			Source:    "spreadsheet",
		}

		if hasOrder && orderIdx < len(row) {
			event.SAPOrderNumber = strings.TrimSpace(row[orderIdx])
		}

		if hasLoc && locIdx < len(row) {
			event.InstallationLocation = strings.TrimSpace(row[locIdx])
		}

		if hasStation && stationIdx < len(row) {
			event.StationCode = strings.TrimSpace(row[stationIdx])
		}

		if startIdx < len(row) {
			event.StartTimestamp = strings.TrimSpace(row[startIdx])
		}

		if hasEnd && endIdx < len(row) {
			event.EndTimestamp = strings.TrimSpace(row[endIdx])
		}

		if hasDur && durIdx < len(row) {
			if d, err := strconv.ParseFloat(strings.TrimSpace(row[durIdx]), 64); err == nil {
				event.DurationHours = d
			}
		}

		if hasDesc && descIdx < len(row) {
			event.Description = strings.TrimSpace(row[descIdx])
		}

		if hasSymptom && symptomIdx < len(row) {
			event.Symptom = strings.TrimSpace(row[symptomIdx])
		}

		if hasCause && causeIdx < len(row) {
			event.Cause = strings.TrimSpace(row[causeIdx])
		}

		if hasAction && actionIdx < len(row) {
			event.ActionTaken = strings.TrimSpace(row[actionIdx])
		}

		batch = append(batch, event)

		if len(batch) >= batchSize {
			if err := uc.maintRepo.BatchInsertMaintenance(bgCtx, batch); err != nil {
				log.Printf("erro ao inserir lote de manutencao: %v", err)
			} else {
				insertedCount += len(batch)
			}
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := uc.maintRepo.BatchInsertMaintenance(bgCtx, batch); err != nil {
			log.Printf("erro ao inserir lote final de manutencao: %v", err)
		} else {
			insertedCount += len(batch)
		}
	}

	now := time.Now()
	upload, err := uc.uploadRepo.GetUploadByID(bgCtx, uploadID)
	if err == nil {
		upload.RowsProcessed = processedCount
		upload.RowsInserted = insertedCount
		upload.Status = "completed"
		upload.CompletedAt = &now
		_ = uc.uploadRepo.UpdateUpload(bgCtx, upload)
	}
}

func (uc *IngestMaintenanceUseCase) failUpload(uploadID int64, reason string) {
	bgCtx := context.Background()
	now := time.Now()
	upload, err := uc.uploadRepo.GetUploadByID(bgCtx, uploadID)
	if err == nil {
		upload.Status = "failed"
		upload.ErrorLog = reason
		upload.CompletedAt = &now
		_ = uc.uploadRepo.UpdateUpload(bgCtx, upload)
	}
}

func getFirstIndex(colMap map[string]int, names ...string) (int, bool) {
	for _, n := range names {
		if idx, exists := colMap[n]; exists {
			return idx, true
		}
	}
	return 0, false
}
