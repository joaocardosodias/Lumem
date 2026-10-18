package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/gin-gonic/gin"
)

type mockUserRepoForHTTP struct {
	user *domain.User
	err  error
}

func (m *mockUserRepoForHTTP) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func (m *mockUserRepoForHTTP) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, nil
}

func (m *mockUserRepoForHTTP) Create(ctx context.Context, user *domain.User) error {
	return nil
}

type mockHasherForHTTP struct {
	err error
}

func (m *mockHasherForHTTP) Hash(password string) (string, error) {
	return "hashed_" + password, nil
}

func (m *mockHasherForHTTP) Compare(hashedPassword, password string) error {
	return m.err
}

type mockTokenGenForHTTP struct {
	token string
	err   error
}

func (m *mockTokenGenForHTTP) Generate(claims map[string]any, exp time.Duration) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockUserRepoForHTTP{
		user: &domain.User{
			ID:           "usr_1",
			Name:         "Operador",
			Email:        "op@linha4.com.br",
			PasswordHash: "hashed_secret",
			Role:         "operator",
			IsActive:     true,
		},
	}
	hasher := &mockHasherForHTTP{err: nil}
	tokenGen := &mockTokenGenForHTTP{token: "valid-jwt-token"}
	loginUC := usecase.NewLoginUseCase(userRepo, hasher, tokenGen, time.Hour)
	handler := httpAdapter.NewAuthHandler(loginUC)

	r := gin.New()
	r.POST("/login", handler.Login)

	t.Run("invalid json body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("bad-json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava status 400, obteve %d", w.Code)
		}
	})

	t.Run("auth error credentials", func(t *testing.T) {
		hasher.err = errors.New("mismatch")
		body, _ := json.Marshal(map[string]string{
			"email":    "op@linha4.com.br",
			"password": "wrong-password",
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("esperava status 401, obteve %d", w.Code)
		}
		hasher.err = nil
	})

	t.Run("login success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    "op@linha4.com.br",
			"password": "secret",
		})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava status 200, obteve %d", w.Code)
		}
	})
}

type mockUploadRepoForHTTP struct {
	upload *domain.FileUpload
	err    error
}

func (m *mockUploadRepoForHTTP) CreateUpload(ctx context.Context, u *domain.FileUpload) error {
	if m.err != nil {
		return m.err
	}
	u.ID = "upload-123"
	return nil
}

func (m *mockUploadRepoForHTTP) UpdateUpload(ctx context.Context, u *domain.FileUpload) error {
	return nil
}

func (m *mockUploadRepoForHTTP) GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.upload, nil
}

func (m *mockUploadRepoForHTTP) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	return nil, nil
}

func (m *mockUploadRepoForHTTP) ReconcileStaleUploads(ctx context.Context) error {
	return nil
}

type mockAlarmRepoForHTTP struct {
	mockUploadRepoForHTTP
}

func (m *mockAlarmRepoForHTTP) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	return nil
}

func (m *mockAlarmRepoForHTTP) DeleteByUploadID(ctx context.Context, uploadID string) error {
	return nil
}

func (m *mockAlarmRepoForHTTP) List(ctx context.Context, filter port.AlarmFilter) ([]*domain.Alarm, error) {
	return nil, nil
}

type mockMaintRepoForHTTP struct {
	insertErr error
	createErr error
	listErr   error
	events    []*domain.MaintenanceEvent
	created   *domain.MaintenanceEvent
}

func (m *mockMaintRepoForHTTP) BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error {
	return m.insertErr
}

func (m *mockMaintRepoForHTTP) DeleteByUploadID(ctx context.Context, uploadID string) error {
	return nil
}

func (m *mockMaintRepoForHTTP) Create(ctx context.Context, event *domain.MaintenanceEvent) error {
	if m.createErr != nil {
		return m.createErr
	}
	event.ID = "maint-100"
	m.created = event
	return nil
}

func (m *mockMaintRepoForHTTP) List(ctx context.Context, limit, offset int) ([]*domain.MaintenanceEvent, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.events, nil
}

type mockStorageForHTTP struct {
	storePath string
	hash      string
	storeErr  error
}

func (m *mockStorageForHTTP) Store(ctx context.Context, filename string, r io.Reader) (string, string, error) {
	if m.storeErr != nil {
		return "", "", m.storeErr
	}
	return m.storePath, m.hash, nil
}

