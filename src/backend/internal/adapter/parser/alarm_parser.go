package parser

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/g04/backend/internal/usecase/port"
)

var _ port.AlarmParser = (*DefaultAlarmParser)(nil)

type DefaultAlarmParser struct{}

func NewDefaultAlarmParser() *DefaultAlarmParser {
	return &DefaultAlarmParser{}
}

func (p *DefaultAlarmParser) ParseAlarms(r io.Reader, format string) (records []port.RawAlarmRecord, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panico recuperado durante leitura de alarmes: %v", rec)
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

	tagIdx, hasTag := getFirstIndex(colMap, "tag", "tag_alarme")
	stationIdx, hasStation := getFirstIndex(colMap, "estação", "estacao", "estacao_original", "estacao_extraida")
	doorIdx, hasDoor := getFirstIndex(colMap, "porta", "porta_original", "porta_extraida")
	alarmIdx, hasAlarm := getFirstIndex(colMap, "alarme", "tipo_alarme", "tipo_evento", "tipo_evento_extraido")
	durIdx, hasDur := getFirstIndex(colMap, "tempo_normalizacao_seg", "duracao_calculada_seg", "duracao_seg")
	normIdx, hasNorm := getFirstIndex(colMap, "horário da normalizacao", "horario da normalizacao", "horario_normalizacao", "dt_normalizacao")
	platIdx, hasPlat := getFirstIndex(colMap, "plataforma_extraida", "plataforma")
	timeIdx, hasTime := getFirstIndex(colMap, "horário do alarme", "horario do alarme", "horario_alarme", "dt_alarme")

	if !hasTag || !hasStation || !hasDoor || !hasAlarm || !hasTime {
		return nil, errors.New("colunas obrigatorias ausentes no arquivo de alarmes")
	}

	records = make([]port.RawAlarmRecord, 0, len(rows)-1)
	for _, row := range rows[1:] {
		doorRaw := safeGetCell(row, doorIdx)
		doorNum, _ := strconv.Atoi(strings.TrimSpace(strings.Split(doorRaw, ".")[0]))

		durSec := 0
		if hasDur {
			durRaw := safeGetCell(row, durIdx)
			durSec, _ = strconv.Atoi(strings.TrimSpace(strings.Split(durRaw, ".")[0]))
		}

		tag := safeGetCell(row, tagIdx)
		platform := ""
		if hasPlat {
			platform = safeGetCell(row, platIdx)
		}
		if platform == "" {
			if strings.Contains(tag, "-P1-") || strings.Contains(tag, "-P1") {
				platform = "P1"
			} else if strings.Contains(tag, "-P2-") || strings.Contains(tag, "-P2") {
				platform = "P2"
			}
		}

		normTime := ""
		if hasNorm {
			normTime = safeGetCell(row, normIdx)
		}

		records = append(records, port.RawAlarmRecord{
			Tag:                    tag,
			StationCode:            safeGetCell(row, stationIdx),
			Platform:               platform,
			DoorNumber:             doorNum,
			AlarmType:              safeGetCell(row, alarmIdx),
			DurationSeconds:        durSec,
			EventTimestamp:         safeGetCell(row, timeIdx),
			NormalizationTimestamp: normTime,
		})
	}

	return records, nil
}
