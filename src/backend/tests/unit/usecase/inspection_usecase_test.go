package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
)

type mockInspectionRepo struct {
	mu          sync.Mutex
	inspections map[int64]*domain.Inspection
	nextID      int64
}

func newMockInspectionRepo() *mockInspectionRepo {
	return &mockInspectionRepo{
		inspections: make(map[int64]*domain.Inspection),
		nextID:      1,
	}
}

func (m *mockInspectionRepo) Create(ctx context.Context, ins *domain.Inspection) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ins.ID = m.nextID
	ins.CreatedAt = time.Now()
	ins.UpdatedAt = time.Now()
	m.nextID++
	m.inspections[ins.ID] = ins
	return nil
}

func (m *mockInspectionRepo) UpdateStatus(ctx context.Context, id int64, status, notes string, completedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ins, ok := m.inspections[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	ins.Status = status
	if notes != "" {
		ins.Notes = notes
	}
	ins.CompletedAt = completedAt
	ins.UpdatedAt = time.Now()
	return nil
}

func (m *mockInspectionRepo) GetByID(ctx context.Context, id int64) (*domain.Inspection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ins, ok := m.inspections[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return ins, nil
}

func (m *mockInspectionRepo) List(ctx context.Context, limit, offset int, statusFilter string) ([]*domain.Inspection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*domain.Inspection
	for _, ins := range m.inspections {
		if statusFilter == "" || ins.Status == statusFilter {
			list = append(list, ins)
		}
	}
	return list, nil
}

func TestInspectionUseCase_CreateSuccess(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	input := usecase.CreateInspectionInputDTO{
		StationCode:  "FAL",
		Platform:     "P1",
		DoorNumber:   4,
		Priority:     "critical",
		Notes:        "Alerta de falha gerado pela IA",
		ScheduledFor: "2026-09-03 10:00:00",
	}

	dto, err := uc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("esperava sucesso, obteve: %v", err)
	}

	if dto.ID == 0 {
		t.Errorf("id deveria ser preenchido")
	}

	if dto.Status != "pending" {
		t.Errorf("esperava pending, obteve %s", dto.Status)
	}

	if dto.Priority != "critical" {
		t.Errorf("esperava critical, obteve %s", dto.Priority)
	}
}

func TestInspectionUseCase_InvalidDoor(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	input := usecase.CreateInspectionInputDTO{
		StationCode: "FAL",
		DoorNumber:  25,
	}

	_, err := uc.Create(context.Background(), input)
	if err == nil {
		t.Fatalf("esperava erro para porta > 24")
	}
}

func TestInspectionUseCase_UpdateStatusCompleted(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	created, err := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "PIN",
		Platform:    "P2",
		DoorNumber:  12,
	})
	if err != nil {
		t.Fatalf("falha ao criar inspecao: %v", err)
	}

	updateInput := usecase.UpdateInspectionStatusInputDTO{
		Status: "completed",
		Notes:  "Sensor calibrado com sucesso pelo tecnico Ronaldo",
	}

	updated, err := uc.UpdateStatus(context.Background(), created.ID, updateInput)
	if err != nil {
		t.Fatalf("falha ao atualizar status: %v", err)
	}

	if updated.Status != "completed" {
		t.Errorf("esperava completed, obteve %s", updated.Status)
	}

	if updated.CompletedAt == nil {
		t.Errorf("completed_at deveria estar preenchido")
	}
}

func TestInspectionUseCase_InvalidStatus(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	created, _ := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "PIN",
		Platform:    "P2",
		DoorNumber:  12,
	})

	_, err := uc.UpdateStatus(context.Background(), created.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "status_inexistente",
	})
	if err == nil {
		t.Fatalf("esperava erro para status invalido")
	}
}
