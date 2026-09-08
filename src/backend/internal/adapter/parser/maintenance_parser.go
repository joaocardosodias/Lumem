package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/g04/backend/internal/usecase/port"
)

var _ port.MaintenanceParser = (*DefaultMaintenanceParser)(nil)

type DefaultMaintenanceParser struct{}

func NewDefaultMaintenanceParser() *DefaultMaintenanceParser {
	return &DefaultMaintenanceParser{}
}

type maintenanceColumnIndices struct {
	orderIdx, typeIdx, locIdx, stationIdx, startIdx          int
	endIdx, durIdx, descIdx, symptomIdx, causeIdx, actionIdx int
	hasOrder, hasType, hasLoc, hasStation, hasStart          bool
	hasEnd, hasDur, hasDesc, hasSymptom, hasCause, hasAction bool
}

func extractMaintenanceIndices(headers []string) (maintenanceColumnIndices, error) {
	colMap := make(map[string]int, len(headers))
	for i, h := range headers {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}
	var cols maintenanceColumnIndices
	cols.orderIdx, cols.hasOrder = getFirstIndex(colMap, "ordem")
	cols.typeIdx, cols.hasType = getFirstIndex(colMap, "tipo de ordem", "tipo_de_nota", "tipo_de_ordem")
	cols.locIdx, cols.hasLoc = getFirstIndex(colMap, "local de instalação", "local_de_instalacao")
	cols.stationIdx, cols.hasStation = getFirstIndex(colMap, "estacao", "estação", "centro_de_trabalho_responsavel")
	cols.startIdx, cols.hasStart = getFirstIndex(colMap, "inicio_avaria_datetime", "dt_inicio", "data-base do início")
	cols.endIdx, cols.hasEnd = getFirstIndex(colMap, "fim_avaria_datetime", "dt_fim", "data-base do fim")
	cols.durIdx, cols.hasDur = getFirstIndex(colMap, "duracao_avaria_horas", "duracao_h")
	cols.descIdx, cols.hasDesc = getFirstIndex(colMap, "descricao", "texto breve", "texto_longo_da_ordem")
	cols.symptomIdx, cols.hasSymptom = getFirstIndex(colMap, "sintoma")
	cols.causeIdx, cols.hasCause = getFirstIndex(colMap, "causa")
	cols.actionIdx, cols.hasAction = getFirstIndex(colMap, "acao", "ação")

	if !cols.hasStart {
		return cols, errors.New("coluna obrigatoria de data de inicio nao encontrada")
	}
	return cols, nil
}

func safeColumnValue(row []string, has bool, idx int) string {
	if !has {
		return ""
	}
	return safeGetCell(row, idx)
}

func parseDuration(row []string, has bool, idx int) float64 {
	if !has {
		return 0.0
	}
	val := strings.ReplaceAll(safeGetCell(row, idx), ",", ".")
	duration, _ := strconv.ParseFloat(val, 64)
	return duration
}

func resolveOrderType(row []string, hasType bool, typeIdx int, defaultOrderType string) string {
	orderType := defaultOrderType
	if orderType == "" && hasType {
		orderType = safeGetCell(row, typeIdx)
	}
	if orderType == "" {
		return "MANUTENCAO"
	}
	return orderType
}

func parseMaintenanceRow(row []string, cols maintenanceColumnIndices, defaultOrderType string) port.RawMaintenanceRecord {
	return port.RawMaintenanceRecord{
		SAPOrderNumber:       safeColumnValue(row, cols.hasOrder, cols.orderIdx),
		OrderType:            resolveOrderType(row, cols.hasType, cols.typeIdx, defaultOrderType),
		InstallationLocation: safeColumnValue(row, cols.hasLoc, cols.locIdx),
		StationCode:          safeColumnValue(row, cols.hasStation, cols.stationIdx),
		StartTimestamp:       safeGetCell(row, cols.startIdx),
		EndTimestamp:         safeColumnValue(row, cols.hasEnd, cols.endIdx),
		DurationHours:        parseDuration(row, cols.hasDur, cols.durIdx),
		Description:          safeColumnValue(row, cols.hasDesc, cols.descIdx),
		Symptom:              safeColumnValue(row, cols.hasSymptom, cols.symptomIdx),
		Cause:                safeColumnValue(row, cols.hasCause, cols.causeIdx),
		ActionTaken:          safeColumnValue(row, cols.hasAction, cols.actionIdx),
	}
}

// ParseMaintenance collects small inputs; ingestion uses StreamMaintenance.
func (p *DefaultMaintenanceParser) ParseMaintenance(r io.Reader, format, orderType string) (records []port.RawMaintenanceRecord, err error) {
	err = p.StreamMaintenance(context.Background(), r, format, orderType, func(row port.RawMaintenanceRecord) error { records = append(records, row); return nil })
	return
}

func (p *DefaultMaintenanceParser) StreamMaintenance(ctx context.Context, r io.Reader, format, orderType string, yield func(port.RawMaintenanceRecord) error) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panico recuperado durante leitura de manutencao: %v", rec)
		}
	}()
	var cols maintenanceColumnIndices
	count := 0
	err = walkRows(ctx, r, format, func(row []string) error {
		count++
		if count == 1 {
			var err error
			cols, err = extractMaintenanceIndices(row)
			return err
		}
		return yield(parseMaintenanceRow(row, cols, orderType))
	})
	if err != nil {
		return err
	}
	if count < 2 {
		return errors.New("arquivo vazio ou sem linhas de dados")
	}
	return nil
}
