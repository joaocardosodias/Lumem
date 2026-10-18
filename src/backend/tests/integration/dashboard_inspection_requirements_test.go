package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

type rfSummaryRepo struct {
	port.DashboardRepository
	value *domain.DashboardSummary
	err   error
}

func (r *rfSummaryRepo) GetSummary(context.Context) (*domain.DashboardSummary, error) {
	return r.value, r.err
}

func TestRF10_SummaryHTTPContract(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  *domain.DashboardSummary
		err    error
		status int
	}{
		{"populated", &domain.DashboardSummary{TotalDoors: 10, DoorsAtRisk: 4, PendingInspections: 3, CompletedInspections: 2, CriticalStations: []domain.StationRiskSummary{{StationCode: "FAL", DoorsAtRisk: 3}, {StationCode: "PIN", DoorsAtRisk: 1}}}, nil, 200},
		{"empty", &domain.DashboardSummary{}, nil, 200},
		{"repository_error", nil, errors.New("summary unavailable"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newImportFixture(t)
			uc := usecase.NewDashboardUseCase(&rfSummaryRepo{value: tc.value, err: tc.err}, nil, nil, nil, nil)
			h := httpAdapter.NewDashboardHandler(uc)
			f.router.GET("/dashboard/summary", h.GetSummary)
			r := maintenanceRequest(t, f, http.MethodGet, "/dashboard/summary", nil)
			if r.Code != tc.status {
				t.Fatalf("unexpected response: %d %s", r.Code, r.Body.String())
			}
			if tc.err != nil {
				var problem httpAdapter.ProblemDetails
				json.Unmarshal(r.Body.Bytes(), &problem)
				if problem.Code != "ERR_INTERNAL_SERVER" {
					t.Fatalf("unexpected problem: %+v", problem)
				}
				return
			}
			var body struct {
				Total     int                               `json:"total_doors"`
				Risk      int                               `json:"doors_at_risk"`
				Pending   int                               `json:"pending_inspections"`
				Completed int                               `json:"completed_inspections"`
				Stations  []httpAdapter.StationRiskResponse `json:"critical_stations"`
			}
			if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Total != tc.value.TotalDoors || body.Risk != tc.value.DoorsAtRisk || body.Pending != tc.value.PendingInspections || body.Completed != tc.value.CompletedInspections || body.Stations == nil || len(body.Stations) != len(tc.value.CriticalStations) {
				t.Fatalf("summary lost data: %s", r.Body.String())
			}
			for i, s := range body.Stations {
				if s.StationCode != tc.value.CriticalStations[i].StationCode || s.DoorsAtRisk != tc.value.CriticalStations[i].DoorsAtRisk {
					t.Fatal("ranking changed in HTTP mapping")
				}
			}
		})
	}
}

func TestRF11_AutomaticInspectionLifecycle(t *testing.T) {
	f := predictionFixture(t, func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(rfPredictionResponse)) })
	h := httpAdapter.NewInspectionHandler(usecase.NewInspectionUseCase(&intInspectionRepo{store: f.store}))
	f.router.GET("/inspections", h.List)
	f.router.GET("/inspections/:id", h.GetByID)
	f.router.PATCH("/inspections/:id/status", h.UpdateStatus)
	created := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(`{}`))
	if created.Code != 201 || len(f.store.inspections) != 1 {
		t.Fatalf("automatic creation: %s", created.Body.String())
	}
	var run httpAdapter.PredictionRunResponse
	if err := json.Unmarshal(created.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	var inspection *domain.Inspection
	for _, v := range f.store.inspections {
		inspection = v
	}
	if inspection.Status != "pending" || inspection.PredictionID == nil || *inspection.PredictionID != run.Predictions[0].ID || inspection.StationCode != "FAL" || inspection.Platform != "P1" || inspection.DoorNumber != 4 {
		t.Fatalf("invalid automatic inspection: %+v", inspection)
	}
	for _, status := range []string{"pending", "in_progress", "completed", "pending", "canceled"} {
		r := maintenanceRequest(t, f, http.MethodPatch, "/inspections/"+inspection.ID+"/status", []byte(`{"status":"`+status+`","notes":"Verificacao de campo"}`))
		var updated httpAdapter.InspectionResponse
		if err := json.Unmarshal(r.Body.Bytes(), &updated); err != nil {
			t.Fatal(err)
		}
		if r.Code != 200 || updated.Status != status || updated.PredictionID == nil || *updated.PredictionID != run.Predictions[0].ID || (updated.CompletedAt != nil) != (status == "completed") || updated.Notes != "Verificacao de campo" {
			t.Fatalf("invalid transition response: %s", r.Body.String())
		}
		get := maintenanceRequest(t, f, http.MethodGet, "/inspections/"+inspection.ID, nil)
		var queried httpAdapter.InspectionResponse
		if err := json.Unmarshal(get.Body.Bytes(), &queried); err != nil {
			t.Fatal(err)
		}
		if get.Code != 200 || !reflect.DeepEqual(updated, queried) {
			t.Fatal("query differs after status update")
		}
	}
	for _, tc := range []struct{ name, body string }{{"invalid_status", `{"status":"urgent"}`}, {"reopen_canceled", `{"status":"pending"}`}, {"malformed", `{`}} {
		t.Run(tc.name, func(t *testing.T) {
			before := *inspection
			r := maintenanceRequest(t, f, http.MethodPatch, "/inspections/"+inspection.ID+"/status", []byte(tc.body))
			if r.Code != 400 || !reflect.DeepEqual(before, *inspection) {
				t.Fatalf("invalid update changed inspection: %s", r.Body.String())
			}
		})
	}
	missing := maintenanceRequest(t, f, http.MethodGet, "/inspections/missing", nil)
	if missing.Code != 404 {
		t.Fatalf("missing inspection: %d", missing.Code)
	}
	list := maintenanceRequest(t, f, http.MethodGet, "/inspections?status=pending", nil)
	var empty struct {
		Data  []httpAdapter.InspectionResponse `json:"data"`
		Total int                              `json:"total"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if list.Code != 200 || empty.Total != 0 || empty.Data == nil || len(empty.Data) != 0 {
		t.Fatalf("canceled still pending: %s", list.Body.String())
	}
}
