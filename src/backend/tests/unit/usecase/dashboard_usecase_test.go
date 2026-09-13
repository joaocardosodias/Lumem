package usecase_test

import (
	"context"
	"errors"
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
	doors      []domain.Door
	runErr     error
	doorsErr   error
	saveRunErr error
}

func (m *mockPredictionRepo) SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	if m.saveRunErr != nil {
		return m.saveRunErr
	}
	run.ID = "run-test-123"
	m.latestRun = run
	return nil
}

func (m *mockPredictionRepo) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.latestRun, nil
}

func (m *mockPredictionRepo) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	if m.doorsErr != nil {
		return nil, m.doorsErr
	}
	return m.doors, nil
}

type mockMLClient struct {
	items         []domain.PredictionItem
	err           error
	lastThreshold *float64
}

func (m *mockMLClient) PredictBatch(ctx context.Context, doors []domain.Door, threshold *float64) ([]domain.PredictionItem, error) {
	m.lastThreshold = threshold
	if m.err != nil {
		return nil, m.err
	}
	if m.items != nil {
		return m.items, nil
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
	return items, nil
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

func TestDashboardUseCase_TriggerPredictions_CustomThreshold(t *testing.T) {
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
	thresholdVal := 0.35
	run, err := uc.TriggerPredictionsWithThreshold(context.Background(), 14, &thresholdVal)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if run == nil || run.HorizonDays != 14 {
		t.Fatalf("expected horizon 14, got %+v", run)
	}
	if ml.lastThreshold == nil || *ml.lastThreshold != 0.35 {
		t.Fatalf("expected threshold 0.35, got %v", ml.lastThreshold)
	}
}

