package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type operationalHTTPFixture struct {
	*importFixture
	server     *httptest.Server
	client     *http.Client
	mode       atomic.Int32
	modelCalls atomic.Int32
}

func newOperationalHTTPFixture(t *testing.T) *operationalHTTPFixture {
	t.Helper()
	f := &operationalHTTPFixture{importFixture: newImportFixture(t)}
	user := *f.store.users["operador@motiva.com.br"]
	user.ID = "engineer-test"
	user.Email = "engineer@example.test"
	user.Role = "enginner"
	f.store.users[user.Email] = &user
	f.store.doors = []domain.Door{{ID: 1, StationCode: "FAL", Platform: "P1", DoorNumber: 4, Status: "active"}, {ID: 2, StationCode: "PIN", Platform: "P2", DoorNumber: 8, Status: "active"}}
	mlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.modelCalls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/v1/predictions" {
			http.Error(w, "wrong contract", 400)
			return
		}
		var input mlclient.BatchPredictionRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		want := []mlclient.PredictionRequest{{StationCode: "FAL", Platform: "P1", DoorNumber: 4}, {StationCode: "PIN", Platform: "P2", DoorNumber: 8}}
		if !reflect.DeepEqual(input.Doors, want) {
			t.Error("wrong inference identities")
			w.WriteHeader(400)
			return
		}
		switch f.mode.Load() {
		case 1:
			w.WriteHeader(503)
		case 2:
			w.Write([]byte(`{"model_version":"broken","predictions":[]}`))
		default:
			w.Write([]byte(rfPredictionResponse))
		}
	}))
	t.Cleanup(mlServer.Close)
	pred := &rfPredictionRepo{&intPredictionRepo{store: f.store}}
	ins := &intInspectionRepo{store: f.store}
	uow := &intUnitOfWork{store: &intUOWStore{predRepo: pred, insRepo: ins}}
	dash := httpAdapter.NewDashboardHandler(usecase.NewDashboardUseCase(&intDashboardRepo{store: f.store}, pred, mlclient.NewClient(mlServer.URL), nil, uow))
	inspection := httpAdapter.NewInspectionHandler(usecase.NewInspectionUseCase(ins))
	upload := httpAdapter.NewUploadHandler(f.alarm, f.maintenance)
	maintenance := httpAdapter.NewMaintenanceHandler(usecase.NewCreateMaintenanceUseCase(&intMaintenanceRepo{store: f.store}))
	login := httpAdapter.NewAuthHandler(usecase.NewLoginUseCase(&intUserRepo{store: f.store}, security.NewBcryptPasswordHasher(bcrypt.MinCost), security.NewJWTTokenGenerator("workflow-test-key"), time.Hour))
	router := gin.New()
	router.Use(httpAdapter.CorrelationIDMiddleware())
	router.POST("/api/v1/auth/login", login.Login)
	protected := router.Group("/api/v1", httpAdapter.AuthMiddleware("workflow-test-key"))
	protected.POST("/uploads/alarms", upload.UploadAlarms)
	protected.POST("/uploads/maintenance", upload.UploadMaintenance)
	protected.GET("/uploads/:id", upload.GetUploadStatus)
	protected.POST("/maintenance", maintenance.Create)
	protected.GET("/maintenance", maintenance.List)
	protected.POST("/predictions/trigger", httpAdapter.RequireRole("enginner"), dash.TriggerPredictions)
	protected.GET("/predictions/latest", dash.GetLatestPredictions)
	protected.GET("/dashboard/summary", dash.GetSummary)
	protected.GET("/doors/:id/timeline", dash.GetDoorTimeline)
	protected.GET("/inspections", inspection.List)
	protected.GET("/inspections/:id", inspection.GetByID)
	protected.PATCH("/inspections/:id/status", inspection.UpdateStatus)
	f.server = httptest.NewServer(router)
	t.Cleanup(f.server.Close)
	f.client = f.server.Client()
	f.client.Timeout = 5 * time.Second
	return f
}

