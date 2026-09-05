package parser

import (
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

func (p *DefaultMaintenanceParser) ParseMaintenance(r io.Reader, format string, defaultOrderType string) (records []port.RawMaintenanceRecord, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panico recuperado durante leitura de manutencao: %v", rec)
		}
	}()

	rows, err := readRowsFromReader(r, format)
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, errors.New("arquivo vazio ou sem linhas de dados")
	}

	cols, err := extractMaintenanceIndices(rows[0])
	if err != nil {
		return nil, err
	}

	records = make([]port.RawMaintenanceRecord, 0, len(rows)-1)
	for _, row := range rows[1:] {
		records = append(records, parseMaintenanceRow(row, cols, defaultOrderType))
	}

	return records, nil
}