func (m *mockStorageForHTTP) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("sample")), nil
}

func (m *mockStorageForHTTP) Delete(ctx context.Context, path string) error {
	return nil
}

type mockAlarmParserForHTTP struct{}

func (m *mockAlarmParserForHTTP) ParseAlarms(r io.Reader, format string) ([]port.RawAlarmRecord, error) {
	return []port.RawAlarmRecord{}, nil
}

type mockMaintenanceParserForHTTP struct{}

func (m *mockMaintenanceParserForHTTP) ParseMaintenance(r io.Reader, format string, defaultOrderType string) ([]port.RawMaintenanceRecord, error) {
	return []port.RawMaintenanceRecord{}, nil
}

type mockJobRunnerForHTTP struct{}

func (m *mockJobRunnerForHTTP) Run(task func()) {
}

func createMultipartRequest(url, fieldName, filename, content string) (*http.Request, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, strings.NewReader(content)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req := httptest.NewRequest(http.MethodPost, url, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func setupUploadTestRouter() (*gin.Engine, *mockUploadRepoForHTTP, *mockStorageForHTTP) {
	gin.SetMode(gin.TestMode)
	alarmRepo := &mockAlarmRepoForHTTP{}
	uploadRepo := &mockUploadRepoForHTTP{}
	maintRepo := &mockMaintRepoForHTTP{}
	store := &mockStorageForHTTP{storePath: "/tmp/test.csv", hash: "hash123"}
	ap := &mockAlarmParserForHTTP{}
	mp := &mockMaintenanceParserForHTTP{}
	runner := &mockJobRunnerForHTTP{}

	alarmsUC := usecase.NewIngestAlarmsUseCase(alarmRepo, uploadRepo, store, ap, runner, nil, nil)
	maintUC := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, store, mp, runner, nil, nil)
	handler := httpAdapter.NewUploadHandler(alarmsUC, maintUC)

	r := gin.New()
	r.POST("/uploads/alarms", handler.UploadAlarms)
	r.POST("/uploads/maintenance", handler.UploadMaintenance)
	r.GET("/uploads/:id", handler.GetUploadStatus)
	return r, uploadRepo, store
}

func TestUploadHandler_Alarms(t *testing.T) {
	r, _, store := setupUploadTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/uploads/alarms", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para envio sem arquivo, obteve %d", w.Code)
	}

	req, _ = createMultipartRequest("/uploads/alarms", "file", "alarmes.csv", "station,door\nPAULISTA,1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("esperava 202 aceito, obteve %d", w.Code)
	}

	store.storeErr = errors.New("falha no storage")
	req, _ = createMultipartRequest("/uploads/alarms", "file", "alarmes.csv", "station,door\nPAULISTA,1")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, obteve %d", w.Code)
	}
}

func TestUploadHandler_Maintenance(t *testing.T) {
	r, _, store := setupUploadTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/uploads/maintenance", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para envio sem arquivo, obteve %d", w.Code)
	}

	req, _ = createMultipartRequest("/uploads/maintenance", "file", "maint.csv", "order,station\nORD1,HIG")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("esperava 202 aceito, obteve %d", w.Code)
	}

	store.storeErr = errors.New("falha no storage")
	req, _ = createMultipartRequest("/uploads/maintenance", "file", "maint.csv", "order,station\nORD1,HIG")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, obteve %d", w.Code)
	}
}

func TestUploadHandler_GetStatus(t *testing.T) {
	r, uploadRepo, _ := setupUploadTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/uploads/%20", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para id vazio, obteve %d", w.Code)
	}

	uploadRepo.err = domain.ErrUploadNotFound
	req = httptest.NewRequest(http.MethodGet, "/uploads/upload-999", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("esperava 404, obteve %d", w.Code)
	}
	uploadRepo.err = nil

	now := time.Now()
	uploadRepo.upload = &domain.FileUpload{
		ID:          "upload-123",
		FileName:    "test.csv",
		FileType:    "alarms",
		Status:      "completed",
		CreatedAt:   now,
		CompletedAt: &now,
	}
	req = httptest.NewRequest(http.MethodGet, "/uploads/upload-123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}

	uploadRepo.upload.CompletedAt = nil
	uploadRepo.upload.Status = "processing"
	req = httptest.NewRequest(http.MethodGet, "/uploads/upload-123", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}
}

