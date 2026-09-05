package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/eventbus"
	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/parser"
	"github.com/g04/backend/internal/adapter/scheduler"
	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type inMemoryStore struct {
	mu           sync.RWMutex
	users        map[string]*domain.User
	uploads      map[string]*domain.FileUpload
	alarms       []*domain.Alarm
	maintenances []*domain.MaintenanceEvent
	inspections  map[string]*domain.Inspection
	doors        []domain.Door
	stations     []domain.Station
	runs         []*domain.PredictionRun
	predictions  []domain.PredictionItem
}

func newInMemoryStore() *inMemoryStore {
	store := &inMemoryStore{
		users:       make(map[string]*domain.User),
		uploads:     make(map[string]*domain.FileUpload),
		inspections: make(map[string]*domain.Inspection),
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("operador123"), bcrypt.MinCost)
	store.users["operador@motiva.com.br"] = &domain.User{
		ID:           "usr_operator_01",
		Name:         "Operador Linha 4",
		Email:        "operador@motiva.com.br",
		PasswordHash: string(hash),
		Role:         "operator",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	store.stations = []domain.Station{
		{ID: 1, Code: "FAL", Name: "Faria Lima"},
		{ID: 2, Code: "PIN", Name: "Pinheiros"},
	}

	for _, s := range store.stations {
		for _, plt := range []string{"P1", "P2"} {
			for d := 1; d <= 24; d++ {
				doorID := len(store.doors) + 1
				tag := fmt.Sprintf("SP04-6-PSD-%s-%s-PP%02d", s.Code, plt, d)
				store.doors = append(store.doors, domain.Door{
					ID:          doorID,
					StationID:   s.ID,
					StationCode: s.Code,
					Platform:    plt,
					DoorNumber:  d,
					SAPTag:      tag,
					Status:      "active",
					CreatedAt:   time.Now(),
				})
			}
		}
	}

	return store
}

type intUserRepo struct{ store *inMemoryStore }

func (r *intUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	u, exists := r.store.users[email]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}
func (r *intUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	for _, u := range r.store.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
func (r *intUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.users[user.Email] = user
	return nil
}

type intAlarmRepo struct{ store *inMemoryStore }

func (r *intAlarmRepo) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if upload.ID == "" {
		upload.ID = uuid.New().String()
	}
	upload.CreatedAt = time.Now()
	r.store.uploads[upload.ID] = upload
	return nil
}
func (r *intAlarmRepo) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.uploads[upload.ID] = upload
	return nil
}
func (r *intAlarmRepo) GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	u, exists := r.store.uploads[id]
	if !exists {
		return nil, domain.ErrUploadNotFound
	}
	return u, nil
}
func (r *intAlarmRepo) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	for _, u := range r.store.uploads {
		if u.FileHash == hash {
			return u, nil
		}
	}
	return nil, nil
}
func (r *intAlarmRepo) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.alarms = append(r.store.alarms, alarms...)
	return nil
}

func (r *intAlarmRepo) DeleteByUploadID(ctx context.Context, uploadID string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	filtered := r.store.alarms[:0]
	for _, a := range r.store.alarms {
		if a.UploadID != uploadID {
			filtered = append(filtered, a)
		}
	}
	r.store.alarms = filtered
	return nil
}

func (r *intAlarmRepo) ReconcileStaleUploads(ctx context.Context) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	for _, u := range r.store.uploads {
		if u.Status == "processing" {
			u.Status = "failed"
			u.ErrorLog = "processo interrompido por reinicializacao do servidor"
		}
	}
	return nil
}

type intMaintenanceRepo struct{ store *inMemoryStore }

func (r *intMaintenanceRepo) BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.maintenances = append(r.store.maintenances, events...)
	return nil
}

