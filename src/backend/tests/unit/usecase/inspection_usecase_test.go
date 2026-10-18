package usecase_test

import (
	"context"
	"errors"
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
		return domain.ErrInspectionNotFound
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
		return nil, domain.ErrInspectionNotFound
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

func TestInspectionUseCase_ValidationErrors(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	cases := []struct {
		name        string
		input       usecase.CreateInspectionInputDTO
		expectedErr error
	}{
		{
			name:        "missing station code",
			input:       usecase.CreateInspectionInputDTO{StationCode: "", Platform: "P1", DoorNumber: 1},
			expectedErr: domain.ErrStationCodeRequired,
		},
		{
			name:        "invalid platform",
			input:       usecase.CreateInspectionInputDTO{StationCode: "FAL", Platform: "P3", DoorNumber: 1},
			expectedErr: domain.ErrInvalidPlatform,
		},
		{
			name:        "invalid door number below range",
			input:       usecase.CreateInspectionInputDTO{StationCode: "FAL", Platform: "P1", DoorNumber: 0},
			expectedErr: domain.ErrInvalidDoorNumber,
		},
		{
			name:        "invalid door number above range",
			input:       usecase.CreateInspectionInputDTO{StationCode: "FAL", Platform: "P1", DoorNumber: 25},
			expectedErr: domain.ErrInvalidDoorNumber,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.Create(context.Background(), tc.input)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("esperava erro %v, obteve %v", tc.expectedErr, err)
			}
		})
	}
}

func TestInspectionUseCase_UpdateStatus_InvalidStatus(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	created, err := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "PIN",
		Platform:    "P1",
		DoorNumber:  2,
	})
	if err != nil {
		t.Fatalf("falha ao criar inspecao: %v", err)
	}

	_, err = uc.UpdateStatus(context.Background(), created.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "invalid_status",
	})
	if !errors.Is(err, domain.ErrInvalidInspectionStatus) {
		t.Fatalf("esperava ErrInvalidInspectionStatus, obteve %v", err)
	}
}

func TestInspectionUseCase_GetByID_NotFound(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	_, err := uc.GetByID(context.Background(), "non-existent-id")
	if !errors.Is(err, domain.ErrInspectionNotFound) {
		t.Fatalf("esperava ErrInspectionNotFound, obteve %v", err)
	}
}

func TestInspectionUseCase_UpdateStatus_ReopenFromCompletedClearsCompletedAt(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	created, err := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "PIN",
		Platform:    "P1",
		DoorNumber:  2,
	})
	if err != nil {
		t.Fatalf("falha ao criar inspecao: %v", err)
	}

	completed, err := uc.UpdateStatus(context.Background(), created.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "completed",
		Notes:  "Concluida primeira vez",
	})
	if err != nil {
		t.Fatalf("falha ao completar inspecao: %v", err)
	}
	if completed.CompletedAt == nil {
		t.Fatalf("esperava completed_at preenchido")
	}

	reopened, err := uc.UpdateStatus(context.Background(), created.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "in_progress",
		Notes:  "Reaberta para calibracao adicional",
	})
	if err != nil {
		t.Fatalf("falha ao reabrir inspecao: %v", err)
	}
	if reopened.Status != "in_progress" {
		t.Errorf("esperava status in_progress, obteve %s", reopened.Status)
	}
	if reopened.CompletedAt != nil {
		t.Errorf("esperava completed_at nulo apos reabrir inspecao, obteve %s", *reopened.CompletedAt)
	}

	fetched, err := uc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("falha ao buscar inspecao: %v", err)
	}
	if fetched.CompletedAt != nil {
		t.Errorf("esperava completed_at nulo no repositorio/DTO, obteve %s", *fetched.CompletedAt)
	}
}

func TestInspectionUseCase_UpdateStatus_InvalidTransitions(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)

	created, err := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "PIN",
		Platform:    "P1",
		DoorNumber:  2,
	})
	if err != nil {
		t.Fatalf("falha ao criar inspecao: %v", err)
	}

	_, err = uc.UpdateStatus(context.Background(), created.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "completed",
	})
	if err != nil {
		t.Fatalf("falha ao completar: %v", err)
	}

	_, err = uc.UpdateStatus(context.Background(), created.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "canceled",
	})
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("esperava ErrInvalidStatusTransition ao tentar cancelar inspecao completed, obteve %v", err)
	}

	canceledIns, err := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "FAL",
		Platform:    "P2",
		DoorNumber:  5,
	})
	if err != nil {
		t.Fatalf("falha ao criar inspecao: %v", err)
	}

	_, err = uc.UpdateStatus(context.Background(), canceledIns.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "canceled",
	})
	if err != nil {
		t.Fatalf("falha ao cancelar inspecao: %v", err)
	}

	_, err = uc.UpdateStatus(context.Background(), canceledIns.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "in_progress",
	})
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("esperava ErrInvalidStatusTransition ao tentar reabrir inspecao canceled, obteve %v", err)
	}

	_, err = uc.UpdateStatus(context.Background(), canceledIns.ID, usecase.UpdateInspectionStatusInputDTO{
		Status: "completed",
	})
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Fatalf("esperava ErrInvalidStatusTransition ao tentar completar inspecao canceled, obteve %v", err)
	}
}

