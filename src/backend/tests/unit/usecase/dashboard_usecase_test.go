package usecase_test

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

type mockDashboardRepo struct {
	summary     *domain.DashboardSummary
	timeline    *domain.DoorTimeline
	status      []*domain.DoorStatusItem
	summaryErr  error
	timelineErr error
	statusErr   error
}

func (m *mockDashboardRepo) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summary, nil
}

func (m *mockDashboardRepo) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	if m.timelineErr != nil {
		return nil, m.timelineErr
	}
	return m.timeline, nil
}

func (m *mockDashboardRepo) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return m.status, nil
}

type mockPredictionRepo struct {
	latestRun  *domain.PredictionRun
	history    []domain.DoorPredictionHistoryItem
	doors      []domain.Door
	runErr     error
	doorsErr   error
	saveRunErr error
}

func (m *mockPredictionRepo) GetRunByID(ctx context.Context, id string) (*domain.PredictionRun, error) {
	if m.latestRun == nil || m.latestRun.ID != id {
		return nil, domain.ErrExperimentNotFound
	}
	return m.latestRun, nil
}

func (m *mockPredictionRepo) ListExperiments(ctx context.Context, limit int) ([]domain.PredictionRun, error) {
	if m.latestRun != nil && m.latestRun.IsExperiment {
		return []domain.PredictionRun{*m.latestRun}, nil
	}
	return []domain.PredictionRun{}, nil
}

func (m *mockPredictionRepo) PromoteExperiment(ctx context.Context, id, userID string) error {
	if m.latestRun == nil || m.latestRun.ID != id || !m.latestRun.IsExperiment {
		return domain.ErrExperimentNotFound
	}
	m.latestRun.IsExperiment = false
	return nil
}

func (m *mockPredictionRepo) SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	if m.saveRunErr != nil {
		return m.saveRunErr
	}
	run.ID = "run-test-123"
	run.Predictions = append([]domain.PredictionItem(nil), items...)
	m.latestRun = run
	return nil
}

func (m *mockPredictionRepo) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.latestRun, nil
}

func (m *mockPredictionRepo) GetDoorHistory(ctx context.Context, doorID, limit int) ([]domain.DoorPredictionHistoryItem, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.history, nil
}

func (m *mockPredictionRepo) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	if m.doorsErr != nil {
		return nil, m.doorsErr
	}
	return m.doors, nil
}

type mockMLClient struct {
	items []domain.PredictionItem
	err   error
}

func (m *mockMLClient) PredictBatch(ctx context.Context, doors []domain.Door) (*port.MLPredictionBatch, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.items != nil {
		return &port.MLPredictionBatch{ModelVersion: "v1.0.0", Predictions: m.items}, nil
	}
	items := make([]domain.PredictionItem, len(doors))
	for i, d := range doors {
		items[i] = domain.PredictionItem{
			ID:          "pred-item",
			RunID:       "run-test-123",
			StationCode: d.StationCode,
			Platform:    d.Platform,
			DoorNumber:  d.DoorNumber,
			WillFail:    d.DoorNumber%7 == 0,
		}
	}
	return &port.MLPredictionBatch{ModelVersion: "v1.0.0", Predictions: items}, nil
}

type mockUOWStore struct {
	predRepo port.PredictionRepository
	insRepo  port.InspectionRepository
}

func (s *mockUOWStore) Predictions() port.PredictionRepository   { return s.predRepo }
func (s *mockUOWStore) Inspections() port.InspectionRepository   { return s.insRepo }
func (s *mockUOWStore) Maintenances() port.MaintenanceRepository { return nil }
func (s *mockUOWStore) Alarms() port.AlarmRepository             { return nil }
func (s *mockUOWStore) Uploads() port.UploadRepository           { return nil }

type mockUnitOfWork struct {
	called bool
	store  *mockUOWStore
}

func (u *mockUnitOfWork) Do(ctx context.Context, fn func(store port.UnitOfWorkStore) error) error {
	u.called = true
	return fn(u.store)
}

