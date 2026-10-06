package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

// Metrics here only satisfy the orchestration contract; they are not model evidence.
type rfClassificationConfig struct {
	port.MLOpsRepository
	threshold float64
}

func (r *rfClassificationConfig) GetConfig(context.Context) (*domain.MLConfig, error) {
	return &domain.MLConfig{Threshold: r.threshold, DefaultHorizonDays: 7}, nil
}

func (r *rfClassificationConfig) GetThresholdMetric(_ context.Context, _ string, threshold float64) (*domain.ThresholdMetric, error) {
	return &domain.ThresholdMetric{Threshold: threshold}, nil
}

func TestRF09_ClassificationBoundariesAndAssociation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		threshold  float64
		configured bool
		values     []float64
		atRisk     []bool
	}{
		{"fallback_050", 0.5, false, []float64{0, 0.499999, 0.5, 0.500001, 1}, []bool{false, false, true, true, true}},
		{"configured_027", 0.27, true, []float64{0, 0.269999, 0.27, 0.270001, 1}, []bool{false, false, true, true, true}},
		{"configured_075", 0.75, true, []float64{0, 0.749999, 0.75, 0.750001, 1}, []bool{false, false, true, true, true}},
		{"all_stable", 0.27, true, []float64{0, 0, 0, 0, 0}, []bool{false, false, false, false, false}},
		{"all_at_risk", 0.27, true, []float64{1, 1, 1, 1, 1}, []bool{true, true, true, true, true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newImportFixture(t)
			f.store.doors = []domain.Door{
				{ID: 1, StationCode: "FAL", Platform: "P1", DoorNumber: 1, Status: "active"},
				{ID: 2, StationCode: "FAL", Platform: "P2", DoorNumber: 1, Status: "active"},
				{ID: 3, StationCode: "PIN", Platform: "P1", DoorNumber: 1, Status: "active"},
				{ID: 4, StationCode: "FAL", Platform: "P1", DoorNumber: 2, Status: "active"},
				{ID: 5, StationCode: "PIN", Platform: "P2", DoorNumber: 1, Status: "active"},
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request mlclient.BatchPredictionRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				if len(request.Doors) != len(tc.values) {
					t.Error("unexpected number of doors")
					w.WriteHeader(400)
					return
				}
				items := make([]map[string]any, len(request.Doors))
				for i, door := range request.Doors {
					// Contradictory will_fail must be ignored: backend owns classification.
					items[i] = map[string]any{"station_code": door.StationCode, "platform": door.Platform, "door_number": door.DoorNumber, "probability": tc.values[i], "will_fail": !tc.atRisk[i]}
				}
				json.NewEncoder(w).Encode(map[string]any{"model_version": "rf09-v1", "predictions": items})
			}))
			t.Cleanup(server.Close)
			repo := &rfPredictionRepo{&intPredictionRepo{store: f.store}}
			uow := &intUnitOfWork{store: &intUOWStore{predRepo: repo, insRepo: &intInspectionRepo{store: f.store}}}
			uc := usecase.NewDashboardUseCase(&intDashboardRepo{store: f.store}, repo, mlclient.NewClient(server.URL), nil, uow)
			if tc.configured {
				uc.SetMLOpsRepository(&rfClassificationConfig{threshold: tc.threshold})
			}
			h := httpAdapter.NewDashboardHandler(uc)
			f.router.POST("/predictions/trigger", h.TriggerPredictions)
			f.router.GET("/predictions/latest", h.GetLatestPredictions)
			response := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(`{}`))
			var run httpAdapter.PredictionRunResponse
			if err := json.Unmarshal(response.Body.Bytes(), &run); err != nil {
				t.Fatal(err)
			}
			if response.Code != 201 || run.ThresholdUsed == nil || *run.ThresholdUsed != tc.threshold || run.ModelVersion != "rf09-v1" || run.TotalDoorsEvaluated != 5 || len(run.Predictions) != 5 || len(f.store.predictions) != 5 {
				t.Fatalf("unexpected run: %d %s", response.Code, response.Body.String())
			}
			wantRisk := 0
			ids := make(map[string]bool)
			for i, item := range run.Predictions {
				door := f.store.doors[i]
				if item.ID == "" || ids[item.ID] || item.RunID != run.ID || item.StationCode != door.StationCode || item.Platform != door.Platform || item.DoorNumber != door.DoorNumber || item.Probability == nil || *item.Probability != tc.values[i] || item.WillFail != tc.atRisk[i] {
					t.Fatalf("classification/association for door %d: %+v", i, item)
				}
				ids[item.ID] = true
				stored := f.store.predictions[i]
				if stored.ID != item.ID || stored.WillFail != item.WillFail || stored.Probability == nil || *stored.Probability != tc.values[i] {
					t.Fatalf("persistence differs: %+v", stored)
				}
				linked := 0
				for _, inspection := range f.store.inspections {
					if inspection.PredictionID != nil && *inspection.PredictionID == item.ID {
						linked++
						if inspection.StationCode != door.StationCode || inspection.Platform != door.Platform || inspection.DoorNumber != door.DoorNumber {
							t.Fatal("inspection associated with wrong door")
						}
					}
				}
				wantLinked := 0
				if tc.atRisk[i] {
					wantRisk++
					wantLinked = 1
				}
				if linked != wantLinked {
					t.Fatalf("inspection count for prediction %s: %d", item.ID, linked)
				}
			}
			if run.TotalDoorsAtRisk != wantRisk || len(f.store.inspections) != wantRisk {
				t.Fatal("risk totals differ from classifications")
			}
			latest := maintenanceRequest(t, f, http.MethodGet, "/predictions/latest", nil)
			var queried httpAdapter.PredictionRunResponse
			if err := json.Unmarshal(latest.Body.Bytes(), &queried); err != nil {
				t.Fatal(err)
			}
			if latest.Code != 200 || !reflect.DeepEqual(run, queried) {
				t.Fatalf("query changed classification: %s", latest.Body.String())
			}
		})
	}
}

