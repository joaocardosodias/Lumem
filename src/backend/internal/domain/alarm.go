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

type CreateAlarmParams struct {
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
}

func NewAlarm(p CreateAlarmParams) (*Alarm, error) {
	tag := strings.TrimSpace(p.Tag)
	if tag == "" {
		return nil, ErrAlarmTagRequired
	}

	stationCode := strings.TrimSpace(strings.ToUpper(p.StationCode))
	if stationCode == "" {
		return nil, ErrAlarmStationRequired
	}

	platform := strings.TrimSpace(strings.ToUpper(p.Platform))
	if platform != "P1" && platform != "P2" {
		return nil, ErrInvalidAlarmPlatform
	}

	if p.DoorNumber < 1 || p.DoorNumber > 24 {
		return nil, ErrInvalidAlarmDoorNumber
	}

	alarmType := strings.TrimSpace(p.AlarmType)
	if alarmType == "" {
		return nil, ErrAlarmTypeRequired
	}

	eventTimestamp := strings.TrimSpace(p.EventTimestamp)
	if eventTimestamp == "" {
		return nil, ErrAlarmTimestampRequired
	}

	source := p.Source
	if source == "" {
		source = "telemetry"
	}

	return &Alarm{
		ID:                     p.ID,
		UploadID:               p.UploadID,
		Tag:                    tag,
		StationCode:            stationCode,
		Platform:               platform,
		DoorNumber:             p.DoorNumber,
		AlarmType:              alarmType,
		DurationSeconds:        p.DurationSeconds,
		EventTimestamp:         eventTimestamp,
		NormalizationTimestamp: strings.TrimSpace(p.NormalizationTimestamp),
		Source:                 source,
		CreatedAt:              time.Now(),
	}, nil
}
