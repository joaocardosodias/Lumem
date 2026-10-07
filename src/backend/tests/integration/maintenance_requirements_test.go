package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

func maintenanceRequest(t *testing.T, f *importFixture, method, path string, payload []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.token)
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, req)
	return response
}

type rfAlarmFilterSpy struct {
	port.AlarmRepository
	received port.AlarmFilter
}

func (s *rfAlarmFilterSpy) List(_ context.Context, filter port.AlarmFilter) ([]*domain.Alarm, error) {
	s.received = filter
	return []*domain.Alarm{}, nil
}

func TestRF06_AlarmFilterParameters(t *testing.T) {
	f := newImportFixture(t)
	repo := &rfAlarmFilterSpy{}
	handler := httpAdapter.NewAlarmHandler(usecase.NewListAlarmsUseCase(repo))
	f.router.GET("/alarms", handler.List)
	response := maintenanceRequest(t, f, http.MethodGet, "/alarms?station_code=FAL&platform=P2&door_number=8&alarm_type=OBS&tag=TAG-8&limit=10&offset=20", nil)
	want := port.AlarmFilter{StationCode: "FAL", Platform: "P2", DoorNumber: 8, AlarmType: "OBS", Tag: "TAG-8", Limit: 10, Offset: 20}
	if response.Code != 200 || repo.received != want {
		t.Fatalf("filter not forwarded: %+v; response=%s", repo.received, response.Body.String())
	}
}

func TestRF06_InspectionStatusQuery(t *testing.T) {
	for _, status := range []string{"pending", "in_progress", "completed", ""} {
		t.Run("status="+status, func(t *testing.T) {
			f := newImportFixture(t)
			for _, value := range []string{"pending", "in_progress", "completed"} {
				f.store.inspections[value] = &domain.Inspection{ID: value, Status: value, StationCode: "FAL", Platform: "P1", DoorNumber: 4}
			}
			handler := httpAdapter.NewInspectionHandler(usecase.NewInspectionUseCase(&intInspectionRepo{store: f.store}))
			f.router.GET("/inspections", handler.List)
			response := maintenanceRequest(t, f, http.MethodGet, "/inspections?status="+status, nil)
			var result struct {
				Data []httpAdapter.InspectionResponse `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			wantCount := 1
			if status == "" {
				wantCount = 3
			}
			if response.Code != 200 || len(result.Data) != wantCount {
				t.Fatalf("status query: %s", response.Body.String())
			}
			for _, item := range result.Data {
				if status != "" && item.Status != status {
					t.Fatalf("unexpected status: %+v", item)
				}
			}
		})
	}
}

func newMaintenanceFixture(t *testing.T) *importFixture {
	f := newImportFixture(t)
	handler := httpAdapter.NewMaintenanceHandler(usecase.NewCreateMaintenanceUseCase(&intMaintenanceRepo{store: f.store}))
	f.router.POST("/maintenance", handler.Create)
	f.router.GET("/maintenance", handler.List)
	return f
}

func TestRF05_CreateAndQueryAllFields(t *testing.T) {
	f := newMaintenanceFixture(t)
	request := []byte(`{"sap_order_number":"12345","order_type":"zm03","station_code":" fal ","installation_location":"SP04-6-PSD-FAL-P1-PP04","start_timestamp":"2026-10-01 10:00:00","end_timestamp":"2026-10-01 11:30:00","duration_hours":1.5,"description":"Troca do sensor","symptom":"Falha de fechamento","cause":"Desgaste","action_taken":"Sensor substituido"}`)
	created := maintenanceRequest(t, f, http.MethodPost, "/maintenance", request)
	if created.Code != 201 {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var result httpAdapter.MaintenanceResponse
	if err := json.Unmarshal(created.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ID == "" || result.CreatedAt.IsZero() {
		t.Fatal("missing generated metadata")
	}
	want := httpAdapter.MaintenanceResponse{
		ID: result.ID, CreatedAt: result.CreatedAt, SAPOrderNumber: "12345", OrderType: "ZM03", StationCode: "FAL",
		InstallationLocation: "SP04-6-PSD-FAL-P1-PP04", StartTimestamp: "2026-10-01 10:00:00", EndTimestamp: "2026-10-01 11:30:00",
		DurationHours: 1.5, Description: "Troca do sensor", Symptom: "Falha de fechamento", Cause: "Desgaste", ActionTaken: "Sensor substituido", Source: "manual_form",
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("created fields: got %+v, want %+v", result, want)
	}
	listed := maintenanceRequest(t, f, http.MethodGet, "/maintenance?limit=50&offset=0", nil)
	var list struct {
		Total int                               `json:"total"`
		Data  []httpAdapter.MaintenanceResponse `json:"data"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if listed.Code != 200 || list.Total != 1 || len(list.Data) != 1 || !reflect.DeepEqual(list.Data[0], want) {
		t.Fatalf("query lost fields: %s", listed.Body.String())
	}
	if len(f.store.maintenances) != 1 || f.store.maintenances[0].ID != result.ID {
		t.Fatal("record not stored")
	}
}

func TestRF05_InvalidRegistrationDoesNotPersist(t *testing.T) {
	for _, tc := range []struct{ name, payload, code string }{
		{"missing_type", `{"station_code":"FAL","start_timestamp":"2026-10-01"}`, "ERR_VALIDATION"},
		{"missing_station", `{"order_type":"ZM01","start_timestamp":"2026-10-01"}`, "ERR_VALIDATION"},
		{"missing_start", `{"order_type":"ZM01","station_code":"FAL"}`, "ERR_VALIDATION"},
		{"blank_type", `{"order_type":"  ","station_code":"FAL","start_timestamp":"2026-10-01"}`, "ERR_VALIDATION"},
		{"empty_body", `{}`, "ERR_VALIDATION"},
		{"malformed_json", `{`, "ERR_INVALID_PAYLOAD"},
		{"duration_wrong_type", `{"duration_hours":"invalid"}`, "ERR_INVALID_PAYLOAD"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMaintenanceFixture(t)
			response := maintenanceRequest(t, f, http.MethodPost, "/maintenance", []byte(tc.payload))
			var problem httpAdapter.ProblemDetails
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if response.Code != 400 || problem.Code != tc.code || len(f.store.maintenances) != 0 {
				t.Fatalf("unexpected rejection: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRF05_QueryEmptyCollection(t *testing.T) {
	f := newMaintenanceFixture(t)
	response := maintenanceRequest(t, f, http.MethodGet, "/maintenance", nil)
	var result struct {
		Total int                               `json:"total"`
		Data  []httpAdapter.MaintenanceResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || result.Total != 0 || result.Data == nil || len(result.Data) != 0 {
		t.Fatalf("empty collection: %s", response.Body.String())
	}
}
