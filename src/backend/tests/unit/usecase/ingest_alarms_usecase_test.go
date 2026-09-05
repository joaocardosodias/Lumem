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
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type mockAlarmRepo struct {
	mu              sync.Mutex
	uploads         map[string]*domain.FileUpload
	alarms          []*domain.Alarm
	failBatchInsert bool
}

func newMockAlarmRepo() *mockAlarmRepo {
	return &mockAlarmRepo{
		uploads: make(map[string]*domain.FileUpload),
	}
}

func (m *mockAlarmRepo) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if upload.ID == "" {
		upload.ID = uuid.New().String()
	}
	m.uploads[upload.ID] = upload
	return nil
}

func (m *mockAlarmRepo) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploads[upload.ID] = upload
	return nil
}

func (m *mockAlarmRepo) GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, exists := m.uploads[id]
	if !exists {
		return nil, domain.ErrUploadNotFound
	}
	return u, nil
}

func (m *mockAlarmRepo) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.uploads {
		if u.FileHash == hash {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockAlarmRepo) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failBatchInsert {
		return errors.New("falha simulada no banco ao inserir lote")
	}
	m.alarms = append(m.alarms, alarms...)
	return nil
}

func (m *mockAlarmRepo) DeleteByUploadID(ctx context.Context, uploadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	filtered := m.alarms[:0]
	for _, a := range m.alarms {
		if a.UploadID != uploadID {
			filtered = append(filtered, a)
		}
	}
	m.alarms = filtered
	return nil
}

func (m *mockAlarmRepo) ReconcileStaleUploads(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.uploads {
		if u.Status == "processing" {
			u.Status = "failed"
			u.ErrorLog = "processo interrompido por reinicializacao do servidor"
		}
	}
	return nil
}

func (m *mockAlarmRepo) Predictions() port.PredictionRepository   { return nil }
func (m *mockAlarmRepo) Inspections() port.InspectionRepository   { return nil }
func (m *mockAlarmRepo) Maintenances() port.MaintenanceRepository { return nil }
func (m *mockAlarmRepo) Alarms() port.AlarmRepository             { return m }
func (m *mockAlarmRepo) Uploads() port.UploadRepository           { return m }

type mockAlarmUOW struct {
	repo *mockAlarmRepo
}

func (u *mockAlarmUOW) Do(ctx context.Context, fn func(store port.UnitOfWorkStore) error) error {
	u.repo.mu.Lock()
	savedAlarms := make([]*domain.Alarm, len(u.repo.alarms))
	copy(savedAlarms, u.repo.alarms)
	u.repo.mu.Unlock()

	if err := fn(u.repo); err != nil {
		u.repo.mu.Lock()
		u.repo.alarms = savedAlarms
		u.repo.mu.Unlock()
		return err
	}
	return nil
}

func TestIngestAlarmsUseCase_ValidCSV(t *testing.T) {
	repo := newMockAlarmRepo()
	alarmStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	runner := scheduler.NewSyncJobRunner()
	uc := usecase.NewIngestAlarmsUseCase(repo, repo, alarmStorage, alarmParser, runner, &mockAlarmUOW{repo: repo})

	csvData := `tag,estação,porta,alarme,tempo_normalizacao_seg,horário do alarme
SP04-6-PSD-FAL-P1-PP04,FAL,4,FALHA DE COMUNICACAO,12,2025-11-14 06:35:28
SP04-6-PSD-PIN-P2-PP08,PIN,8,LAÇO DE PDM ABERTO,0,2025-11-14 07:00:00
`
	uploadID, err := uc.StartIngestion(context.Background(), "telemetria.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	status, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status: %v", err)
	}

	if status.Status != "completed" {
		t.Errorf("esperava status completed, obteve %s", status.Status)
	}
	if status.RowsInserted != 2 {
		t.Errorf("esperava 2 linhas inseridas, obteve %d", status.RowsInserted)
	}
}

func TestIngestAlarmsUseCase_ValidXLSX(t *testing.T) {
	repo := newMockAlarmRepo()
	alarmStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	runner := scheduler.NewSyncJobRunner()
	uc := usecase.NewIngestAlarmsUseCase(repo, repo, alarmStorage, alarmParser, runner, &mockAlarmUOW{repo: repo})

	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(0)
	_ = f.SetSheetRow(sheet, "A1", &[]any{"tag", "estacao", "plataforma", "porta", "alarme", "tempo_normalizacao_seg", "horario_alarme"})
	_ = f.SetSheetRow(sheet, "A2", &[]any{"SP04-6-PSD-FAL-P1-PP01", "FAL", "P1", 1, "OBSTACULO", 5, "2026-09-01 10:00:00"})
	_ = f.SetSheetRow(sheet, "A3", &[]any{"SP04-6-PSD-FAL-P2-PP02", "FAL", "P2", 2, "LACO DE PDM", 0, "2026-09-01 10:05:00"})

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("falha ao gerar arquivo xlsx em memoria: %v", err)
	}

	uploadID, err := uc.StartIngestion(context.Background(), "telemetria.xlsx", &buf)
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao xlsx: %v", err)
	}

	status, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status do upload: %v", err)
	}

	if status.Status != "completed" {
		t.Errorf("esperava status completed para XLSX, obteve %s (erro: %s)", status.Status, status.ErrorLog)
	}
	if status.RowsInserted != 2 {
		t.Errorf("esperava 2 linhas inseridas a partir do XLSX, obteve %d", status.RowsInserted)
	}
}