func (f *operationalHTTPFixture) request(t *testing.T, method, path, token, contentType string, body []byte, want int) []byte {
	t.Helper()
	req, err := http.NewRequest(method, f.server.URL+"/api/v1"+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Request-ID", "workflow-test-request")
	response, err := f.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != want {
		t.Fatalf("%s %s: status %d, expected %d; %s", method, path, response.StatusCode, want, data)
	}
	if response.Header.Get("X-Request-ID") != "workflow-test-request" {
		t.Fatal("request ID not propagated")
	}
	return data
}
func decodeWorkflow[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func (f *operationalHTTPFixture) login(t *testing.T, email string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": "operador123"})
	result := decodeWorkflow[struct {
		Token string `json:"token"`
	}](t, f.request(t, "POST", "/auth/login", "", "application/json", body, 200))
	if result.Token == "" {
		t.Fatal("login returned no token")
	}
	return result.Token
}
func (f *operationalHTTPFixture) uploadFile(t *testing.T, token, kind, filename, orderType string, data []byte, want int) string {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteField("order_type", orderType); err != nil {
		t.Fatal(err)
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	result := f.request(t, "POST", "/uploads/"+kind, token, w.FormDataContentType(), body.Bytes(), want)
	if want != 202 {
		return ""
	}
	u := decodeWorkflow[struct {
		ID     string `json:"upload_id"`
		Status string `json:"status"`
	}](t, result)
	if u.ID == "" || u.Status != "processing" {
		t.Fatal("missing processing upload")
	}
	return u.ID
}
func (f *operationalHTTPFixture) processLastJob(t *testing.T) {
	t.Helper()
	j := f.queue.jobs[len(f.queue.jobs)-1]
	var err error
	if j.Type == port.IngestionJobAlarms {
		err = f.alarm.ProcessJob(context.Background(), j)
		if err != nil {
			err = f.alarm.FailJob(context.Background(), j, err.Error())
		}
	} else {
		err = f.maintenance.ProcessJob(context.Background(), j)
		if err != nil {
			err = f.maintenance.FailJob(context.Background(), j, err.Error())
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(j.StorageKey); !os.IsNotExist(err) {
		t.Fatalf("temporary upload not removed: %v", err)
	}
}
func (f *operationalHTTPFixture) uploadStatus(t *testing.T, token, id string) httpAdapter.UploadStatusResponse {
	t.Helper()
	return decodeWorkflow[httpAdapter.UploadStatusResponse](t, f.request(t, "GET", "/uploads/"+id, token, "application/json", nil, 200))
}

func TestRFIntegration_AuthenticatedOperationalFlow(t *testing.T) {
	for mode, name := range []string{"success", "recover_after_ml_503", "recover_after_invalid_ml_response"} {
		t.Run(name, func(t *testing.T) {
			f := newOperationalHTTPFixture(t)
			f.request(t, "GET", "/dashboard/summary", "", "application/json", nil, 401)
			operator := f.login(t, "operador@motiva.com.br")
			engineer := f.login(t, "engineer@example.test")
			f.request(t, "POST", "/predictions/trigger", operator, "application/json", []byte(`{}`), 403)
			if f.modelCalls.Load() != 0 {
				t.Fatal("forbidden request reached inference")
			}
			for _, kind := range []string{"alarms", "maintenance"} {
				order := ""
				if kind == "maintenance" {
					order = "ZM01"
				}
				data := importData(t, "csv", importRows(kind, order))
				id := f.uploadFile(t, operator, kind, "valid.csv", order, data, 202)
				before := f.uploadStatus(t, operator, id)
				if before.Status != "processing" || before.RowsInserted != 0 {
					t.Fatal("job ran before worker")
				}
				f.processLastJob(t)
				after := f.uploadStatus(t, operator, id)
				if after.Status != "completed" || after.RowsInserted != 2 || after.RowsProcessed != 2 {
					t.Fatalf("import failed: %+v", after)
				}
				count := len(f.queue.jobs)
				f.uploadFile(t, operator, kind, "duplicate.csv", order, data, 409)
				if len(f.queue.jobs) != count {
					t.Fatal("duplicate upload enqueued again")
				}
			}
			manual := decodeWorkflow[httpAdapter.MaintenanceResponse](t, f.request(t, "POST", "/maintenance", operator, "application/json", []byte(`{"order_type":"ZM03","station_code":"FAL","installation_location":"SP04-6-PSD-FAL-P1-PP04","start_timestamp":"2026-10-01 10:00:00","description":"Manual workflow"}`), 201))
			if manual.ID == "" || manual.Source != "manual_form" {
				t.Fatal("manual registration missing metadata")
			}
			list := decodeWorkflow[struct {
				Total int                               `json:"total"`
				Data  []httpAdapter.MaintenanceResponse `json:"data"`
			}](t, f.request(t, "GET", "/maintenance", operator, "application/json", nil, 200))
			if list.Total != 3 || len(list.Data) != 3 {
				t.Fatal("import and manual data not consolidated")
			}
			if mode != 0 {
				f.mode.Store(int32(mode))
				problem := decodeWorkflow[httpAdapter.ProblemDetails](t, f.request(t, "POST", "/predictions/trigger", engineer, "application/json", []byte(`{}`), 503))
				if problem.Code != "ERR_PREDICTION_UNAVAILABLE" || len(f.store.runs) != 0 || len(f.store.inspections) != 0 {
					t.Fatal("failed inference published a run")
				}
				f.request(t, "GET", "/predictions/latest", operator, "application/json", nil, 404)
				if len(f.store.alarms) != 2 || len(f.store.maintenances) != 3 {
					t.Fatal("inference failure damaged input data")
				}
				f.mode.Store(0)
			}
			run := decodeWorkflow[httpAdapter.PredictionRunResponse](t, f.request(t, "POST", "/predictions/trigger", engineer, "application/json", []byte(`{}`), 201))
			if run.HorizonDays != 7 || run.TotalDoorsEvaluated != 2 || run.TotalDoorsAtRisk != 1 || len(run.Predictions) != 2 || !run.Predictions[0].WillFail || run.Predictions[1].WillFail {
				t.Fatalf("invalid prediction: %+v", run)
			}
			latest := decodeWorkflow[httpAdapter.PredictionRunResponse](t, f.request(t, "GET", "/predictions/latest", operator, "application/json", nil, 200))
			if !reflect.DeepEqual(run, latest) {
				t.Fatal("latest prediction differs from created run")
			}
			inspections := decodeWorkflow[struct {
				Data []httpAdapter.InspectionResponse `json:"data"`
			}](t, f.request(t, "GET", "/inspections?status=pending", operator, "application/json", nil, 200))
			if len(inspections.Data) != 1 {
				t.Fatal("automatic inspection missing")
			}
			ins := inspections.Data[0]
			if ins.PredictionID == nil || *ins.PredictionID != run.Predictions[0].ID || ins.StationCode != "FAL" || ins.Platform != "P1" || ins.DoorNumber != 4 {
				t.Fatal("automatic inspection lost prediction identity")
			}
			for _, status := range []string{"in_progress", "completed"} {
				updated := decodeWorkflow[httpAdapter.InspectionResponse](t, f.request(t, "PATCH", "/inspections/"+ins.ID+"/status", operator, "application/json", []byte(`{"status":"`+status+`","notes":"Workflow verified"}`), 200))
				if updated.Status != status || (updated.CompletedAt != nil) != (status == "completed") {
					t.Fatal("inspection status not persisted")
				}
			}
			summary := decodeWorkflow[httpAdapter.DashboardSummaryResponse](t, f.request(t, "GET", "/dashboard/summary", operator, "application/json", nil, 200))
			if summary.TotalDoors != 2 || summary.DoorsAtRisk != 1 || summary.PendingInspections != 0 || summary.CompletedInspections != 1 || len(summary.CriticalStations) != 1 || summary.CriticalStations[0].StationCode != "FAL" || summary.CriticalStations[0].DoorsAtRisk != 1 {
				t.Fatalf("inconsistent summary: %+v", summary)
			}
			timeline := decodeWorkflow[httpAdapter.DoorTimelineResponse](t, f.request(t, "GET", "/doors/1/timeline", operator, "application/json", nil, 200))
			if timeline.DoorID != 1 || !timeline.WillFail || len(timeline.RecentAlarms) != 1 || len(timeline.Inspections) != 1 || timeline.Inspections[0].ID != ins.ID || timeline.Inspections[0].Status != "completed" {
				t.Fatal("timeline did not consolidate workflow")
			}
			t.Log("HTTP workflow verified: login, role gate, two CSV imports, duplicate rejection, manual maintenance, prediction, inspection completion, dashboard and timeline")
		})
	}
}

func TestRFIntegration_IngestionFailureAndRecovery(t *testing.T) {
	f := newOperationalHTTPFixture(t)
	token := f.login(t, "operador@motiva.com.br")
	id := f.uploadFile(t, token, "alarms", "invalid.csv", "", []byte("unrelated\nvalue\n"), 202)
	f.processLastJob(t)
	failed := f.uploadStatus(t, token, id)
	if failed.Status != "failed" || failed.ErrorLog == "" || failed.RowsInserted != 0 || len(f.store.alarms) != 0 {
		t.Fatalf("invalid upload not failed: %+v", failed)
	}
	valid := f.uploadFile(t, token, "alarms", "valid.csv", "", importData(t, "csv", importRows("alarms", "")), 202)
	f.processLastJob(t)
	completed := f.uploadStatus(t, token, valid)
	if completed.Status != "completed" || completed.RowsInserted != 2 || len(f.store.alarms) != 2 || f.uploadStatus(t, token, id).Status != "failed" {
		t.Fatal("recovery overwrote failed upload or lost records")
	}
}
