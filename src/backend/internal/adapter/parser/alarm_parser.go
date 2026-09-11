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

var _ port.AlarmParser = (*DefaultAlarmParser)(nil)

type DefaultAlarmParser struct{}

func NewDefaultAlarmParser() *DefaultAlarmParser {
	return &DefaultAlarmParser{}
}

// ParseAlarms is a convenience collector for small inputs; ingestion uses StreamAlarms.
func (p *DefaultAlarmParser) ParseAlarms(r io.Reader, format string) (records []port.RawAlarmRecord, err error) {
	err = p.StreamAlarms(context.Background(), r, format, func(row port.RawAlarmRecord) error { records = append(records, row); return nil })
	return
}

func (p *DefaultAlarmParser) StreamAlarms(ctx context.Context, r io.Reader, format string, yield func(port.RawAlarmRecord) error) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panico recuperado durante leitura de alarmes: %v", rec)
		}
	}()
	var cols alarmColumnIndices
	count := 0
	err = walkRows(ctx, r, format, func(row []string) error {
		count++
		if count == 1 {
			var err error
			cols, err = extractAlarmIndices(row)
			return err
		}
		return yield(parseAlarmRow(row, cols))
	})
	if err != nil {
		return err
	}
	if count < 2 {
		return errors.New("arquivo vazio ou sem linhas de dados")
	}
	return nil
}

func extractAlarmIndices(headers []string) (alarmColumnIndices, error) {
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
		return alarmColumnIndices{}, errors.New("colunas obrigatorias ausentes no arquivo de alarmes")
	}

	return alarmColumnIndices{
		tagIdx:     tagIdx,
		stationIdx: stationIdx,
		doorIdx:    doorIdx,
		alarmIdx:   alarmIdx,
		durIdx:     durIdx,
		hasDur:     hasDur,
		normIdx:    normIdx,
		hasNorm:    hasNorm,
		platIdx:    platIdx,
		hasPlat:    hasPlat,
		timeIdx:    timeIdx,
	}, nil
}

type alarmColumnIndices struct {
	tagIdx     int
	stationIdx int
	doorIdx    int
	alarmIdx   int
	durIdx     int
	hasDur     bool
	normIdx    int
	hasNorm    bool
	platIdx    int
	hasPlat    bool
	timeIdx    int
}

func parseAlarmRow(row []string, cols alarmColumnIndices) port.RawAlarmRecord {
	doorRaw := safeGetCell(row, cols.doorIdx)
	doorNum, _ := strconv.Atoi(strings.TrimSpace(strings.Split(doorRaw, ".")[0]))

	durSec := 0
	if cols.hasDur {
		durRaw := safeGetCell(row, cols.durIdx)
		durSec, _ = strconv.Atoi(strings.TrimSpace(strings.Split(durRaw, ".")[0]))
	}

	tag := safeGetCell(row, cols.tagIdx)
	platform := extractPlatform(row, tag, cols.platIdx, cols.hasPlat)

	normTime := ""
	if cols.hasNorm {
		normTime = safeGetCell(row, cols.normIdx)
	}

	return port.RawAlarmRecord{
		Tag:                    tag,
		StationCode:            safeGetCell(row, cols.stationIdx),
		Platform:               platform,
		DoorNumber:             doorNum,
		AlarmType:              safeGetCell(row, cols.alarmIdx),
		DurationSeconds:        durSec,
		EventTimestamp:         safeGetCell(row, cols.timeIdx),
		NormalizationTimestamp: normTime,
	}
}

func extractPlatform(row []string, tag string, platIdx int, hasPlat bool) string {
	if hasPlat {
		if plat := safeGetCell(row, platIdx); plat != "" {
			return plat
		}
	}
	if strings.Contains(tag, "-P1-") || strings.Contains(tag, "-P1") {
		return "P1"
	}
	if strings.Contains(tag, "-P2-") || strings.Contains(tag, "-P2") {
		return "P2"
	}
	return ""
}