func TestRF09_InvalidMLResponsePreservesLatestClassification(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alter func([]map[string]any)
	}{
		{"missing_probability", func(p []map[string]any) { delete(p[0], "probability") }},
		{"null_probability", func(p []map[string]any) { p[0]["probability"] = nil }},
		{"negative_probability", func(p []map[string]any) { p[0]["probability"] = -0.01 }},
		{"probability_above_one", func(p []map[string]any) { p[0]["probability"] = 1.01 }},
		{"text_probability", func(p []map[string]any) { p[0]["probability"] = "0.8" }},
		{"wrong_station", func(p []map[string]any) { p[0]["station_code"] = "PIN" }},
		{"wrong_platform", func(p []map[string]any) { p[0]["platform"] = "P2" }},
		{"wrong_door", func(p []map[string]any) { p[0]["door_number"] = 99 }},
		{"duplicate_door", func(p []map[string]any) { p[1] = p[0] }},
		{"reordered_doors", func(p []map[string]any) { p[0], p[1] = p[1], p[0] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := predictionFixture(t, func(w http.ResponseWriter, _ *http.Request) {
				if calls.Add(1) == 1 {
					w.Write([]byte(rfPredictionResponse))
					return
				}
				var body struct {
					ModelVersion string           `json:"model_version"`
					Predictions  []map[string]any `json:"predictions"`
				}
				if err := json.Unmarshal([]byte(rfPredictionResponse), &body); err != nil {
					t.Error(err)
				}
				tc.alter(body.Predictions)
				json.NewEncoder(w).Encode(body)
			})
			created := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(`{}`))
			if created.Code != 201 {
				t.Fatalf("could not seed latest run: %s", created.Body.String())
			}
			var before httpAdapter.PredictionRunResponse
			if err := json.Unmarshal(created.Body.Bytes(), &before); err != nil {
				t.Fatal(err)
			}
			response := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(`{}`))
			var problem httpAdapter.ProblemDetails
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if response.Code != 503 || problem.Code != "ERR_PREDICTION_UNAVAILABLE" || len(f.store.runs) != 1 || len(f.store.predictions) != 2 || len(f.store.inspections) != 1 {
				t.Fatalf("invalid ML response replaced classification: %d %s", response.Code, response.Body.String())
			}
			latest := maintenanceRequest(t, f, http.MethodGet, "/predictions/latest", nil)
			var after httpAdapter.PredictionRunResponse
			if err := json.Unmarshal(latest.Body.Bytes(), &after); err != nil {
				t.Fatal(err)
			}
			if latest.Code != 200 || !reflect.DeepEqual(before, after) {
				t.Fatalf("last valid classification lost: %s", latest.Body.String())
			}
		})
	}
}
