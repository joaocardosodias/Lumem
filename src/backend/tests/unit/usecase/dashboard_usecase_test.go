package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

type mockDashboardRepo struct {
	summary  *domain.DashboardSummary
	timeline *domain.DoorTimeline
	status   []*domain.DoorStatusItem
}

func (m *mockDashboardRepo) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	return m.summary, nil
}

func (m *mockDashboardRepo) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	return m.timeline, nil
}

func (m *mockDashboardRepo) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	return m.status, nil
}

type mockPredictionRepo struct {
	latestRun *domain.PredictionRun
	doors     []domain.Door
}

func (m *mockPredictionRepo) SaveRun(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	run.ID = "run-test-123"
	m.latestRun = run
	return nil
}

func (m *mockPredictionRepo) GetLatestRun(ctx context.Context) (*domain.PredictionRun, error) {
	return m.latestRun, nil
}

func (m *mockPredictionRepo) ListDoorsForInference(ctx context.Context) ([]domain.Door, error) {
	return m.doors, nil
}

type mockMLClient struct {
	items []domain.PredictionItem
}

func (m *mockMLClient) PredictBatch(ctx context.Context, doors []domain.Door) ([]domain.PredictionItem, error) {
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
	dashRepo := &mockDashboardRepo{
		timeline: &domain.DoorTimeline{
			Door: domain.Door{
				ID:         4,
				Platform:   "P1",
				DoorNumber: 4,
				SAPTag:     "SP04-6-PSD-FAL-PLT1-PP04",
			},
			StationName: "Faria Lima",
			WillFail:    true,
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
