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

type mockAlarmRepo struct {
	mu      sync.Mutex
	uploads map[int64]*domain.FileUpload
	alarms  []*domain.Alarm
	nextID  int64
}

func newMockAlarmRepo() *mockAlarmRepo {
	return &mockAlarmRepo{
		uploads: make(map[int64]*domain.FileUpload),
		nextID:  1,
	}
}

func (m *mockAlarmRepo) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	upload.ID = m.nextID
	m.nextID++
	m.uploads[upload.ID] = upload
	return nil
}

func (m *mockAlarmRepo) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploads[upload.ID] = upload
	return nil
}

func (m *mockAlarmRepo) GetUploadByID(ctx context.Context, id int64) (*domain.FileUpload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.uploads[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockAlarmRepo) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.uploads {
		if u.FileHash == hash && u.Status == "completed" {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockAlarmRepo) BatchInsertAlarms(ctx context.Context, alarms []*domain.Alarm) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alarms = append(m.alarms, alarms...)
	return nil
}

func TestIngestAlarmsUseCase_ValidCSV(t *testing.T) {
	repo := newMockAlarmRepo()
	uc := usecase.NewIngestAlarmsUseCase(repo)

	csvData := `TAG,Horário do Alarme,Estação,Porta,Alarme,Horário da Normalizacao,tempo_normalizacao_seg,plataforma_extraida,porta_extraida
PSD_FAL_FAL2_PDM09_OBS,2026-05-12 08:16:52,FAL,9.0,OBS,2026-05-12 08:16:53,1,FAL2,9.0
PSD_LUZ_LUZ1_PDM08_OBS,2026-05-12 08:15:09,LUZ,8.0,OBS,2026-05-12 08:15:10,1,LUZ1,8.0
`
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	upload, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status: %v", err)
	}

	if upload.Status != "completed" {
		t.Errorf("esperava status completed, obteve %s", upload.Status)
	}

	if upload.RowsInserted != 2 {
		t.Errorf("esperava 2 linhas inseridas, obteve %d", upload.RowsInserted)
	}

	if len(repo.alarms) != 2 {
		t.Errorf("esperava 2 alarmes no repositorio, obteve %d", len(repo.alarms))
	}
}

func TestIngestAlarmsUseCase_IdempotencyDuplicate(t *testing.T) {
	repo := newMockAlarmRepo()
	uc := usecase.NewIngestAlarmsUseCase(repo)

	csvData := `TAG,Horário do Alarme,Estação,Porta,Alarme,Horário da Normalizacao,tempo_normalizacao_seg,plataforma_extraida,porta_extraida
PSD_FAL_FAL2_PDM09_OBS,2026-05-12 08:16:52,FAL,9.0,OBS,2026-05-12 08:16:53,1,FAL2,9.0
`
	uploadID1, err := uc.StartIngestion(context.Background(), "duplicado.csv", strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("falha ao iniciar primeira ingestao: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	uploadID2, err := uc.StartIngestion(context.Background(), "duplicado.csv", strings.NewReader(csvData))
	if err != domain.ErrDuplicateUpload {
		t.Fatalf("esperava ErrDuplicateUpload, obteve: %v", err)
	}

	if uploadID2 != uploadID1 {
		t.Errorf("deveria retornar o uploadID original %d, obteve %d", uploadID1, uploadID2)
	}
}

func TestIngestAlarmsUseCase_MissingHeaders(t *testing.T) {
	repo := newMockAlarmRepo()
	uc := usecase.NewIngestAlarmsUseCase(repo)

	invalidCSV := `ColunaA,ColunaB,ColunaC
1,2,3
`
	uploadID, err := uc.StartIngestion(context.Background(), "invalido.csv", strings.NewReader(invalidCSV))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	upload, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil {
		t.Fatalf("falha ao buscar status: %v", err)
	}

	if upload.Status != "failed" {
		t.Errorf("esperava status failed, obteve %s", upload.Status)
	}
}
