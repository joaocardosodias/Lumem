package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/g04/backend/internal/adapter/parser"
	"github.com/g04/backend/internal/adapter/scheduler"
	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/xuri/excelize/v2"
)

type mockMaintRepo struct {
	mu              sync.Mutex
	events          []*domain.MaintenanceEvent
	failBatchInsert bool
}

func (m *mockMaintRepo) BatchInsertMaintenance(ctx context.Context, events []*domain.MaintenanceEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failBatchInsert {
		return errors.New("falha simulada no banco ao inserir manutencao")
	}
	m.events = append(m.events, events...)
	return nil
}

func (m *mockMaintRepo) DeleteByUploadID(ctx context.Context, uploadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	filtered := m.events[:0]
	for _, e := range m.events {
		if e.UploadID != uploadID {
			filtered = append(filtered, e)
		}
	}
	m.events = filtered
	return nil
}

type mockMaintStore struct {
	maintRepo  *mockMaintRepo
	uploadRepo *mockAlarmRepo
}

func (s *mockMaintStore) Predictions() port.PredictionRepository   { return nil }
func (s *mockMaintStore) Inspections() port.InspectionRepository   { return nil }
func (s *mockMaintStore) Maintenances() port.MaintenanceRepository { return s.maintRepo }
func (s *mockMaintStore) Alarms() port.AlarmRepository             { return nil }
func (s *mockMaintStore) Uploads() port.UploadRepository           { return s.uploadRepo }

type mockMaintUOW struct {
	store *mockMaintStore
}

func (u *mockMaintUOW) Do(ctx context.Context, fn func(store port.UnitOfWorkStore) error) error {
	u.store.maintRepo.mu.Lock()
	savedEvents := make([]*domain.MaintenanceEvent, len(u.store.maintRepo.events))
	copy(savedEvents, u.store.maintRepo.events)
	u.store.maintRepo.mu.Unlock()

	if err := fn(u.store); err != nil {
		u.store.maintRepo.mu.Lock()
		u.store.maintRepo.events = savedEvents
		u.store.maintRepo.mu.Unlock()
		return err
	}
	return nil
}

func TestIngestMaintenanceUseCase_ValidCSV(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	uploadRepo := newMockAlarmRepo()
	maintStorage := storage.NewLocalTempStorage()
	maintParser := parser.NewDefaultMaintenanceParser()
	runner := scheduler.NewSyncJobRunner()
	uow := &mockMaintUOW{store: &mockMaintStore{maintRepo: maintRepo, uploadRepo: uploadRepo}}
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, maintStorage, maintParser, runner, uow, nil)

	csvData := `ordem,tipo_de_nota,local_de_instalacao,inicio_avaria_datetime,fim_avaria_datetime,sintoma
53954586,ZM01,SP04-6-PSD-BUT,2025-11-14 06:35:28,2025-11-14 08:00:00,LAÇO DE PDM ABERTO
53954587,ZM02,SP04-6-PSD-FAL,2025-11-15 10:00:00,2025-11-15 12:00:00,FALHA DE COMUNICACAO
`
	uploadID, err := uc.StartIngestion(context.Background(), "zm01_test.csv", strings.NewReader(csvData), "ZM01")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

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
	for _, e := range maintRepo.events {
		if e.UploadID != uploadID {
			t.Errorf("esperava UploadID associado ao evento de manutencao, obteve %s", e.UploadID)
		}
	}
}

func TestIngestMaintenanceUseCase_ValidXLSX(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	uploadRepo := newMockAlarmRepo()
	maintStorage := storage.NewLocalTempStorage()
	maintParser := parser.NewDefaultMaintenanceParser()
	runner := scheduler.NewSyncJobRunner()
	uow := &mockMaintUOW{store: &mockMaintStore{maintRepo: maintRepo, uploadRepo: uploadRepo}}
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, maintStorage, maintParser, runner, uow, nil)

	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	_ = f.SetSheetRow(sheet, "A1", &[]any{"ordem", "tipo_de_ordem", "estacao", "inicio_avaria_datetime", "fim_avaria_datetime", "sintoma"})
	_ = f.SetSheetRow(sheet, "A2", &[]any{"1001", "ZM01", "PIN", "2026-09-01 10:00:00", "2026-09-01 11:00:00", "OBSTACULO"})
	_ = f.SetSheetRow(sheet, "A3", &[]any{"1002", "ZM01", "FAL", "2026-09-01 12:00:00", "2026-09-01 13:00:00", "TRAVA MECANICA"})

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("falha ao gerar xlsx de manutencao: %v", err)
	}

	uploadID, err := uc.StartIngestion(context.Background(), "manutencao.xlsx", &buf, "ZM01")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao xlsx: %v", err)
	}

	upload, err := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar upload: %v", err)
	}

	if upload.Status != "completed" {
		t.Errorf("esperava status completed para XLSX de manutencao, obteve %s (erro: %s)", upload.Status, upload.ErrorLog)
	}
	if upload.RowsInserted != 2 {
		t.Errorf("esperava 2 linhas inseridas, obteve %d", upload.RowsInserted)
	}
}