type errInspectionRepo struct {
	*mockInspectionRepo
	createErr       error
	getErr          error
	updateStatusErr error
	listErr         error
}

func (r *errInspectionRepo) Create(ctx context.Context, ins *domain.Inspection) error {
	if r.createErr != nil {
		return r.createErr
	}
	if r.mockInspectionRepo != nil {
		return r.mockInspectionRepo.Create(ctx, ins)
	}
	return nil
}

func (r *errInspectionRepo) GetByID(ctx context.Context, id string) (*domain.Inspection, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.mockInspectionRepo != nil {
		return r.mockInspectionRepo.GetByID(ctx, id)
	}
	return nil, nil
}

func (r *errInspectionRepo) UpdateStatus(ctx context.Context, id string, status, notes string, completedAt *time.Time) error {
	if r.updateStatusErr != nil {
		return r.updateStatusErr
	}
	if r.mockInspectionRepo != nil {
		return r.mockInspectionRepo.UpdateStatus(ctx, id, status, notes, completedAt)
	}
	return nil
}

func (r *errInspectionRepo) List(ctx context.Context, limit, offset int, statusFilter string) ([]*domain.Inspection, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	if r.mockInspectionRepo != nil {
		return r.mockInspectionRepo.List(ctx, limit, offset, statusFilter)
	}
	return nil, nil
}

func TestInspectionUseCase_ErrorPaths(t *testing.T) {
	errRepo := &errInspectionRepo{mockInspectionRepo: newMockInspectionRepo(), createErr: errors.New("db create error")}
	uc := usecase.NewInspectionUseCase(errRepo)

	_, err := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "FAL",
		Platform:    "P1",
		DoorNumber:  1,
	})
	if err == nil || err.Error() != "db create error" {
		t.Fatalf("expected db create error")
	}

	_, err = uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "",
	})
	if !errors.Is(err, domain.ErrStationCodeRequired) {
		t.Fatalf("expected ErrStationCodeRequired")
	}

	if _, err := uc.UpdateStatus(context.Background(), "  ", usecase.UpdateInspectionStatusInputDTO{}); !errors.Is(err, domain.ErrInspectionIDRequired) {
		t.Fatalf("expected ErrInspectionIDRequired on empty id")
	}

	errRepo.getErr = domain.ErrInspectionNotFound
	if _, err := uc.UpdateStatus(context.Background(), "some-id", usecase.UpdateInspectionStatusInputDTO{}); !errors.Is(err, domain.ErrInspectionNotFound) {
		t.Fatalf("expected ErrInspectionNotFound")
	}

	if _, err := uc.GetByID(context.Background(), ""); !errors.Is(err, domain.ErrInspectionIDRequired) {
		t.Fatalf("expected ErrInspectionIDRequired on GetByID with empty id")
	}

	if _, err := uc.GetByID(context.Background(), "some-id"); !errors.Is(err, domain.ErrInspectionNotFound) {
		t.Fatalf("expected ErrInspectionNotFound on GetByID")
	}

	errRepo.listErr = errors.New("list failed")
	if _, err := uc.List(context.Background(), 10, 0, ""); err == nil {
		t.Fatalf("expected list error")
	}

	validRepo := newMockInspectionRepo()
	validUC := usecase.NewInspectionUseCase(validRepo)
	created, _ := validUC.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "FAL",
		Platform:    "P1",
		DoorNumber:  1,
	})

	failingUpdateRepo := &errInspectionRepo{
		mockInspectionRepo: validRepo,
		updateStatusErr:    errors.New("update status error"),
	}
	failingUpdateUC := usecase.NewInspectionUseCase(failingUpdateRepo)
	if _, err := failingUpdateUC.UpdateStatus(context.Background(), created.ID, usecase.UpdateInspectionStatusInputDTO{Status: "in_progress"}); err == nil {
		t.Fatalf("expected update status error")
	}
}

func TestInspectionUseCase_List_Success(t *testing.T) {
	repo := newMockInspectionRepo()
	uc := usecase.NewInspectionUseCase(repo)
	_, err := uc.Create(context.Background(), usecase.CreateInspectionInputDTO{
		StationCode: "PIN",
		Platform:    "P1",
		DoorNumber:  2,
	})
	if err != nil {
		t.Fatalf("falha ao criar: %v", err)
	}

	list, err := uc.List(context.Background(), 10, 0, "")
	if err != nil {
		t.Fatalf("falha ao listar: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("esperava 1 item na lista, obteve %d", len(list))
	}
}
