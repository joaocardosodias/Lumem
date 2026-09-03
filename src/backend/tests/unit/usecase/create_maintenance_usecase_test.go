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
