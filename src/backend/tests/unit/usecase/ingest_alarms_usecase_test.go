package usecase_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/google/uuid"
)

type mockAlarmRepo struct {
	mu      sync.Mutex
	uploads map[string]*domain.FileUpload
	alarms  []*domain.Alarm
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
		return nil, domain.ErrUserNotFound
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
	m.alarms = append(m.alarms, alarms...)
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

func TestIngestAlarmsUseCase_ValidCSV(t *testing.T) {
	repo := newMockAlarmRepo()
	uc := usecase.NewIngestAlarmsUseCase(repo)

	csvData := `tag,estação,porta,alarme,tempo_normalizacao_seg,horário do alarme
SP04-6-PSD-FAL-P1-PP04,FAL,4,FALHA DE COMUNICACAO,12,2025-11-14 06:35:28
SP04-6-PSD-PIN-P2-PP08,PIN,8,LAÇO DE PDM ABERTO,0,2025-11-14 07:00:00
`
	uploadID, err := uc.StartIngestion(context.Background(), "telemetria.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

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

func TestIngestAlarmsUseCase_IdempotencyDuplicate(t *testing.T) {
	repo := newMockAlarmRepo()
	uc := usecase.NewIngestAlarmsUseCase(repo)

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