func TestDashboardUseCase_GetSummary(t *testing.T) {
	dashRepo := &mockDashboardRepo{
		summary: &domain.DashboardSummary{
			TotalDoors:         526,
			DoorsAtRisk:        12,
			PendingInspections: 3,
			CriticalStations: []domain.StationRiskSummary{
				{StationCode: "FAL", DoorsAtRisk: 5},
			},
		},
	}
	predRepo := &mockPredictionRepo{}
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{store: &mockUOWStore{}}
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	res, err := uc.GetSummary(context.Background())
	if err != nil {
		t.Fatalf("esperava sucesso no summary: %v", err)
	}

	if res.TotalDoors != 526 {
		t.Errorf("esperava 526 portas, obteve %d", res.TotalDoors)
	}
}

func TestDashboardUseCase_GetDoorTimeline(t *testing.T) {
	now := time.Now()
	dashRepo := &mockDashboardRepo{
		timeline: &domain.DoorTimeline{
			Door: domain.Door{
				ID:         4,
				Platform:   "P1",
				DoorNumber: 4,
				SAPTag:     "SP04-6-PSD-FAL-PLT1-PP04",
			},
			StationName:     "Faria Lima",
			WillFail:        true,
			LastEvaluatedAt: &now,
			RecentAlarms: []*domain.Alarm{
				{ID: "alm-1", Tag: "TAG-1", AlarmType: "CRITICAL", DurationSeconds: 10, EventTimestamp: "2026-09-05 10:00:00"},
			},
			Maintenances: []*domain.MaintenanceEvent{
				{ID: "mnt-1", OrderType: "CORRETIVA", SAPOrderNumber: "ORD-1", StartTimestamp: "2026-09-05 10:00:00", DurationHours: 1.0, Description: "Fix", Source: "manual"},
			},
			Inspections: []*domain.Inspection{
				{ID: "ins-1", Status: "completed", CompletedAt: &now},
			},
		},
	}
	predRepo := &mockPredictionRepo{}
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{store: &mockUOWStore{}}
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	res, err := uc.GetDoorTimeline(context.Background(), 4)
	if err != nil {
		t.Fatalf("esperava sucesso no timeline: %v", err)
	}

	if !res.WillFail {
		t.Errorf("esperava WillFail = true")
	}

	_, err = uc.GetDoorTimeline(context.Background(), 0)
	if !errors.Is(err, domain.ErrInvalidDoorID) {
		t.Fatalf("esperava ErrInvalidDoorID para id de porta 0, obteve %v", err)
	}
}