type mockDashboardRepoForHTTP struct {
	summary     *domain.DashboardSummary
	timeline    *domain.DoorTimeline
	status      []*domain.DoorStatusItem
	summaryErr  error
	timelineErr error
	statusErr   error
}

func (m *mockDashboardRepoForHTTP) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summary, nil
}

func (m *mockDashboardRepoForHTTP) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	if m.timelineErr != nil {
		return nil, m.timelineErr
	}
	return m.timeline, nil
}

func (m *mockDashboardRepoForHTTP) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return m.status, nil
}

type mockPredictionRepoForHTTP struct {
	run        *domain.PredictionRun
	doors      []domain.Door
	runErr     error
	doorsErr   error
	saveRunErr error
}

func (m *mockPredictionRepoForHTTP) GetRunByID(ctx context.Context, id string) (*domain.PredictionRun, error) {
	if m.run == nil || m.run.ID != id {
		return nil, domain.ErrExperimentNotFound
	}
	return m.run, nil
}

func (m *mockPredictionRepoForHTTP) ListExperiments(ctx context.Context, limit int) ([]domain.PredictionRun, error) {
	return []domain.PredictionRun{}, nil
}

func (m *mockPredictionRepoForHTTP) PromoteExperiment(ctx context.Context, id, userID string) error {
	if m.run == nil || m.run.ID != id {
		return domain.ErrExperimentNotFound
	}
	m.run.IsExperiment = false
	return nil
}

func (m *mockPredictionRepoForHTTP) SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	if m.saveRunErr != nil {
		return m.saveRunErr
	}
	run.ID = "run-1"
	run.Predictions = append([]domain.PredictionItem(nil), items...)
	m.run = run
	return nil
}

func (m *mockPredictionRepoForHTTP) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.run, nil
}

func (m *mockPredictionRepoForHTTP) GetDoorHistory(ctx context.Context, doorID, limit int) ([]domain.DoorPredictionHistoryItem, error) {
	return []domain.DoorPredictionHistoryItem{}, nil
}

func (m *mockPredictionRepoForHTTP) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	if m.doorsErr != nil {
		return nil, m.doorsErr
	}
	return m.doors, nil
}

type mockMLClientForHTTP struct {
	preds []domain.PredictionItem
	err   error
}

func (m *mockMLClientForHTTP) PredictBatch(ctx context.Context, doors []domain.Door) (*port.MLPredictionBatch, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &port.MLPredictionBatch{ModelVersion: "v1.0.0", Predictions: m.preds}, nil
}

type mockUOWStoreForHTTP struct {
	predRepo port.PredictionRepository
	insRepo  port.InspectionRepository
}

func (s *mockUOWStoreForHTTP) Predictions() port.PredictionRepository   { return s.predRepo }
func (s *mockUOWStoreForHTTP) Inspections() port.InspectionRepository   { return s.insRepo }
func (s *mockUOWStoreForHTTP) Maintenances() port.MaintenanceRepository { return nil }
func (s *mockUOWStoreForHTTP) Alarms() port.AlarmRepository             { return nil }
func (s *mockUOWStoreForHTTP) Uploads() port.UploadRepository           { return nil }

type mockUOWForHTTP struct {
	store port.UnitOfWorkStore
}

func (u *mockUOWForHTTP) Do(ctx context.Context, fn func(store port.UnitOfWorkStore) error) error {
	return fn(u.store)
}

