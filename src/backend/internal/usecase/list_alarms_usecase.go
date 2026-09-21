package usecase

import (
	"context"
	"time"

	"github.com/g04/backend/internal/usecase/port"
)

type ListAlarmsInputDTO struct {
	StationCode string
	Platform    string
	DoorNumber  int
	AlarmType   string
	Tag         string
	Limit       int
	Offset      int
}

type AlarmOutputDTO struct {
	ID                     string    `json:"id"`
	UploadID               string    `json:"upload_id,omitempty"`
	Tag                    string    `json:"tag"`
	StationCode            string    `json:"station_code"`
	Platform               string    `json:"platform,omitempty"`
	DoorNumber             int       `json:"door_number,omitempty"`
	AlarmType              string    `json:"alarm_type"`
	DurationSeconds        int       `json:"duration_seconds"`
	EventTimestamp         string    `json:"event_timestamp"`
	NormalizationTimestamp string    `json:"normalization_timestamp,omitempty"`
	Source                 string    `json:"source"`
	CreatedAt              time.Time `json:"created_at"`
}

type ListAlarmsUseCase struct {
	repo port.AlarmRepository
}

func NewListAlarmsUseCase(repo port.AlarmRepository) *ListAlarmsUseCase {
	return &ListAlarmsUseCase{repo: repo}
}

func (uc *ListAlarmsUseCase) Execute(ctx context.Context, input ListAlarmsInputDTO) ([]*AlarmOutputDTO, error) {
	alarms, err := uc.repo.List(ctx, port.AlarmFilter{
		StationCode: input.StationCode,
		Platform:    input.Platform,
		DoorNumber:  input.DoorNumber,
		AlarmType:   input.AlarmType,
		Tag:         input.Tag,
		Limit:       input.Limit,
		Offset:      input.Offset,
	})
	if err != nil {
		return nil, err
	}

	dtos := make([]*AlarmOutputDTO, 0, len(alarms))
	for _, a := range alarms {
		dtos = append(dtos, &AlarmOutputDTO{
			ID:                     a.ID,
			UploadID:               a.UploadID,
			Tag:                    a.Tag,
			StationCode:            a.StationCode,
			Platform:               a.Platform,
			DoorNumber:             a.DoorNumber,
			AlarmType:              a.AlarmType,
			DurationSeconds:        a.DurationSeconds,
			EventTimestamp:         a.EventTimestamp,
			NormalizationTimestamp: a.NormalizationTimestamp,
			Source:                 a.Source,
			CreatedAt:              a.CreatedAt,
		})
	}

	return dtos, nil
}
