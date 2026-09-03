package usecase_test

import (
	"context"
	"testing"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
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
	run.ID = 1
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
			StationCode: d.StationCode,
			Platform:    d.Platform,
			DoorNumber:  d.DoorNumber,
			WillFail:    d.DoorNumber%7 == 0,
		}
	}
	return items, nil
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
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml)

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
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml)

	res, err := uc.GetDoorTimeline(context.Background(), 4)
	if err != nil {
		t.Fatalf("esperava sucesso no timeline: %v", err)
	}

	if !res.WillFail {
		t.Errorf("esperava WillFail = true")
	}

	_, err = uc.GetDoorTimeline(context.Background(), 0)
	if err == nil {
		t.Fatalf("esperava erro para id de porta 0")
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
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml)

	items, err := uc.GetStationDoorsStatus(context.Background(), "FAL")
	if err != nil {
		t.Fatalf("esperava sucesso no status da estacao: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("esperava 2 itens, obteve %d", len(items))
	}

	_, err = uc.GetStationDoorsStatus(context.Background(), "")
	if err == nil {
		t.Fatalf("esperava erro para codigo vazio")
	}
}

func TestDashboardUseCase_TriggerPredictions(t *testing.T) {
	dashRepo := &mockDashboardRepo{}
	predRepo := &mockPredictionRepo{
		doors: []domain.Door{
			{ID: 1, Platform: "P1", DoorNumber: 1},
			{ID: 7, Platform: "P1", DoorNumber: 7},
		},
	}
	ml := &mockMLClient{}
	uc := usecase.NewDashboardUseCase(dashRepo, predRepo, ml)

	run, err := uc.TriggerPredictions(context.Background(), 7)
	if err != nil {
		t.Fatalf("esperava sucesso na inferencia: %v", err)
	}

	if run.TotalDoorsEvaluated != 2 {
		t.Errorf("esperava 2 portas avaliadas, obteve %d", run.TotalDoorsEvaluated)
	}

	if run.CreatedAt == "" {
		t.Errorf("created_at deveria estar preenchido")
	}
}