func TestDashboardUseCase_GetStationDoorsStatus(t *testing.T) {
	dashRepo := &mockDashboardRepo{
		status: []*domain.DoorStatusItem{
			{ID: 1, Platform: "P1", DoorNumber: 1, Status: "healthy", WillFail: false},
			{ID: 2, Platform: "P1", DoorNumber: 2, Status: "at_risk", WillFail: true},
		},
	}
	predRepo := &mockPredictionRepo{}
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{store: &mockUOWStore{}}
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	items, err := uc.GetStationDoorsStatus(context.Background(), "FAL")
	if err != nil {
		t.Fatalf("esperava sucesso no status da estacao: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("esperava 2 itens, obteve %d", len(items))
	}

	_, err = uc.GetStationDoorsStatus(context.Background(), "")
	if !errors.Is(err, domain.ErrStationRequired) {
		t.Fatalf("esperava ErrStationRequired para codigo vazio, obteve %v", err)
	}
}

func TestDashboardUseCase_TriggerPredictions(t *testing.T) {
	dashRepo := &mockDashboardRepo{}
	predRepo := &mockPredictionRepo{
		doors: []domain.Door{
			{ID: 1, StationCode: "FAL", Platform: "P1", DoorNumber: 1},
			{ID: 7, StationCode: "FAL", Platform: "P1", DoorNumber: 7},
		},
	}
	insRepo := newMockInspectionRepo()
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{
		store: &mockUOWStore{predRepo: predRepo, insRepo: insRepo},
	}
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	run, err := uc.TriggerPredictions(context.Background(), 7)
	if err != nil {
		t.Fatalf("esperava sucesso na inferencia: %v", err)
	}

	if run.TotalDoorsEvaluated != 2 {
		t.Errorf("esperava 2 portas avaliadas, obteve %d", run.TotalDoorsEvaluated)
	}

	if run.CreatedAt.IsZero() {
		t.Errorf("created_at deveria estar preenchido")
	}

	if len(bus.published) < 2 {
		t.Errorf("esperava pelo menos 2 eventos publicados (DoorRiskDetected + PredictionRunCompleted), obteve %d", len(bus.published))
	}

	inspections, err := insRepo.List(context.Background(), 10, 0, "")
	if err != nil {
		t.Fatalf("falha ao listar inspecoes: %v", err)
	}
	if len(inspections) != 1 {
		t.Errorf("esperava 1 inspecao gerada para porta com risco, obteve %d", len(inspections))
	}
}

func TestDashboardUseCase_TriggerPredictions_WithUnitOfWork(t *testing.T) {
	dashRepo := &mockDashboardRepo{}
	predRepo := &mockPredictionRepo{
		doors: []domain.Door{
			{ID: 7, StationCode: "FAL", Platform: "P1", DoorNumber: 7},
		},
	}
	insRepo := newMockInspectionRepo()
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{
		store: &mockUOWStore{predRepo: predRepo, insRepo: insRepo},
	}

	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	_, err := uc.TriggerPredictions(context.Background(), 7)
	if err != nil {
		t.Fatalf("esperava sucesso na inferencia com UoW: %v", err)
	}

	if !uow.called {
		t.Errorf("esperava que o UnitOfWork.Do fosse chamado durante TriggerPredictions")
	}

	inspections, err := insRepo.List(context.Background(), 10, 0, "")
	if err != nil {
		t.Fatalf("falha ao listar inspecoes: %v", err)
	}
	if len(inspections) != 1 {
		t.Errorf("esperava 1 inspecao persistida atomicamente no UoW, obteve %d", len(inspections))
	}
}

type failingInspectionRepo struct {
	port.InspectionRepository
}

func (f *failingInspectionRepo) Create(ctx context.Context, ins *domain.Inspection) error {
	return errors.New("falha forcada ao persistir inspecao")
}

func TestDashboardUseCase_TriggerPredictions_UnitOfWorkRollbackOnInspectionFailure(t *testing.T) {
	dashRepo := &mockDashboardRepo{}
	predRepo := &mockPredictionRepo{
		doors: []domain.Door{
			{ID: 7, StationCode: "FAL", Platform: "P1", DoorNumber: 7},
		},
	}
	failingRepo := &failingInspectionRepo{InspectionRepository: newMockInspectionRepo()}
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{
		store: &mockUOWStore{predRepo: predRepo, insRepo: failingRepo},
	}

	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	_, err := uc.TriggerPredictions(context.Background(), 7)
	if err == nil {
		t.Fatalf("esperava erro ao falhar criacao de inspecao na transacao atomica")
	}
}

func TestDashboardUseCase_TriggerPredictions_NoActiveDoors(t *testing.T) {
	dashRepo := &mockDashboardRepo{}
	predRepo := &mockPredictionRepo{doors: []domain.Door{}}
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{store: &mockUOWStore{}}
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	_, err := uc.TriggerPredictions(context.Background(), 7)
	if !errors.Is(err, domain.ErrNoActiveDoors) {
		t.Fatalf("esperava ErrNoActiveDoors, obteve %v", err)
	}
}

func TestDashboardUseCase_GetLatestPredictions_NoPredictions(t *testing.T) {
	dashRepo := &mockDashboardRepo{}
	mockNotFoundRepo := &mockPredictionRepoNotFound{}
	ml := &mockMLClient{}
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{store: &mockUOWStore{}}
	ucNotFound := usecase.NewDashboardUseCase(dashRepo, mockNotFoundRepo, ml, bus, uow)

	_, err := ucNotFound.GetLatestPredictions(context.Background())
	if !errors.Is(err, domain.ErrNoPredictions) {
		t.Fatalf("esperava ErrNoPredictions, obteve %v", err)
	}
}

func TestDashboardUseCase_GetLatestPredictions_Success(t *testing.T) {
	predRepo := &mockPredictionRepo{
		latestRun: &domain.PredictionRun{
			ID:                  "run-latest-1",
			HorizonDays:         15,
			TotalDoorsEvaluated: 10,
			TotalDoorsAtRisk:    1,
			CreatedAt:           time.Now(),
			Predictions: []domain.PredictionItem{
				{
					ID:          "p-1",
					RunID:       "run-latest-1",
					StationCode: "HIG",
					Platform:    "P1",
					DoorNumber:  3,
					WillFail:    true,
				},
			},
		},
	}
	uc := usecase.NewGetLatestPredictionsUseCase(predRepo)
	res, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("esperava sucesso, obteve %v", err)
	}
	if res.ID != "run-latest-1" || len(res.Predictions) != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
}

