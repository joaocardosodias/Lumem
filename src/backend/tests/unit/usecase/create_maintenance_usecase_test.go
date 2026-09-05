package usecase_test

import (
	"context"
	"testing"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/google/uuid"
)

func (m *mockMaintRepo) Create(ctx context.Context, event *domain.MaintenanceEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	m.events = append(m.events, event)
	return nil
}

func (m *mockMaintRepo) List(ctx context.Context, limit, offset int) ([]*domain.MaintenanceEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.events, nil
}

func TestCreateMaintenanceUseCase_Success(t *testing.T) {
	repo := &mockMaintRepo{}
	uc := usecase.NewCreateMaintenanceUseCase(repo)

	input := usecase.CreateMaintenanceInputDTO{
		OrderType:            "ZM01",
		StationCode:          "FAL",
		InstallationLocation: "SP04-6-PSD-FAL-PLT1-PP04",
		StartTimestamp:       "2026-09-02 14:30:00",
		Description:          "Falha no sensor de presenca da porta",
		Symptom:              "Porta travada aberta",
		Cause:                "Sujeira no trilho",
		ActionTaken:          "Limpeza e recalibracao",
	}

	dto, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("esperava sucesso na criacao manual, obteve: %v", err)
	}

	if dto.ID == "" {
		t.Errorf("id deveria ser preenchido")
	}

	if dto.Source != "manual_form" {
		t.Errorf("esperava source manual_form, obteve %s", dto.Source)
	}

	if dto.StationCode != "FAL" {
		t.Errorf("esperava FAL, obteve %s", dto.StationCode)
	}
}

func TestCreateMaintenanceUseCase_MissingOrderType(t *testing.T) {
	repo := &mockMaintRepo{}
	uc := usecase.NewCreateMaintenanceUseCase(repo)

	input := usecase.CreateMaintenanceInputDTO{
		OrderType:   "",
		StationCode: "FAL",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatalf("esperava erro para order_type vazio")
	}
}

func TestCreateMaintenanceUseCase_MissingStationCode(t *testing.T) {
	repo := &mockMaintRepo{}
	uc := usecase.NewCreateMaintenanceUseCase(repo)

	input := usecase.CreateMaintenanceInputDTO{
		OrderType:   "ZM02",
		StationCode: "",
	}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatalf("esperava erro para station_code vazio")
	}
}

type errMaintRepo struct {
	mockMaintRepo
	createErr error
	listErr   error
}

func (r *errMaintRepo) Create(ctx context.Context, event *domain.MaintenanceEvent) error {
	if r.createErr != nil {
		return r.createErr
	}
	return r.mockMaintRepo.Create(ctx, event)
}

func (r *errMaintRepo) List(ctx context.Context, limit, offset int) ([]*domain.MaintenanceEvent, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.mockMaintRepo.List(ctx, limit, offset)
}

func TestCreateMaintenanceUseCase_RepoCreateError(t *testing.T) {
	repo := &errMaintRepo{createErr: domain.ErrMaintenanceNotFound}
	uc := usecase.NewCreateMaintenanceUseCase(repo)

	input := usecase.CreateMaintenanceInputDTO{
		OrderType:      "ZM01",
		StationCode:    "FAL",
		StartTimestamp: "2026-09-02 14:30:00",
	}

	_, err := uc.Execute(context.Background(), input)
	if err != domain.ErrMaintenanceNotFound {
		t.Fatalf("expected ErrMaintenanceNotFound, got %v", err)
	}
}

func TestCreateMaintenanceUseCase_List(t *testing.T) {
	repo := &mockMaintRepo{}
	uc := usecase.NewCreateMaintenanceUseCase(repo)

	_, _ = uc.Execute(context.Background(), usecase.CreateMaintenanceInputDTO{
		OrderType:      "ZM01",
		StationCode:    "FAL",
		StartTimestamp: "2026-09-02 14:30:00",
	})

	list, err := uc.List(context.Background(), 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 maintenance event in list, got %v", err)
	}

	errRepo := &errMaintRepo{listErr: domain.ErrMaintenanceNotFound}
	errUc := usecase.NewCreateMaintenanceUseCase(errRepo)
	_, err = errUc.List(context.Background(), 10, 0)
	if err != domain.ErrMaintenanceNotFound {
		t.Fatalf("expected list error, got %v", err)
	}
}