func TestIngestMaintenanceUseCase_ShortRowNoPanic(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	uploadRepo := newMockAlarmRepo()
	maintStorage := storage.NewLocalTempStorage()
	maintParser := parser.NewDefaultMaintenanceParser()
	runner := scheduler.NewSyncJobRunner()
	uow := &mockMaintUOW{store: &mockMaintStore{maintRepo: maintRepo, uploadRepo: uploadRepo}}
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, maintStorage, maintParser, runner, uow, nil)

	csvData := `ordem,tipo_de_nota,local_de_instalacao,inicio_avaria_datetime,fim_avaria_datetime,sintoma
53954586,ZM01
53954587,ZM02,SP04-6-PSD-FAL,2025-11-15 10:00:00,2025-11-15 12:00:00,FALHA DE COMUNICACAO
`
	uploadID, err := uc.StartIngestion(context.Background(), "curto.csv", strings.NewReader(csvData), "ZM01")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	upload, err := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar upload: %v", err)
	}

	if upload.Status != "completed" {
		t.Errorf("esperava status completed sem panico, obteve %s", upload.Status)
	}
	if upload.RowsInserted != 1 {
		t.Errorf("esperava 1 linha inserida, obteve %d", upload.RowsInserted)
	}
}

func TestIngestMaintenanceUseCase_RollbackOnBatchFailure(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	maintRepo.failBatchInsert = true
	uploadRepo := newMockAlarmRepo()
	maintStorage := storage.NewLocalTempStorage()
	maintParser := parser.NewDefaultMaintenanceParser()
	runner := scheduler.NewSyncJobRunner()
	uow := &mockMaintUOW{store: &mockMaintStore{maintRepo: maintRepo, uploadRepo: uploadRepo}}
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, maintStorage, maintParser, runner, uow, nil)

	csvData := `ordem,tipo_de_nota,local_de_instalacao,inicio_avaria_datetime,fim_avaria_datetime,sintoma
53954586,ZM01,SP04-6-PSD-BUT,2025-11-14 06:35:28,2025-11-14 08:00:00,LAÇO DE PDM ABERTO
`
	uploadID, err := uc.StartIngestion(context.Background(), "falha.csv", strings.NewReader(csvData), "ZM01")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	upload, err := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar upload: %v", err)
	}

	if upload.Status != "failed" {
		t.Errorf("esperava status failed apos erro de lote, obteve %s", upload.Status)
	}
	if len(maintRepo.events) != 0 {
		t.Errorf("esperava 0 eventos apos rollback da transacao, obteve %d", len(maintRepo.events))
	}
}

func TestIngestMaintenanceUseCase_FailUploadErrorLogged(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	maintRepo.failBatchInsert = true
	uploadRepo := newMockAlarmRepo()
	uploadRepo.failUpdateUpload = true
	logger := &mockLogger{}
	maintStorage := storage.NewLocalTempStorage()
	maintParser := parser.NewDefaultMaintenanceParser()
	runner := scheduler.NewSyncJobRunner()
	uow := &mockMaintUOW{store: &mockMaintStore{maintRepo: maintRepo, uploadRepo: uploadRepo}}
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, maintStorage, maintParser, runner, uow, logger)

	csvData := `ordem,tipo_de_nota,local_de_instalacao,inicio_avaria_datetime,fim_avaria_datetime,sintoma
53954586,ZM01,SP04-6-PSD-BUT,2025-11-14 06:35:28,2025-11-14 08:00:00,LAÇO DE PDM ABERTO
`
	_, err := uc.StartIngestion(context.Background(), "falha_log.csv", strings.NewReader(csvData), "ZM01")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	if len(logger.errors) == 0 {
		t.Fatalf("esperava que erro de falha ao marcar upload como failed fosse registrado no logger")
	}
}

func TestIngestMaintenanceUseCase_ConcurrentDuplicateUploads(t *testing.T) {
	maintRepo := &mockMaintRepo{}
	uploadRepo := newMockAlarmRepo()
	maintStorage := storage.NewLocalTempStorage()
	maintParser := parser.NewDefaultMaintenanceParser()
	runner := scheduler.NewSyncJobRunner()
	uow := &mockMaintUOW{store: &mockMaintStore{maintRepo: maintRepo, uploadRepo: uploadRepo}}
	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, maintStorage, maintParser, runner, uow, nil)

	csvData := `ordem,tipo_de_nota,local_de_instalacao,inicio_avaria_datetime,fim_avaria_datetime,sintoma
53954586,ZM01,SP04-6-PSD-BUT,2025-11-14 06:35:28,2025-11-14 08:00:00,LAÇO DE PDM ABERTO
`
	var wg sync.WaitGroup
	results := make([]struct {
		id  string
		err error
	}, 2)

	for i := 0; i < 2; i++ {
		idx := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := uc.StartIngestion(context.Background(), "concorrente_maint.csv", strings.NewReader(csvData), "ZM01")
			results[idx] = struct {
				id  string
				err error
			}{id: id, err: err}
		}()
	}
	wg.Wait()

	successCount := 0
	duplicateCount := 0
	for _, res := range results {
		if res.err == nil {
			successCount++
		} else if errors.Is(res.err, domain.ErrDuplicateUpload) {
			duplicateCount++
		}
	}

	if successCount != 1 || duplicateCount != 1 {
		t.Fatalf("esperava 1 sucesso e 1 erro de duplicidade sob concorrencia, obteve %d sucessos e %d duplicatas (erros: %v, %v)",
			successCount, duplicateCount, results[0].err, results[1].err)
	}

	if len(uploadRepo.uploads) != 1 {
		t.Fatalf("esperava exatamente 1 upload persistido no repositorio, obteve %d", len(uploadRepo.uploads))
	}
}