func TestDashboardUseCase_GetDoorPredictionHistory(t *testing.T) {
	probability := 0.72
	threshold := 0.28
	createdAt := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	repo := &mockPredictionRepo{history: []domain.DoorPredictionHistoryItem{{
		PredictionItem: domain.PredictionItem{ID: "pred-1", RunID: "run-1", Probability: &probability, WillFail: true},
		HorizonDays:    7, ThresholdUsed: &threshold, ModelVersion: "v1.0.0", CreatedAt: createdAt,
	}}}
	uc := usecase.NewGetDoorPredictionHistoryUseCase(repo)

	items, err := uc.Execute(context.Background(), 12, 30)
	if err != nil || len(items) != 1 {
		t.Fatalf("historico inesperado: %+v, erro: %v", items, err)
	}
	if items[0].Probability == nil || *items[0].Probability != probability {
		t.Fatalf("probabilidade nao preservada: %+v", items[0])
	}
	if !items[0].WindowEndsAt.Equal(createdAt.AddDate(0, 0, 7)) {
		t.Fatalf("fim da janela incorreto: %v", items[0].WindowEndsAt)
	}
	if _, err := uc.Execute(context.Background(), 0, 30); !errors.Is(err, domain.ErrInvalidDoorID) {
		t.Fatalf("esperava ErrInvalidDoorID, obteve %v", err)
	}
}

type mockPredictionRepoNotFound struct {
	mockPredictionRepo
}

func (m *mockPredictionRepoNotFound) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	return nil, domain.ErrNoPredictions
}

type failingSyncEventBus struct {
	mockEventBus
	publishSyncCalls int
}

func (f *failingSyncEventBus) PublishSync(ctx context.Context, event domain.Event) error {
	f.publishSyncCalls++
	return errors.New("falha no publish sync")
}

func TestDashboardUseCase_TriggerPredictions_AsyncEventPublishingDoesNotFail(t *testing.T) {
	dashRepo := &mockDashboardRepo{}
	predRepo := &mockPredictionRepo{
		doors: []domain.Door{
			{ID: 7, StationCode: "FAL", Platform: "P1", DoorNumber: 7},
		},
	}
	insRepo := newMockInspectionRepo()
	ml := &mockMLClient{}
	bus := &failingSyncEventBus{}
	uow := &mockUnitOfWork{
		store: &mockUOWStore{predRepo: predRepo, insRepo: insRepo},
	}

	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, bus, uow)

	run, err := uc.TriggerPredictions(context.Background(), 7)
	if err != nil {
		t.Fatalf("esperava sucesso na inferencia mesmo com falha no publish sync: %v", err)
	}

	if run == nil {
		t.Fatalf("esperava retorno da rodada de predicao preenchido")
	}

	if bus.publishSyncCalls != 0 {
		t.Errorf("esperava 0 chamadas a PublishSync, obteve %d", bus.publishSyncCalls)
	}

	if len(bus.published) < 2 {
		t.Errorf("esperava eventos publicados assincronamente via Publish, obteve %d", len(bus.published))
	}
}

func TestDashboardUseCase_GetLatestPredictions_Error(t *testing.T) {
	predRepo := &mockPredictionRepo{runErr: errors.New("db error")}
	uc := usecase.NewGetLatestPredictionsUseCase(predRepo)

	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Fatalf("expected error from GetLatestPredictions")
	}
}

func TestGetDashboardSummaryUseCase_Error(t *testing.T) {
	dashRepo := &mockDashboardRepo{summaryErr: errors.New("summary failed")}
	uc := usecase.NewGetDashboardSummaryUseCase(dashRepo)

	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Fatalf("expected summary error")
	}
}

func TestGetDoorTimelineUseCase_Error(t *testing.T) {
	dashRepo := &mockDashboardRepo{timelineErr: errors.New("timeline failed")}
	uc := usecase.NewGetDoorTimelineUseCase(dashRepo)

	_, err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Fatalf("expected timeline error")
	}
}