func setupDashboardTestRouter() (*gin.Engine, *mockDashboardRepoForHTTP, *mockPredictionRepoForHTTP) {
	gin.SetMode(gin.TestMode)
	now := time.Now()

	dashRepo := &mockDashboardRepoForHTTP{
		summary: &domain.DashboardSummary{
			TotalDoors:           10,
			DoorsAtRisk:          2,
			PendingInspections:   1,
			CompletedInspections: 1,
			CriticalStations: []domain.StationRiskSummary{
				{StationCode: "HIG", DoorsAtRisk: 2},
			},
		},
		timeline: &domain.DoorTimeline{
			Door: domain.Door{
				ID:          1,
				StationCode: "HIG",
				Platform:    "1",
				DoorNumber:  1,
				SAPTag:      "TAG-1",
				Status:      "OPERATIONAL",
			},
			StationName:     "Higienopolis",
			WillFail:        true,
			LastEvaluatedAt: &now,
			RecentAlarms: []*domain.Alarm{
				{
					ID:                     "alm-1",
					Tag:                    "TAG-1",
					AlarmType:              "CRITICAL",
					DurationSeconds:        10,
					EventTimestamp:         "2026-09-05 10:00:00",
					NormalizationTimestamp: "2026-09-05 10:00:10",
				},
			},
			Maintenances: []*domain.MaintenanceEvent{
				{
					ID:             "mnt-1",
					OrderType:      "CORRETIVA",
					SAPOrderNumber: "ORD-1",
					StartTimestamp: "2026-09-05 10:00:00",
					DurationHours:  1.0,
					Description:    "Fix",
					Symptom:        "Err",
					Source:         "manual",
				},
			},
			Inspections: []*domain.Inspection{
				{
					ID:          "ins-1",
					Status:      "completed",
					Notes:       "ok",
					CompletedAt: &now,
				},
			},
		},
		status: []*domain.DoorStatusItem{
			{
				ID:         1,
				Platform:   "1",
				DoorNumber: 1,
				SAPTag:     "TAG-1",
				Status:     "OPERATIONAL",
				WillFail:   false,
			},
		},
	}

	predRepo := &mockPredictionRepoForHTTP{
		run: &domain.PredictionRun{
			ID:                  "run-1",
			HorizonDays:         15,
			TotalDoorsEvaluated: 10,
			TotalDoorsAtRisk:    2,
			CreatedAt:           time.Now(),
		},
		doors: []domain.Door{
			{ID: 1, StationCode: "HIG", Platform: "1", DoorNumber: 1},
		},
	}

	mlClient := &mockMLClientForHTTP{
		preds: []domain.PredictionItem{
			{ID: "pred-1", RunID: "run-1", StationCode: "HIG", Platform: "1", DoorNumber: 1, WillFail: false},
		},
	}

	uow := &mockUOWForHTTP{
		store: &mockUOWStoreForHTTP{
			predRepo: predRepo,
			insRepo:  &mockInspectionRepoForHTTP{},
		},
	}

	dashUC := usecase.NewDashboardUseCase(dashRepo, predRepo, mlClient, nil, uow)
	handler := httpAdapter.NewDashboardHandler(dashUC)

	r := gin.New()
	r.GET("/summary", handler.GetSummary)
	r.GET("/doors/:id/timeline", handler.GetDoorTimeline)
	r.GET("/stations/:code/doors/status", handler.GetStationDoorsStatus)
	r.POST("/predictions/trigger", handler.TriggerPredictions)
	r.POST("/predictions/experiments", handler.RunPredictionExperiment)
	r.GET("/predictions/experiments", handler.ListPredictionExperiments)
	r.POST("/predictions/experiments/:id/promote", handler.PromotePredictionExperiment)
	r.GET("/predictions/latest", handler.GetLatestPredictions)
	return r, dashRepo, predRepo
}

func TestDashboardHandler_Summary(t *testing.T) {
	r, dashRepo, _ := setupDashboardTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}

	dashRepo.summaryErr = errors.New("db fail")
	req = httptest.NewRequest(http.MethodGet, "/summary", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, obteve %d", w.Code)
	}
}

func TestDashboardHandler_Timeline(t *testing.T) {
	r, dashRepo, _ := setupDashboardTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/doors/invalid/timeline", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para id nao-inteiro, obteve %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/doors/1/timeline", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}

	dashRepo.timelineErr = errors.New("timeline error")
	req = httptest.NewRequest(http.MethodGet, "/doors/1/timeline", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, obteve %d", w.Code)
	}
}

func TestDashboardHandler_StationDoorsStatus(t *testing.T) {
	r, dashRepo, _ := setupDashboardTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/stations/HIG/doors/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}

	dashRepo.statusErr = errors.New("doors error")
	req = httptest.NewRequest(http.MethodGet, "/stations/HIG/doors/status", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, obteve %d", w.Code)
	}
}