func (r *intMaintenanceRepo) DeleteByUploadID(ctx context.Context, uploadID string) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	filtered := r.store.maintenances[:0]
	for _, m := range r.store.maintenances {
		if m.UploadID != uploadID {
			filtered = append(filtered, m)
		}
	}
	r.store.maintenances = filtered
	return nil
}
func (r *intMaintenanceRepo) Create(ctx context.Context, event *domain.MaintenanceEvent) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.CreatedAt = time.Now()
	r.store.maintenances = append(r.store.maintenances, event)
	return nil
}
func (r *intMaintenanceRepo) List(ctx context.Context, limit, offset int) ([]*domain.MaintenanceEvent, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	if offset > len(r.store.maintenances) {
		return []*domain.MaintenanceEvent{}, nil
	}
	end := offset + limit
	if end > len(r.store.maintenances) {
		end = len(r.store.maintenances)
	}
	return r.store.maintenances[offset:end], nil
}

type intInspectionRepo struct{ store *inMemoryStore }

func (r *intInspectionRepo) Create(ctx context.Context, ins *domain.Inspection) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if ins.ID == "" {
		ins.ID = uuid.New().String()
	}
	r.store.inspections[ins.ID] = ins
	return nil
}
func (r *intInspectionRepo) UpdateStatus(ctx context.Context, id string, status, notes string, completedAt *time.Time) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	ins, exists := r.store.inspections[id]
	if !exists {
		return domain.ErrInspectionNotFound
	}
	ins.Status = status
	ins.Notes = notes
	ins.CompletedAt = completedAt
	ins.UpdatedAt = time.Now()
	return nil
}
func (r *intInspectionRepo) GetByID(ctx context.Context, id string) (*domain.Inspection, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	ins, exists := r.store.inspections[id]
	if !exists {
		return nil, domain.ErrInspectionNotFound
	}
	return ins, nil
}
func (r *intInspectionRepo) List(ctx context.Context, limit, offset int, statusFilter string) ([]*domain.Inspection, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	var list []*domain.Inspection
	for _, ins := range r.store.inspections {
		if statusFilter == "" || ins.Status == statusFilter {
			list = append(list, ins)
		}
	}
	return list, nil
}

type intDashboardRepo struct{ store *inMemoryStore }

func (r *intDashboardRepo) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	pending := 0
	completed := 0
	for _, ins := range r.store.inspections {
		if ins.Status == "pending" {
			pending++
		} else if ins.Status == "completed" {
			completed++
		}
	}

	doorsAtRisk := 0
	if len(r.store.runs) > 0 {
		doorsAtRisk = r.store.runs[len(r.store.runs)-1].TotalDoorsAtRisk
	}

	return &domain.DashboardSummary{
		TotalDoors:           len(r.store.doors),
		DoorsAtRisk:          doorsAtRisk,
		PendingInspections:   pending,
		CompletedInspections: completed,
		CriticalStations: []domain.StationRiskSummary{
			{StationCode: "FAL", DoorsAtRisk: doorsAtRisk},
		},
	}, nil
}
func (r *intDashboardRepo) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	var targetDoor *domain.Door
	for _, d := range r.store.doors {
		if d.ID == doorID {
			targetDoor = &d
			break
		}
	}
	if targetDoor == nil {
		return nil, domain.ErrDoorNotFound
	}

	stationName := "Faria Lima"
	if targetDoor.StationCode == "PIN" {
		stationName = "Pinheiros"
	}

	var doorAlarms []*domain.Alarm
	for _, a := range r.store.alarms {
		if a.DoorNumber == targetDoor.DoorNumber && a.StationCode == targetDoor.StationCode {
			doorAlarms = append(doorAlarms, a)
		}
	}

	var doorInspections []*domain.Inspection
	for _, i := range r.store.inspections {
		if i.DoorNumber == targetDoor.DoorNumber && i.StationCode == targetDoor.StationCode && i.Platform == targetDoor.Platform {
			doorInspections = append(doorInspections, i)
		}
	}

	willFail := false
	for _, p := range r.store.predictions {
		if p.DoorNumber == targetDoor.DoorNumber && p.StationCode == targetDoor.StationCode && p.Platform == targetDoor.Platform {
			willFail = p.WillFail
			break
		}
	}

	now := time.Now()
	return &domain.DoorTimeline{
		Door:            *targetDoor,
		StationName:     stationName,
		WillFail:        willFail,
		RecentAlarms:    doorAlarms,
		Maintenances:    r.store.maintenances,
		Inspections:     doorInspections,
		LastEvaluatedAt: &now,
	}, nil
}
func (r *intDashboardRepo) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()

	var list []*domain.DoorStatusItem
	for _, d := range r.store.doors {
		if d.StationCode == stationCode {
			willFail := false
			for _, p := range r.store.predictions {
				if p.DoorNumber == d.DoorNumber && p.StationCode == d.StationCode && p.Platform == d.Platform {
					willFail = p.WillFail
					break
				}
			}
			list = append(list, &domain.DoorStatusItem{
				ID:         d.ID,
				Platform:   d.Platform,
				DoorNumber: d.DoorNumber,
				SAPTag:     d.SAPTag,
				Status:     d.Status,
				WillFail:   willFail,
			})
		}
	}
	return list, nil
}

