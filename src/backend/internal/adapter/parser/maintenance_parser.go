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
		return nil, errors.New("coluna obrigatoria de data de inicio nao encontrada")
	}

	records = make([]port.RawMaintenanceRecord, 0, len(rows)-1)
	for _, row := range rows[1:] {
		orderType := defaultOrderType
		if orderType == "" && hasType {
			orderType = safeGetCell(row, typeIdx)
		}
		if orderType == "" {
			orderType = "MANUTENCAO"
		}

		sapOrder := ""
		if hasOrder {
			sapOrder = safeGetCell(row, orderIdx)
		}

		location := ""
		if hasLoc {
			location = safeGetCell(row, locIdx)
		}

		stationCode := ""
		if hasStation {
			stationCode = safeGetCell(row, stationIdx)
		}

		startTime := safeGetCell(row, startIdx)

		endTime := ""
		if hasEnd {
			endTime = safeGetCell(row, endIdx)
		}

		duration := 0.0
		if hasDur {
			val := strings.ReplaceAll(safeGetCell(row, durIdx), ",", ".")
			duration, _ = strconv.ParseFloat(val, 64)
		}

		desc := ""
		if hasDesc {
			desc = safeGetCell(row, descIdx)
		}

		symptom := ""
		if hasSymptom {
			symptom = safeGetCell(row, symptomIdx)
		}

		cause := ""
		if hasCause {
			cause = safeGetCell(row, causeIdx)
		}

		action := ""
		if hasAction {
			action = safeGetCell(row, actionIdx)
		}

		records = append(records, port.RawMaintenanceRecord{
			SAPOrderNumber:       sapOrder,
			OrderType:            orderType,
			InstallationLocation: location,
			StationCode:          stationCode,
			StartTimestamp:       startTime,
			EndTimestamp:         endTime,
			DurationHours:        duration,
			Description:          desc,
			Symptom:              symptom,
			Cause:                cause,
			ActionTaken:          action,
		})
	}

	return records, nil
}