func TestDashboardHandler_Predictions(t *testing.T) {
	r, _, predRepo := setupDashboardTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/predictions/trigger", strings.NewReader(`{"horizon_days":15}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("esperava 201, obteve %d", w.Code)
	}

	predRepo.doorsErr = errors.New("doors inference error")
	req = httptest.NewRequest(http.MethodPost, "/predictions/trigger", strings.NewReader(`{"horizon_days":15}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, obteve %d", w.Code)
	}
	predRepo.doorsErr = nil

	req = httptest.NewRequest(http.MethodGet, "/predictions/latest", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}

	predRepo.runErr = errors.New("latest run error")
	req = httptest.NewRequest(http.MethodGet, "/predictions/latest", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("esperava 500, obteve %d", w.Code)
	}
}

func TestDashboardHandler_PredictionExperimentLifecycle(t *testing.T) {
	r, _, _ := setupDashboardTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/predictions/experiments", strings.NewReader(`{"threshold":0.35}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("esperava 201 ao executar experimento, obteve %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/predictions/experiments?limit=20", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200 ao listar experimentos, obteve %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/predictions/experiments/run-1/promote", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200 ao promover experimento, obteve %d: %s", w.Code, w.Body.String())
	}
}

type mockInspectionRepoForHTTP struct {
	created   *domain.Inspection
	createErr error
	item      *domain.Inspection
	getErr    error
	items     []*domain.Inspection
	listErr   error
	updateErr error
}

func (m *mockInspectionRepoForHTTP) Create(ctx context.Context, ins *domain.Inspection) error {
	if m.createErr != nil {
		return m.createErr
	}
	ins.ID = "ins-10"
	m.created = ins
	return nil
}

func (m *mockInspectionRepoForHTTP) UpdateStatus(ctx context.Context, id, status, notes string, completedAt *time.Time) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *mockInspectionRepoForHTTP) GetByID(ctx context.Context, id string) (*domain.Inspection, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.item, nil
}

func (m *mockInspectionRepoForHTTP) List(ctx context.Context, limit, offset int, status string) ([]*domain.Inspection, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func setupInspectionTestRouter() (*gin.Engine, *mockInspectionRepoForHTTP, *domain.Inspection) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	ins := &domain.Inspection{
		ID:          "ins-1",
		StationCode: "HIG",
		Platform:    "P1",
		DoorNumber:  1,
		Status:      "pending",
		CompletedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	repo := &mockInspectionRepoForHTTP{
		item:  ins,
		items: []*domain.Inspection{ins},
	}

	uc := usecase.NewInspectionUseCase(repo)
	handler := httpAdapter.NewInspectionHandler(uc)

	r := gin.New()
	r.POST("/inspections", handler.Create)
	r.PATCH("/inspections/:id/status", handler.UpdateStatus)
	r.GET("/inspections", handler.List)
	r.GET("/inspections/:id", handler.GetByID)

	return r, repo, ins
}

func TestInspectionHandler_Create(t *testing.T) {
	r, _, _ := setupInspectionTestRouter()

	t.Run("create invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/inspections", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, obteve %d", w.Code)
		}
	})

	t.Run("create validation error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/inspections", strings.NewReader(`{"station_code":""}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, obteve %d", w.Code)
		}
	})

	t.Run("create success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"station_code": "HIG",
			"platform":     "P1",
			"door_number":  2,
		})
		req := httptest.NewRequest(http.MethodPost, "/inspections", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("esperava 201, obteve %d", w.Code)
		}
	})
}

func TestInspectionHandler_UpdateStatus(t *testing.T) {
	r, repo, _ := setupInspectionTestRouter()

	t.Run("update status empty id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/inspections/%20/status", strings.NewReader(`{"status":"in_progress"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, obteve %d", w.Code)
		}
	})

	t.Run("update status invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/inspections/ins-1/status", strings.NewReader("bad-json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, obteve %d", w.Code)
		}
	})

	t.Run("update status error", func(t *testing.T) {
		repo.getErr = domain.ErrInspectionNotFound
		req := httptest.NewRequest(http.MethodPatch, "/inspections/ins-1/status", strings.NewReader(`{"status":"in_progress"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("esperava 404, obteve %d", w.Code)
		}
		repo.getErr = nil
	})

	t.Run("update status success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/inspections/ins-1/status", strings.NewReader(`{"status":"in_progress"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, obteve %d", w.Code)
		}
	})

	t.Run("update status completed success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/inspections/ins-1/status", strings.NewReader(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, obteve %d", w.Code)
		}
	})
}