type intPredictionRepo struct{ store *inMemoryStore }

func (r *intPredictionRepo) SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.runs = append(r.store.runs, run)
	r.store.predictions = items
	return nil
}
func (r *intPredictionRepo) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	if len(r.store.runs) == 0 {
		return nil, domain.ErrNoPredictions
	}
	latest := r.store.runs[len(r.store.runs)-1]
	latest.Predictions = r.store.predictions
	return latest, nil
}
func (r *intPredictionRepo) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	r.store.mu.RLock()
	defer r.store.mu.RUnlock()
	return r.store.doors, nil
}

type intMLClient struct{}

func (m *intMLClient) PredictBatch(ctx context.Context, doors []domain.Door) ([]domain.PredictionItem, error) {
	items := make([]domain.PredictionItem, 0, len(doors))
	for _, d := range doors {
		willFail := (d.StationCode == "FAL" && d.Platform == "P1" && d.DoorNumber == 4)
		items = append(items, domain.PredictionItem{
			StationCode: d.StationCode,
			Platform:    d.Platform,
			DoorNumber:  d.DoorNumber,
			WillFail:    willFail,
		})
	}
	return items, nil
}

func setupCompleteWorkflowApp() (*gin.Engine, *inMemoryStore) {
	gin.SetMode(gin.TestMode)
	jwtSecret := "segredo-super-seguro-de-teste-de-integracao-123"

	store := newInMemoryStore()
	userRepo := &intUserRepo{store: store}
	alarmRepo := &intAlarmRepo{store: store}
	maintRepo := &intMaintenanceRepo{store: store}
	insRepo := &intInspectionRepo{store: store}
	dashRepo := &intDashboardRepo{store: store}
	predRepo := &intPredictionRepo{store: store}

	bus := eventbus.NewInMemoryEventBus(nil)
	usecase.RegisterEventSubscribers(bus, insRepo, nil, nil)

	ingestionStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	maintParser := parser.NewDefaultMaintenanceParser()
	mlClient := &intMLClient{}

	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator(jwtSecret)
	syncRunner := scheduler.NewSyncJobRunner()
	loginUC := usecase.NewLoginUseCase(userRepo, hasher, tokenGen, time.Hour)
	ingestAlarmsUC := usecase.NewIngestAlarmsUseCase(alarmRepo, alarmRepo, ingestionStorage, alarmParser, syncRunner)
	ingestMaintUC := usecase.NewIngestMaintenanceUseCase(maintRepo, alarmRepo, ingestionStorage, maintParser, syncRunner)
	createMaintUC := usecase.NewCreateMaintenanceUseCase(maintRepo)
	insUC := usecase.NewInspectionUseCase(insRepo)
	dashUC := usecase.NewDashboardUseCase(dashRepo, predRepo, mlClient, bus)

	authHandler := httpAdapter.NewAuthHandler(loginUC)
	uploadHandler := httpAdapter.NewUploadHandler(ingestAlarmsUC, ingestMaintUC)
	maintHandler := httpAdapter.NewMaintenanceHandler(createMaintUC)
	insHandler := httpAdapter.NewInspectionHandler(insUC)
	dashHandler := httpAdapter.NewDashboardHandler(dashUC)
	docsHandler := httpAdapter.NewDocsHandler("../../docs/openapi.json")

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpAdapter.CorrelationIDMiddleware())

	router.GET("/openapi.json", docsHandler.OpenAPISpec)
	router.GET("/docs", docsHandler.SwaggerUI)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/uploads/alarms", uploadHandler.UploadAlarms)
		v1.POST("/uploads/maintenance", uploadHandler.UploadMaintenance)
		v1.GET("/uploads/:id", uploadHandler.GetUploadStatus)

		v1.GET("/maintenance", maintHandler.List)
		v1.POST("/maintenance", maintHandler.Create)

		v1.POST("/inspections", insHandler.Create)
		v1.GET("/inspections", insHandler.List)
		v1.GET("/inspections/:id", insHandler.GetByID)
		v1.PATCH("/inspections/:id/status", insHandler.UpdateStatus)

		v1.GET("/dashboard/summary", dashHandler.GetSummary)
		v1.GET("/doors/:id/timeline", dashHandler.GetDoorTimeline)
		v1.GET("/stations/:code/doors/status", dashHandler.GetStationDoorsStatus)
		v1.POST("/predictions/trigger", dashHandler.TriggerPredictions)
		v1.GET("/predictions/latest", dashHandler.GetLatestPredictions)
	}

	protected := v1.Group("")
	protected.Use(httpAdapter.AuthMiddleware(jwtSecret))
	{
		protected.GET("/auth/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"id":    c.GetString("userID"),
				"email": c.GetString("userEmail"),
				"role":  c.GetString("userRole"),
			})
		})
	}

	return router, store
}

