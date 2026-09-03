package usecase_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
)

type mockMaintRepo struct {
	mu     sync.Mutex
	events []*domain.MaintenanceEvent
}

func (m *mockMaintRepo) BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, events...)
	return nil
}

func TestIngestMaintenanceUseCase_ValidCSV(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	uploadRepo := newMockAlarmRepo()
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo)

	csvData := `ordem,tipo_de_nota,local_de_instalacao,inicio_avaria_datetime,fim_avaria_datetime,sintoma
53954586,ZM01,SP04-6-PSD-BUT,2025-11-14 06:35:28,2025-11-14 08:00:00,LAÇO DE PDM ABERTO
53954587,ZM02,SP04-6-PSD-FAL,2025-11-15 10:00:00,2025-11-15 12:00:00,FALHA DE COMUNICACAO
`
	uploadID, err := uc.StartIngestion(context.Background(), "zm01_test.csv", strings.NewReader(csvData), "ZM01")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	upload, err := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status do upload: %v", err)
	}

	if upload.Status != "completed" {
		t.Errorf("esperava status completed, obteve %s", upload.Status)
	}

	if upload.RowsInserted != 2 {
		t.Errorf("esperava 2 linhas inseridas, obteve %d", upload.RowsInserted)
	}

	if len(maintRepo.events) != 2 {
		t.Errorf("esperava 2 eventos no repositorio, obteve %d", len(maintRepo.events))
	}
}

func TestIngestMaintenanceUseCase_MissingDate(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	uploadRepo := newMockAlarmRepo()
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo)

	csvData := `ordem,tipo_de_nota,sintoma
53954586,ZM01,SEM DATA DE INICIO
`
	uploadID, err := uc.StartIngestion(context.Background(), "zm01_invalido.csv", strings.NewReader(csvData), "ZM01")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	upload, err := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status do upload: %v", err)
	}

	if upload.Status != "failed" {
		t.Errorf("esperava status failed, obteve %s", upload.Status)
	}
}
