package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidRecordFormat    = errors.New("formato de registro invalido")
	ErrDuplicateUpload        = errors.New("arquivo duplicado detectado")
	ErrAlarmTagRequired       = errors.New("a tag do alarme e obrigatoria")
	ErrAlarmStationRequired   = errors.New("o codigo da estacao e obrigatorio")
	ErrInvalidAlarmPlatform   = errors.New("a plataforma deve ser P1 ou P2")
	ErrInvalidAlarmDoorNumber = errors.New("o numero da porta deve estar entre 1 e 24")
	ErrAlarmTypeRequired      = errors.New("o tipo do alarme e obrigatorio")
	ErrAlarmTimestampRequired = errors.New("o horario do evento e obrigatorio")
)

type Alarm struct {
	ID                     string
	UploadID               string
	Tag                    string
	StationCode            string
	Platform               string
	DoorNumber             int
	AlarmType              string
	DurationSeconds        int
	EventTimestamp         string
	NormalizationTimestamp string
	Source                 string
	CreatedAt              time.Time
}

func NewAlarm(
	id string,
	uploadID string,
	tag string,
	stationCode string,
	platform string,
	doorNumber int,
	alarmType string,
	durationSeconds int,
	eventTimestamp string,
	normalizationTimestamp string,
	source string,
) (*Alarm, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, ErrAlarmTagRequired
	}

	stationCode = strings.TrimSpace(strings.ToUpper(stationCode))
	if stationCode == "" {
		return nil, ErrAlarmStationRequired
	}

	platform = strings.TrimSpace(strings.ToUpper(platform))
	if platform != "P1" && platform != "P2" {
		return nil, ErrInvalidAlarmPlatform
	}

	if doorNumber < 1 || doorNumber > 24 {
		return nil, ErrInvalidAlarmDoorNumber
	}

	alarmType = strings.TrimSpace(alarmType)
	if alarmType == "" {
		return nil, ErrAlarmTypeRequired
	}

	eventTimestamp = strings.TrimSpace(eventTimestamp)
	if eventTimestamp == "" {
		return nil, ErrAlarmTimestampRequired
	}

	if source == "" {
		source = "telemetry"
	}

	return &Alarm{
		ID:                     id,
		UploadID:               uploadID,
		Tag:                    tag,
		StationCode:            stationCode,
		Platform:               platform,
		DoorNumber:             doorNumber,
		AlarmType:              alarmType,
		DurationSeconds:        durationSeconds,
		EventTimestamp:         eventTimestamp,
		NormalizationTimestamp: strings.TrimSpace(normalizationTimestamp),
		Source:                 source,
		CreatedAt:              time.Now(),
	}, nil
}