func TestIntegration_CompleteOperationalPipeline(t *testing.T) {
	router, store := setupCompleteWorkflowApp()

	loginPayload := map[string]string{
		"email":    "operador@motiva.com.br",
		"password": "operador123",
	}
	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("etapa 1 falhou: status %d (body: %s)", w.Code, w.Body.String())
	}

	var authResp struct {
		Token string `json:"token"`
		User  struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &authResp)

	if authResp.Token == "" || authResp.User.Role != "operator" {
		t.Fatalf("etapa 1 falhou: token invalido ou perfil incorreto")
	}

	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+authResp.Token)
	wMe := httptest.NewRecorder()
	router.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("etapa 2 falhou: acesso autenticado negado com codigo %d", wMe.Code)
	}

	csvAlarmData := `tag,estação,porta,alarme,tempo_normalizacao_seg,horário do alarme
SP04-6-PSD-FAL-P1-PP04,FAL,4,FALHA DE COMUNICACAO,22,2026-09-03 14:00:00
SP04-6-PSD-PIN-P2-PP08,PIN,8,LACO ABERTO,10,2026-09-03 14:10:00
`
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	part, _ := mw.CreateFormFile("file", "alarmes_operacao.csv")
	_, _ = io.Copy(part, strings.NewReader(csvAlarmData))
	_ = mw.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/alarms", buf)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	uploadW := httptest.NewRecorder()
	router.ServeHTTP(uploadW, uploadReq)

	if uploadW.Code != http.StatusAccepted {
		t.Fatalf("etapa 3 falhou: upload de alarmes rejeitado com codigo %d", uploadW.Code)
	}

	var uploadResp struct {
		UploadID string `json:"upload_id"`
		Status   string `json:"status"`
	}
	_ = json.Unmarshal(uploadW.Body.Bytes(), &uploadResp)

	if uploadResp.UploadID == "" || uploadResp.Status != "processing" {
		t.Fatalf("etapa 3 falhou: payload de resposta do upload invalido")
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/uploads/"+uploadResp.UploadID, nil)
	statusW := httptest.NewRecorder()
	router.ServeHTTP(statusW, statusReq)

	if statusW.Code != http.StatusOK {
		t.Fatalf("etapa 4 falhou: consulta de upload retornou %d", statusW.Code)
	}

	var statusResp struct {
		Status        string `json:"status"`
		RowsProcessed int    `json:"rows_processed"`
		RowsInserted  int    `json:"rows_inserted"`
	}
	_ = json.Unmarshal(statusW.Body.Bytes(), &statusResp)

	if statusResp.Status != "completed" || statusResp.RowsInserted != 2 {
		t.Fatalf("etapa 4 falhou: linhas nao processadas corretamente: %+v", statusResp)
	}

	maintPayload := httpAdapter.CreateMaintenanceRequest{
		OrderType:            "ZM01",
		StationCode:          "FAL",
		InstallationLocation: "SP04-6-PSD-FAL-P1-PP04",
		StartTimestamp:       "2026-09-03 14:15:00",
		Description:          "Verificacao no trilho da porta 4",
		Symptom:              "Porta com fechamento lento",
		Cause:                "Atrito mecanico",
		ActionTaken:          "Limpeza e lubrificacao",
	}
	maintBody, _ := json.Marshal(maintPayload)
	maintReq := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance", bytes.NewReader(maintBody))
	maintReq.Header.Set("Content-Type", "application/json")
	maintW := httptest.NewRecorder()
	router.ServeHTTP(maintW, maintReq)

	if maintW.Code != http.StatusCreated {
		t.Fatalf("etapa 5 falhou: criacao de manutencao falhou com status %d", maintW.Code)
	}

	predReq := httptest.NewRequest(http.MethodPost, "/api/v1/predictions/trigger", bytes.NewReader([]byte(`{"horizon_days": 7}`)))
	predReq.Header.Set("Content-Type", "application/json")
	predW := httptest.NewRecorder()
	router.ServeHTTP(predW, predReq)

	if predW.Code != http.StatusCreated {
		t.Fatalf("etapa 6 falhou: disparo de inferencia falhou com status %d", predW.Code)
	}

	var predResp httpAdapter.PredictionRunResponse
	_ = json.Unmarshal(predW.Body.Bytes(), &predResp)

	if predResp.TotalDoorsAtRisk != 1 {
		t.Fatalf("etapa 6 falhou: esperava 1 porta em risco (FAL P1 Door 4), obteve %d", predResp.TotalDoorsAtRisk)
	}

	inspListReq := httptest.NewRequest(http.MethodGet, "/api/v1/inspections?status=pending", nil)
	inspListW := httptest.NewRecorder()
	router.ServeHTTP(inspListW, inspListReq)

	if inspListW.Code != http.StatusOK {
		t.Fatalf("etapa 7 falhou: listagem de inspecoes retornou %d", inspListW.Code)
	}

	var inspListResp struct {
		Total int                              `json:"total"`
		Data  []httpAdapter.InspectionResponse `json:"data"`
	}
	_ = json.Unmarshal(inspListW.Body.Bytes(), &inspListResp)

	if inspListResp.Total != 1 || inspListResp.Data[0].Priority != "critical" {
		t.Fatalf("etapa 7 falhou: inspecao critica nao gerada pelo subscriber do event bus: %+v", inspListResp)
	}

	autoInspectionID := inspListResp.Data[0].ID

	updateStatusReq := httptest.NewRequest(http.MethodPatch, "/api/v1/inspections/"+autoInspectionID+"/status", bytes.NewReader([]byte(`{
		"status": "completed",
		"notes": "Inspecao preventiva executada com sucesso pelo operador de campo"
	}`)))
	updateStatusReq.Header.Set("Content-Type", "application/json")
	updateStatusW := httptest.NewRecorder()
	router.ServeHTTP(updateStatusW, updateStatusReq)

	if updateStatusW.Code != http.StatusOK {
		t.Fatalf("etapa 8 falhou: atualizacao de status da inspecao falhou com status %d", updateStatusW.Code)
	}

	var falDoor4ID int
	for _, d := range store.doors {
		if d.StationCode == "FAL" && d.Platform == "P1" && d.DoorNumber == 4 {
			falDoor4ID = d.ID
			break
		}
	}

	timelineReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/doors/%d/timeline", falDoor4ID), nil)
	timelineW := httptest.NewRecorder()
	router.ServeHTTP(timelineW, timelineReq)

	if timelineW.Code != http.StatusOK {
		t.Fatalf("etapa 9 falhou: consulta da linha do tempo falhou com status %d", timelineW.Code)
	}

	var timelineResp httpAdapter.DoorTimelineResponse
	_ = json.Unmarshal(timelineW.Body.Bytes(), &timelineResp)

	if !timelineResp.WillFail {
		t.Errorf("linha do tempo deveria registrar will_fail = true")
	}
	if len(timelineResp.RecentAlarms) == 0 {
		t.Errorf("linha do tempo deveria consolidar os alarmes ingeridos")
	}
	if len(timelineResp.Inspections) == 0 || timelineResp.Inspections[0].Status != "completed" {
		t.Errorf("linha do tempo deveria consolidar a inspecao automatica concluida")
	}

	summaryReq := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	summaryW := httptest.NewRecorder()
	router.ServeHTTP(summaryW, summaryReq)

	if summaryW.Code != http.StatusOK {
		t.Fatalf("etapa 10 falhou: resumo do dashboard falhou com status %d", summaryW.Code)
	}

	var summaryResp httpAdapter.DashboardSummaryResponse
	_ = json.Unmarshal(summaryW.Body.Bytes(), &summaryResp)

	if summaryResp.DoorsAtRisk != 1 {
		t.Errorf("resumo do dashboard deveria acusar 1 porta em risco, obteve %d", summaryResp.DoorsAtRisk)
	}
	if summaryResp.CompletedInspections != 1 {
		t.Errorf("resumo do dashboard deveria acusar 1 inspecao concluida, obteve %d", summaryResp.CompletedInspections)
	}
}

