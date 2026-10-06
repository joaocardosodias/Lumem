package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
)

// Only the PostgreSQL test below verifies the production active-door query.
type rfPredictionRepo struct{ *intPredictionRepo }

func (r *rfPredictionRepo) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	doors, err := r.intPredictionRepo.ListDoorsForInference(ctx)
	var active []domain.Door
	for _, door := range doors {
		if door.Status == "active" {
			active = append(active, door)
		}
	}
	return active, err
}

func predictionFixture(t *testing.T, serve http.HandlerFunc) *importFixture {
	t.Helper()
	f := newImportFixture(t)
	f.store.doors = []domain.Door{
		{ID: 1, StationCode: "FAL", Platform: "P1", DoorNumber: 4, Status: "active"},
		{ID: 2, StationCode: "FAL", Platform: "P1", DoorNumber: 5, Status: "inactive"},
		{ID: 3, StationCode: "PIN", Platform: "P2", DoorNumber: 8, Status: "active"},
	}
	server := httptest.NewServer(serve)
	t.Cleanup(server.Close)
	repo := &rfPredictionRepo{&intPredictionRepo{store: f.store}}
	uow := &intUnitOfWork{store: &intUOWStore{predRepo: repo, insRepo: &intInspectionRepo{store: f.store}}}
	uc := usecase.NewDashboardUseCase(&intDashboardRepo{store: f.store}, repo, mlclient.NewClient(server.URL), nil, uow)
	h := httpAdapter.NewDashboardHandler(uc)
	f.router.POST("/predictions/trigger", h.TriggerPredictions)
	f.router.GET("/predictions/latest", h.GetLatestPredictions)
	return f
}

const rfPredictionResponse = `{"model_version":"rf-test-v1","predictions":[{"station_code":"FAL","platform":"P1","door_number":4,"probability":0.8},{"station_code":"PIN","platform":"P2","door_number":8,"probability":0.2}]}`

func TestRF07_TriggerAndQueryPrediction(t *testing.T) {
	var calls atomic.Int32
	f := predictionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]json.RawMessage
		if r.Method != http.MethodPost || r.URL.Path != "/v1/predictions" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("unexpected ML request contract")
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		var doors []mlclient.PredictionRequest
		if err := json.Unmarshal(body["doors"], &doors); err != nil {
			t.Error(err)
		}
		want := []mlclient.PredictionRequest{{StationCode: "FAL", Platform: "P1", DoorNumber: 4}, {StationCode: "PIN", Platform: "P2", DoorNumber: 8}}
		if len(body) != 1 || !reflect.DeepEqual(doors, want) {
			t.Errorf("unexpected inference payload: %s", body)
		}
		w.Write([]byte(rfPredictionResponse))
	})
	response := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(`{}`))
	var run httpAdapter.PredictionRunResponse
	if err := json.Unmarshal(response.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	if response.Code != 201 || calls.Load() != 1 || run.ID == "" || run.ModelVersion != "rf-test-v1" || run.HorizonDays != 7 || run.TotalDoorsEvaluated != 2 || run.TotalDoorsAtRisk != 1 || len(run.Predictions) != 2 {
		t.Fatalf("unexpected run: %d %s", response.Code, response.Body.String())
	}
	for i, item := range run.Predictions {
		want := f.store.doors[i*2]
		if item.ID == "" || item.RunID != run.ID || item.StationCode != want.StationCode || item.Platform != want.Platform || item.DoorNumber != want.DoorNumber || item.Probability == nil || item.WillFail != (i == 0) {
			t.Fatalf("prediction association lost: %+v", item)
		}
	}
	if len(f.store.runs) != 1 || len(f.store.inspections) != 1 {
		t.Fatal("run or automatic inspection not saved")
	}
	latest := maintenanceRequest(t, f, http.MethodGet, "/predictions/latest", nil)
	var queried httpAdapter.PredictionRunResponse
	if err := json.Unmarshal(latest.Body.Bytes(), &queried); err != nil {
		t.Fatal(err)
	}
	if latest.Code != 200 || !reflect.DeepEqual(run, queried) {
		t.Fatalf("latest run differs: %s", latest.Body.String())
	}
}

