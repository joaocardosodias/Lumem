package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/google/uuid"
)

type mockInspectionRepo struct {
	mu          sync.Mutex
	inspections map[string]*domain.Inspection
}

func newMockInspectionRepo() *mockInspectionRepo {
	return &mockInspectionRepo{
		inspections: make(map[string]*domain.Inspection),
	}
}

func (m *mockInspectionRepo) Create(ctx context.Context, ins *domain.Inspection) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ins.ID == "" {
		ins.ID = uuid.New().String()
	}
	m.inspections[ins.ID] = ins
	return nil
}

func (m *mockInspectionRepo) UpdateStatus(ctx context.Context, id string, status, notes string, completedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ins, exists := m.inspections[id]
	if !exists {
		return domain.ErrUserNotFound
	}
	ins.Status = status
	ins.Notes = notes
	ins.CompletedAt = completedAt
	return nil
}

func (m *mockInspectionRepo) GetByID(ctx context.Context, id string) (*domain.Inspection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ins, exists := m.inspections[id]
	if !exists {
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

func TestInspectionUseCase_Create(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	input := usecase.CreateInspectionInputDTO{
		StationCode: "FAL",
		Platform:    "P1",
		DoorNumber:  4,
		Priority:    "critical",
		Notes:       "Ruido excessivo ao abrir",
	}

	dto, err := uc.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("esperava sucesso na criacao de inspecao: %v", err)
	}

	if dto.ID == "" {
		t.Errorf("id gerado deveria ser um UUID preenchido")
	}
	if dto.Status != "pending" {
		t.Errorf("status inicial deveria ser pending, obteve %s", dto.Status)
	}
}

func TestInspectionUseCase_UpdateStatus(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	created, _ := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "PIN",
		Platform:    "P2",
		DoorNumber:  8,
	})

	updateInput := usecase.UpdateInspectionStatusInputDTO{
		Status: "completed",
		Notes:  "Sensor calibrado com sucesso",
	}

	updated, err := uc.UpdateStatus(context.Background(), created.ID, updateInput)
	if err != nil {
		t.Fatalf("esperava sucesso na atualizacao: %v", err)
	}

	if updated.Status != "completed" {
		t.Errorf("esperava status completed, obteve %s", updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Errorf("completed_at deveria estar preenchido apos conclusao")
	}
}