func TestIntegration_SecurityAndProblemDetailsRFC7807(t *testing.T) {
	router, _ := setupCompleteWorkflowApp()

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	unauthReq.Header.Set("X-Request-ID", "req-test-security-01")
	unauthW := httptest.NewRecorder()
	router.ServeHTTP(unauthW, unauthReq)

	if unauthW.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para rota protegida sem token, obteve %d", unauthW.Code)
	}

	if unauthW.Header().Get("X-Request-ID") != "req-test-security-01" {
		t.Errorf("cabecalho X-Request-ID nao propagado na resposta")
	}

	var probResp struct {
		Type      string `json:"type"`
		Title     string `json:"title"`
		Status    int    `json:"status"`
		Detail    string `json:"detail"`
		Code      string `json:"code"`
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(unauthW.Body.Bytes(), &probResp)

	if probResp.Code != "ERR_MISSING_TOKEN" || probResp.Status != 401 {
		t.Errorf("resposta RFC 7807 invalida: %+v", probResp)
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance", bytes.NewReader([]byte(`{"order_type": ""}`)))
	badReq.Header.Set("Content-Type", "application/json")
	badW := httptest.NewRecorder()
	router.ServeHTTP(badW, badReq)

	if badW.Code != http.StatusBadRequest {
		t.Fatalf("esperava 400 para payload invalido, obteve %d", badW.Code)
	}

	var badProb struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	_ = json.Unmarshal(badW.Body.Bytes(), &badProb)

	if badProb.Code != "ERR_VALIDATION" || badProb.Status != 400 {
		t.Errorf("esperava ERR_VALIDATION no problem details, obteve %+v", badProb)
	}
}