func TestGetStationDoorsStatusUseCase_Error(t *testing.T) {
	dashRepo := &mockDashboardRepo{statusErr: errors.New("station status failed")}
	uc := usecase.NewGetStationDoorsStatusUseCase(dashRepo)

	_, err := uc.Execute(context.Background(), "FAL")
	if err == nil {
		t.Fatalf("expected station status error")
	}
}

func TestTriggerPredictionsUseCase_Errors(t *testing.T) {
	errRepo := &mockPredictionRepo{doorsErr: errors.New("doors error")}
	uow := &mockUnitOfWork{}
	uc1 := usecase.NewTriggerPredictionsUseCase(errRepo, &mockMLClient{}, nil, uow)
	if _, err := uc1.Execute(context.Background(), 0); err == nil {
		t.Fatalf("expected error on list doors")
	}

	emptyRepo := &mockPredictionRepo{doors: []domain.Door{}}
	uc2 := usecase.NewTriggerPredictionsUseCase(emptyRepo, &mockMLClient{}, nil, uow)
	if _, err := uc2.Execute(context.Background(), 7); !errors.Is(err, domain.ErrNoActiveDoors) {
		t.Fatalf("expected ErrNoActiveDoors")
	}

	doors := []domain.Door{{StationCode: "FAL", Platform: "P1", DoorNumber: 1}}
	validRepo := &mockPredictionRepo{doors: doors}
	errML := &mockMLClient{err: errors.New("ml failed")}
	uc3 := usecase.NewTriggerPredictionsUseCase(validRepo, errML, nil, uow)
	if _, err := uc3.Execute(context.Background(), 7); err == nil {
		t.Fatalf("expected error on predict batch")
	}

	saveErrRepo := &mockPredictionRepo{doors: doors, saveRunErr: errors.New("save failed")}
	uowSaveErr := &mockUnitOfWork{
		store: &mockUOWStore{predRepo: saveErrRepo, insRepo: newMockInspectionRepo()},
	}
	uc4 := usecase.NewTriggerPredictionsUseCase(saveErrRepo, &mockMLClient{}, nil, uowSaveErr)
	if _, err := uc4.Execute(context.Background(), 7); err == nil {
		t.Fatalf("expected error on save run")
	}

	invalidItemRepo := &mockPredictionRepo{doors: doors}
	uowNormal := &mockUnitOfWork{
		store: &mockUOWStore{predRepo: invalidItemRepo, insRepo: newMockInspectionRepo()},
	}
	invalidML := &mockMLClient{
		items: []domain.PredictionItem{
			{StationCode: "FAL", Platform: "INVALID_PLATFORM", DoorNumber: 1, WillFail: true},
		},
	}
	uc5 := usecase.NewTriggerPredictionsUseCase(invalidItemRepo, invalidML, nil, uowNormal)
	if _, err := uc5.Execute(context.Background(), 7); err == nil {
		t.Fatalf("expected error on invalid inspection creation")
	}

	uc6 := usecase.NewTriggerPredictionsUseCase(invalidItemRepo, &mockMLClient{
		items: []domain.PredictionItem{
			{StationCode: "FAL", Platform: "P1", DoorNumber: 1, WillFail: true},
		},
	}, nil, &mockUnitOfWork{
		store: &mockUOWStore{predRepo: invalidItemRepo, insRepo: &failingInspectionRepo{InspectionRepository: newMockInspectionRepo()}},
	})
	if _, err := uc6.Execute(context.Background(), 7); err == nil {
		t.Fatalf("expected error on inspection repository failure")
	}

	ucNilBus := usecase.NewTriggerPredictionsUseCase(invalidItemRepo, &mockMLClient{}, nil, uowNormal)
	if _, err := ucNilBus.Execute(context.Background(), 7); err != nil {
		t.Fatalf("expected success with nil event bus, got %v", err)
	}
}

