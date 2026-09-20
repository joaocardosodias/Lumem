package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
)

func TestListAlarmsUseCase_Success(t *testing.T) {
	repo := &mockAlarmRepo{
		alarms: []*domain.Alarm{
			{
				ID:             "a1",
				Tag:            "SP04-6-PSD-FAL-P1-PP04",
				StationCode:    "FAL",
				Platform:       "P1",
				DoorNumber:     4,
				AlarmType:      "Obstaculo detectado",
				EventTimestamp: "2026-09-02 10:00:00",
				Source:         "telemetry",
				CreatedAt:      time.Now(),
			},
			{
				ID:             "a2",
				Tag:            "SP04-6-PSD-BUT-P2-PP12",
				StationCode:    "BUT",
				Platform:       "P2",
				DoorNumber:     12,
				AlarmType:      "Tempo de ciclo excedido",
				EventTimestamp: "2026-09-02 11:00:00",
				Source:         "telemetry",
				CreatedAt:      time.Now(),
			},
		},
	}

	uc := usecase.NewListAlarmsUseCase(repo)
	dtos, err := uc.Execute(context.Background(), usecase.ListAlarmsInputDTO{Limit: 10})
	if err != nil {
		t.Fatalf("esperava sucesso ao listar alarmes, obteve: %v", err)
	}

	if len(dtos) != 2 {
		t.Fatalf("esperava 2 alarmes, obteve: %d", len(dtos))
	}

	if dtos[0].Tag != "SP04-6-PSD-FAL-P1-PP04" {
		t.Errorf("tag inesperada no primeiro item: %s", dtos[0].Tag)
	}
}
