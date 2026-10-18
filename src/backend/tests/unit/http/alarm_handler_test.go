package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/gin-gonic/gin"
)

type mockAlarmRepoForHandlerTest struct {
	alarms []*domain.Alarm
	err    error
}

func (m *mockAlarmRepoForHandlerTest) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	return nil
}

func (m *mockAlarmRepoForHandlerTest) DeleteByUploadID(ctx context.Context, uploadID string) error {
	return nil
}

func (m *mockAlarmRepoForHandlerTest) List(ctx context.Context, filter port.AlarmFilter) ([]*domain.Alarm, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.alarms, nil
}

func TestAlarmHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAlarmRepoForHandlerTest{
		alarms: []*domain.Alarm{
			{
				ID:              "al-1",
				Tag:             "SP04-6-PSD-BUT-P1-PP04",
				StationCode:     "BUT",
				Platform:        "P1",
				DoorNumber:      4,
				AlarmType:       "Obstaculo",
				DurationSeconds: 5,
				EventTimestamp:  "2026-09-02 12:00:00",
				Source:          "telemetry",
				CreatedAt:       time.Now(),
			},
		},
	}

	uc := usecase.NewListAlarmsUseCase(repo)
	handler := httpAdapter.NewAlarmHandler(uc)

	r := gin.New()
	r.GET("/api/v1/alarms", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alarms?station_code=BUT&platform=P1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200, obteve %d", w.Code)
	}
}
