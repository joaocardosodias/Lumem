package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type mockUnitOfWorkStore struct {
	insRepo port.InspectionRepository
}

func (m *mockUnitOfWorkStore) Predictions() port.PredictionRepository {
	return nil
}

func (m *mockUnitOfWorkStore) Inspections() port.InspectionRepository {
	return m.insRepo
}

func (m *mockUnitOfWorkStore) Maintenances() port.MaintenanceRepository {
	return nil
}

func (m *mockUnitOfWorkStore) Alarms() port.AlarmRepository {
	return nil
}

func (m *mockUnitOfWorkStore) Uploads() port.UploadRepository {
	return nil
}

type mockUOW struct {
	store      *mockUnitOfWorkStore
	committed  bool
	rolledBack bool
}

func (m *mockUOW) Do(ctx context.Context, fn func(store port.UnitOfWorkStore) error) error {
	if err := fn(m.store); err != nil {
		m.rolledBack = true
		return err
	}
	m.committed = true
	return nil
}

func TestUnitOfWork_CommitOnSuccess(t *testing.T) {
	insRepo := newMockInspectionRepo()
	store := &mockUnitOfWorkStore{insRepo: insRepo}
	uow := &mockUOW{store: store}

	err := uow.Do(context.Background(), func(s port.UnitOfWorkStore) error {
		ins := &domain.Inspection{
			StationCode: "FAL",
			Platform:    "P1",
			DoorNumber:  4,
			Status:      "pending",
			Priority:    "critical",
		}
		return s.Inspections().Create(context.Background(), ins)
	})

	if err != nil {
		t.Fatalf("esperava sucesso na transacao: %v", err)
	}

	if !uow.committed {
		t.Errorf("esperava transacao confirmada (committed = true)")
	}

	inspections, _ := insRepo.List(context.Background(), 10, 0, "")
	if len(inspections) != 1 {
		t.Fatalf("esperava 1 inspecao criada, obteve %d", len(inspections))
	}
}

func TestUnitOfWork_RollbackOnError(t *testing.T) {
	insRepo := newMockInspectionRepo()
	store := &mockUnitOfWorkStore{insRepo: insRepo}
	uow := &mockUOW{store: store}
	expectedErr := errors.New("falha simulada")

	err := uow.Do(context.Background(), func(s port.UnitOfWorkStore) error {
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("esperava erro %v, obteve %v", expectedErr, err)
	}

	if !uow.rolledBack {
		t.Errorf("esperava transacao desfeita (rolledBack = true)")
	}
}