func TestInspectionHandler_ListAndGetByID(t *testing.T) {
	r, repo, _ := setupInspectionTestRouter()

	t.Run("list success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/inspections?limit=10&offset=0", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, obteve %d", w.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		repo.listErr = errors.New("db error")
		req := httptest.NewRequest(http.MethodGet, "/inspections", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("esperava 500, obteve %d", w.Code)
		}
		repo.listErr = nil
	})

	t.Run("get by id empty id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/inspections/%20", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, obteve %d", w.Code)
		}
	})

	t.Run("get by id not found", func(t *testing.T) {
		repo.getErr = domain.ErrInspectionNotFound
		req := httptest.NewRequest(http.MethodGet, "/inspections/ins-not-found", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("esperava 404, obteve %d", w.Code)
		}
		repo.getErr = nil
	})

	t.Run("get by id success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/inspections/ins-1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, obteve %d", w.Code)
		}
	})
}

func TestMaintenanceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	maintEvent := &domain.MaintenanceEvent{
		ID:             "maint-1",
		SAPOrderNumber: "ORD-12345",
		OrderType:      "CORRETIVA",
		StationCode:    "HIG",
		StartTimestamp: "2026-09-05 10:00:00",
		DurationHours:  2.0,
		Description:    "Troca de peca",
		CreatedAt:      now,
	}

	repo := &mockMaintRepoForHTTP{
		events: []*domain.MaintenanceEvent{maintEvent},
	}

	uc := usecase.NewCreateMaintenanceUseCase(repo)
	handler := httpAdapter.NewMaintenanceHandler(uc)

	r := gin.New()
	r.POST("/maintenance", handler.Create)
	r.GET("/maintenance", handler.List)

	t.Run("create invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/maintenance", strings.NewReader("bad-json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, obteve %d", w.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"sap_order_number": "",
		})
		req := httptest.NewRequest(http.MethodPost, "/maintenance", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("esperava 400, obteve %d", w.Code)
		}
	})

	t.Run("create success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"sap_order_number": "ORD-999",
			"order_type":       "corretiva",
			"station_code":     "HIG",
			"start_timestamp":  now.Format(time.RFC3339),
			"duration_hours":   1.5,
			"description":      "Ajuste de sensor",
		})
		req := httptest.NewRequest(http.MethodPost, "/maintenance", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("esperava 201, obteve %d", w.Code)
		}
	})

	t.Run("list success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/maintenance?limit=10&offset=0", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, obteve %d", w.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		repo.listErr = errors.New("db query error")
		req := httptest.NewRequest(http.MethodGet, "/maintenance", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("esperava 500, obteve %d", w.Code)
		}
		repo.listErr = nil
	})
}

func TestMetricsMiddlewareAndHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(httpAdapter.PrometheusMetricsMiddleware())
	r.GET("/metrics", httpAdapter.MetricsHandler())
	r.GET("/healthz/live", func(c *gin.Context) {
		c.String(http.StatusOK, "live")
	})
	r.GET("/api/test-ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.GET("/api/test-429", func(c *gin.Context) {
		c.Status(http.StatusTooManyRequests)
	})

	req1 := httptest.NewRequest(http.MethodGet, "/api/test-ok", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/test-429", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("esperava 429, obteve %d", w2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w3.Code)
	}

	req4 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("esperava 200 para /metrics, obteve %d", w4.Code)
	}
	if !strings.Contains(w4.Body.String(), "http_requests_total") {
		t.Fatalf("esperava conteudo de metricas contendo http_requests_total")
	}

	req5 := httptest.NewRequest(http.MethodGet, "/unregistered-fallback", nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
}

func (p *mockAlarmParserForHTTP) StreamAlarms(ctx context.Context, r io.Reader, format string, yield func(port.RawAlarmRecord) error) error {
	records, err := p.ParseAlarms(r, format)
	if err != nil {
		return err
	}
	for _, row := range records {
		if err := yield(row); err != nil {
			return err
		}
	}
	return nil
}

func (p *mockMaintenanceParserForHTTP) StreamMaintenance(ctx context.Context, r io.Reader, format string, orderType string, yield func(port.RawMaintenanceRecord) error) error {
	records, err := p.ParseMaintenance(r, format, orderType)
	if err != nil {
		return err
	}
	for _, row := range records {
		if err := yield(row); err != nil {
			return err
		}
	}
	return nil
}