func TestRF07_FailuresDoNotPersist(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		mlStatus   int
	}{
		{"unavailable", `{}`, 503},
		{"malformed", `{`, 200},
		{"missing_version", `{"predictions":[]}`, 200},
		{"missing_prediction", `{"model_version":"v1","predictions":[]}`, 200},
		{"wrong_door", `{"model_version":"v1","predictions":[{"station_code":"FAL","platform":"P1","door_number":99,"probability":0.8},{"station_code":"PIN","platform":"P2","door_number":8,"probability":0.2}]}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := predictionFixture(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.mlStatus); w.Write([]byte(tc.body)) })
			response := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(`{}`))
			if response.Code != 503 || !bytes.Contains(response.Body.Bytes(), []byte("ERR_PREDICTION_UNAVAILABLE")) || len(f.store.runs) != 0 || len(f.store.predictions) != 0 || len(f.store.inspections) != 0 {
				t.Fatalf("failure persisted results: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRF07_NoActiveDoorsOrAuthentication(t *testing.T) {
	for _, authenticated := range []bool{true, false} {
		name := "no_active_doors"
		if !authenticated {
			name = "unauthenticated"
		}
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			f := predictionFixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1) })
			if authenticated {
				for i := range f.store.doors {
					f.store.doors[i].Status = "inactive"
				}
			} else {
				f.token = ""
			}
			response := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(`{}`))
			want := 400
			if !authenticated {
				want = 401
			}
			if response.Code != want || calls.Load() != 0 || len(f.store.runs) != 0 {
				t.Fatalf("unexpected side effect: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRF08_DefaultAndCurrentFixedHorizon(t *testing.T) {
	for _, tc := range []struct{ name, body string }{{"omitted", `{}`}, {"empty_body", ``}, {"null", `{"horizon_days":null}`}, {"seven", `{"horizon_days":7}`}, {"fourteen_current_divergence", `{"horizon_days":14}`}} {
		t.Run(tc.name, func(t *testing.T) {
			f := predictionFixture(t, func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(rfPredictionResponse)) })
			response := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(tc.body))
			if response.Code != 201 || len(f.store.runs) != 1 || f.store.runs[0].HorizonDays != 7 {
				t.Fatalf("unexpected horizon: %d %s", response.Code, response.Body.String())
			}
			if tc.name == "fourteen_current_divergence" {
				t.Log("KNOWN RF08 GAP: requested 14 days, persisted 7; characterization, not requirement acceptance")
			}
		})
	}
}

func TestRF08_InvalidInputDoesNotRunInference(t *testing.T) {
	for _, tc := range []struct{ name, body, code string }{
		{"malformed", `{`, "ERR_INVALID_PAYLOAD"},
		{"string", `{"horizon_days":"seven"}`, "ERR_INVALID_PAYLOAD"},
		{"fractional", `{"horizon_days":1.5}`, "ERR_INVALID_PAYLOAD"},
		{"zero", `{"horizon_days":0}`, "ERR_VALIDATION"},
		{"negative", `{"horizon_days":-1}`, "ERR_VALIDATION"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := predictionFixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.Write([]byte(rfPredictionResponse)) })
			response := maintenanceRequest(t, f, http.MethodPost, "/predictions/trigger", []byte(tc.body))
			if response.Code != 400 || !bytes.Contains(response.Body.Bytes(), []byte(tc.code)) || calls.Load() != 0 || len(f.store.runs) != 0 || len(f.store.inspections) != 0 {
				t.Fatalf("invalid input executed inference: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRF07_PostgresActiveDoorSelection(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL absent; production SQL selection not verified")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	// Temporary tables shadow the application tables for this transaction only.
	for _, query := range []string{
		`CREATE TEMP TABLE stations (id integer, code text) ON COMMIT DROP`,
		`CREATE TEMP TABLE doors (id integer, station_id integer, platform text, door_number integer, sap_tag text, status text, created_at timestamp) ON COMMIT DROP`,
		`INSERT INTO stations VALUES (1, 'FAL')`,
		`INSERT INTO doors VALUES (1,1,'P1',1,'TAG1','active',now()),(2,1,'P1',2,'TAG2','inactive',now()),(3,1,'P2',3,'TAG3','active',now())`,
	} {
		if _, err := tx.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	doors, err := repository.NewPostgresPredictionRepository(tx).ListDoorsForInference(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(doors) != 2 || doors[0].ID != 1 || doors[1].ID != 3 {
		t.Fatalf("unexpected active selection: %+v", doors)
	}
}