func TestIngestAlarmsUseCase_ShortRowNoPanic(t *testing.T) {
	repo := newMockAlarmRepo()
	alarmStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	runner := scheduler.NewSyncJobRunner()
	uc := usecase.NewIngestAlarmsUseCase(repo, repo, alarmStorage, alarmParser, runner, &mockAlarmUOW{repo: repo})

	csvData := `tag,estação,porta,alarme,tempo_normalizacao_seg,horário do alarme
SP04-6-PSD-FAL-P1-PP04,FAL,4
SP04-6-PSD-PIN-P2-PP08,PIN,8,LAÇO DE PDM ABERTO,0,2025-11-14 07:00:00
`
	uploadID, err := uc.StartIngestion(context.Background(), "telemetria_curta.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	status, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status: %v", err)
	}

	if status.Status != "completed" {
		t.Errorf("esperava status completed sem panico, obteve %s", status.Status)
	}
	if status.RowsInserted != 1 {
		t.Errorf("esperava 1 linha valida inserida, obteve %d", status.RowsInserted)
	}
}

func TestIngestAlarmsUseCase_UnknownPlatformNoFallback(t *testing.T) {
	repo := newMockAlarmRepo()
	alarmStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	runner := scheduler.NewSyncJobRunner()
	uc := usecase.NewIngestAlarmsUseCase(repo, repo, alarmStorage, alarmParser, runner, &mockAlarmUOW{repo: repo})

	csvData := `tag,estação,porta,alarme,tempo_normalizacao_seg,horário do alarme
SP04-6-PSD-FAL-PP04,FAL,4,FALHA DE COMUNICACAO,12,2025-11-14 06:35:28
`
	uploadID, err := uc.StartIngestion(context.Background(), "sem_plataforma.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	status, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status: %v", err)
	}

	if status.RowsInserted != 0 {
		t.Errorf("esperava 0 insercoes pois plataforma desconhecida nao deve ter fallback para P1, obteve %d", status.RowsInserted)
	}
}

func TestIngestAlarmsUseCase_RollbackOnBatchFailure(t *testing.T) {
	repo := newMockAlarmRepo()
	repo.failBatchInsert = true
	alarmStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	runner := scheduler.NewSyncJobRunner()
	uc := usecase.NewIngestAlarmsUseCase(repo, repo, alarmStorage, alarmParser, runner, &mockAlarmUOW{repo: repo})

	csvData := `tag,estação,porta,alarme,tempo_normalizacao_seg,horário do alarme
SP04-6-PSD-FAL-P1-PP04,FAL,4,FALHA DE COMUNICACAO,12,2025-11-14 06:35:28
`
	uploadID, err := uc.StartIngestion(context.Background(), "falha.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	status, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status: %v", err)
	}

	if status.Status != "failed" {
		t.Errorf("esperava status failed apos falha de lote, obteve %s", status.Status)
	}
	if len(repo.alarms) != 0 {
		t.Errorf("esperava 0 alarmes no repositorio apos rollback da transacao, obteve %d", len(repo.alarms))
	}
}

func TestIngestAlarmsUseCase_IdempotencyDuplicate(t *testing.T) {
	repo := newMockAlarmRepo()
	alarmStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	runner := scheduler.NewSyncJobRunner()
	uc := usecase.NewIngestAlarmsUseCase(repo, repo, alarmStorage, alarmParser, runner, &mockAlarmUOW{repo: repo})

	csvData := `tag,estação,porta,alarme,tempo_normalizacao_seg,horário do alarme
SP04-6-PSD-FAL-P1-PP04,FAL,4,FALHA DE COMUNICACAO,12,2025-11-14 06:35:28
`
	id1, err := uc.StartIngestion(context.Background(), "alarms.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("esperava sucesso no primeiro upload: %v", err)
	}

	id2, err := uc.StartIngestion(context.Background(), "alarms_copia.csv", strings.NewReader(csvData))
	if err == nil {
		t.Fatalf("esperava erro de duplicidade no segundo upload")
	}

	if id1 != id2 {
		t.Errorf("esperava que id2 (%s) fosse igual a id1 (%s)", id2, id1)
	}
}

func TestAlarmRepo_ReconcileStaleUploads(t *testing.T) {
	repo := newMockAlarmRepo()
	repo.uploads["up_stale"] = &domain.FileUpload{
		ID:       "up_stale",
		FileName: "interrupted.csv",
		Status:   "processing",
	}

	err := repo.ReconcileStaleUploads(context.Background())
	if err != nil {
		t.Fatalf("falha ao reconciliar: %v", err)
	}

	if repo.uploads["up_stale"].Status != "failed" {
		t.Errorf("esperava status failed, obteve %s", repo.uploads["up_stale"].Status)
	}
}