func TestDashboardUseCase_TriggerPredictions_FixedHorizon(t *testing.T) {
	doors := []domain.Door{
		{ID: 1, StationID: 1, StationCode: "FAL", Platform: "P1", DoorNumber: 1, Status: "active"},
	}
	predRepo := &mockPredictionRepo{doors: doors}
	dashRepo := &mockDashboardRepo{}
	ml := &mockMLClient{}
	uow := &mockUnitOfWork{
		store: &mockUOWStore{predRepo: predRepo, insRepo: newMockInspectionRepo()},
	}

	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml, nil, uow)
	run, err := uc.TriggerPredictions(context.Background(), 14)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if run == nil || run.HorizonDays != 7 {
		t.Fatalf("expected fixed horizon 7, got %+v", run)
	}
	if run.ThresholdUsed == nil || *run.ThresholdUsed != 0.5 {
		t.Fatalf("expected fallback threshold 0.5 without MLOps repository, got %v", run.ThresholdUsed)
	}
}

func TestDashboardUseCase_PredictionExperimentIsIsolatedUntilPromotion(t *testing.T) {
	doors := []domain.Door{
		{ID: 7, StationID: 1, StationCode: "FAL", Platform: "P1", DoorNumber: 7, Status: "active"},
	}
	predRepo := &mockPredictionRepo{doors: doors}
	insRepo := newMockInspectionRepo()
	bus := &mockEventBus{}
	uow := &mockUnitOfWork{store: &mockUOWStore{predRepo: predRepo, insRepo: insRepo}}
	baseProbability := 0.4
	ml := &mockMLClient{items: []domain.PredictionItem{{
		StationCode: "FAL", Platform: "P1", DoorNumber: 7, Probability: &baseProbability,
	}}}
	uc := usecase.NewDashboardUseCase(&mockDashboardRepo{}, predRepo, ml, bus, uow)

	run, err := uc.RunPredictionExperiment(context.Background(), 0.35)
	if err != nil {
		t.Fatalf("expected experiment success, got %v", err)
	}
	if !run.IsExperiment || run.PublishedAt != nil || run.HorizonDays != 7 {
		t.Fatalf("experiment metadata is invalid: %+v", run)
	}
	if run.ThresholdUsed == nil || *run.ThresholdUsed != 0.35 {
		t.Fatalf("expected threshold 0.35, got %v", run.ThresholdUsed)
	}
	if run.AverageProbability == nil || math.Abs(*run.AverageProbability-0.4) > 0.000001 {
		t.Fatalf("expected original probability 0.4, got %v", run.AverageProbability)
	}
	if run.EstimatedPrecision != nil || run.EstimatedRecall != nil || run.EstimatedF1 != nil {
		t.Fatalf("metrics without model provenance must stay unavailable: precision=%v recall=%v", run.EstimatedPrecision, run.EstimatedRecall)
	}
	inspections, _ := insRepo.List(context.Background(), 10, 0, "")
	if len(inspections) != 0 {
		t.Fatalf("experiment must not create inspections, got %d", len(inspections))
	}
	if len(bus.published) != 0 {
		t.Fatalf("experiment must not publish operational events, got %d", len(bus.published))
	}

	experiments, err := uc.ListPredictionExperiments(context.Background(), 20)
	if err != nil || len(experiments) != 1 {
		t.Fatalf("expected saved experiment, got %+v, err=%v", experiments, err)
	}

	promoted, err := uc.PromotePredictionExperiment(context.Background(), run.ID, "engineer-1")
	if err != nil {
		t.Fatalf("expected promotion success, got %v", err)
	}
	if promoted.IsExperiment || promoted.PublishedAt == nil {
		t.Fatalf("promoted run must be official: %+v", promoted)
	}
	inspections, _ = insRepo.List(context.Background(), 10, 0, "")
	if len(inspections) != 1 {
		t.Fatalf("promotion must create one inspection, got %d", len(inspections))
	}
	if len(bus.published) < 2 {
		t.Fatalf("promotion must publish risk and completion events, got %d", len(bus.published))
	}
}

func TestDashboardUseCase_PredictionExperimentRejectsInvalidThreshold(t *testing.T) {
	predRepo := &mockPredictionRepo{}
	uc := usecase.NewDashboardUseCase(&mockDashboardRepo{}, predRepo, &mockMLClient{}, nil, &mockUnitOfWork{})
	if _, err := uc.RunPredictionExperiment(context.Background(), 0.01); !errors.Is(err, domain.ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}
