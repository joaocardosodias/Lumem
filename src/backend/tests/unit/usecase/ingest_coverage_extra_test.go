package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/g04/backend/internal/adapter/scheduler"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

type customStorageMock struct {
	storeKey  string
	storeHash string
	storeErr  error
	openErr   error
	deleteErr error
}

func (s *customStorageMock) Store(ctx context.Context, name string, r io.Reader) (string, string, error) {
	if s.storeErr != nil {
		return "", "", s.storeErr
	}
	return s.storeKey, s.storeHash, nil
}

func (s *customStorageMock) Open(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	return io.NopCloser(strings.NewReader("dummy")), nil
}

func (s *customStorageMock) Delete(ctx context.Context, storageKey string) error {
	return s.deleteErr
}

type customAlarmParserMock struct {
	records []port.RawAlarmRecord
	err     error
}

func (p *customAlarmParserMock) ParseAlarms(r io.Reader, fileName string) ([]port.RawAlarmRecord, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.records, nil
}

type customMaintParserMock struct {
	records []port.RawMaintenanceRecord
	err     error
}

func (p *customMaintParserMock) ParseMaintenance(r io.Reader, fileName, explicitOrderType string) ([]port.RawMaintenanceRecord, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.records, nil
}

type customUOWStoreMock struct {
	alarmRepo   *mockAlarmRepo
	maintRepo   *mockMaintRepo
	uowGetIDErr error
	uowUpdErr   error
}

func (m *customUOWStoreMock) Predictions() port.PredictionRepository   { return nil }
func (m *customUOWStoreMock) Inspections() port.InspectionRepository   { return nil }
func (m *customUOWStoreMock) Maintenances() port.MaintenanceRepository { return m.maintRepo }
func (m *customUOWStoreMock) Alarms() port.AlarmRepository             { return m.alarmRepo }
func (m *customUOWStoreMock) Uploads() port.UploadRepository           { return &customUploadRepoWrap{inner: m.alarmRepo, getIDErr: m.uowGetIDErr, updErr: m.uowUpdErr} }

type customUploadRepoWrap struct {
	inner    *mockAlarmRepo
	getIDErr error
	updErr   error
}

func (w *customUploadRepoWrap) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	return w.inner.CreateUpload(ctx, upload)
}

func (w *customUploadRepoWrap) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	if w.updErr != nil {
		return w.updErr
	}
	return w.inner.UpdateUpload(ctx, upload)
}

func (w *customUploadRepoWrap) GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error) {
	if w.getIDErr != nil {
		return nil, w.getIDErr
	}
	return w.inner.GetUploadByID(ctx, id)
}

func (w *customUploadRepoWrap) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	return w.inner.GetUploadByHash(ctx, hash)
}

func (w *customUploadRepoWrap) ReconcileStaleUploads(ctx context.Context) error {
	return w.inner.ReconcileStaleUploads(ctx)
}

type stepUploadRepoMock struct {
	inner        *mockAlarmRepo
	getHashCalls int
	foundUpload  *domain.FileUpload
	createErr    error
}

func (s *stepUploadRepoMock) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	if s.createErr != nil {
		return s.createErr
	}
	return s.inner.CreateUpload(ctx, upload)
}

func (s *stepUploadRepoMock) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	return s.inner.UpdateUpload(ctx, upload)
}

func (s *stepUploadRepoMock) GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error) {
	return s.inner.GetUploadByID(ctx, id)
}

func (s *stepUploadRepoMock) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	s.getHashCalls++
	if s.getHashCalls == 1 {
		return nil, nil
	}
	return s.foundUpload, nil
}

func (s *stepUploadRepoMock) ReconcileStaleUploads(ctx context.Context) error {
	return s.inner.ReconcileStaleUploads(ctx)
}

type customUOWMock struct {
	store *customUOWStoreMock
	doErr error
}

func (u *customUOWMock) Do(ctx context.Context, fn func(store port.UnitOfWorkStore) error) error {
	if u.doErr != nil {
		return u.doErr
	}
	return fn(u.store)
}

func TestIngestAlarms_CheckDuplicateHash_DeleteError(t *testing.T) {
	repo := newMockAlarmRepo()
	repo.uploads["existing-id"] = &domain.FileUpload{ID: "existing-id", FileHash: "hash123"}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash123", deleteErr: errors.New("delete failed")}
	parserMock := &customAlarmParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	_, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if !errors.Is(err, domain.ErrDuplicateUpload) {
		t.Fatalf("esperava ErrDuplicateUpload")
	}
	if len(logger.warns) == 0 {
		t.Fatalf("esperava aviso de erro de remocao temporaria no logger")
	}
}

func TestIngestAlarms_HandleCreateUploadError_DeleteErrorAndHashNotFound(t *testing.T) {
	repo := newMockAlarmRepo()
	repo.createUploadErr = domain.ErrDuplicateUpload
	repo.getUploadByHashErr = errors.New("get hash failed")
	stor := &customStorageMock{storeKey: "key1", storeHash: "newhash", deleteErr: errors.New("delete failed")}
	parserMock := &customAlarmParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	id, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if !errors.Is(err, domain.ErrDuplicateUpload) || id != "" {
		t.Fatalf("esperava id vazio e ErrDuplicateUpload")
	}
	if len(logger.warns) == 0 {
		t.Fatalf("esperava log de aviso ao falhar delete")
	}
}

func TestIngestAlarms_HandleCreateUploadError_DuplicateFound(t *testing.T) {
	repo := newMockAlarmRepo()
	stepRepo := &stepUploadRepoMock{
		inner:       repo,
		createErr:   domain.ErrDuplicateUpload,
		foundUpload: &domain.FileUpload{ID: "dup-existing", FileHash: "hash-match"},
	}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash-match"}
	parserMock := &customAlarmParserMock{}
	runner := scheduler.NewSyncJobRunner()
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, stepRepo, stor, parserMock, runner, uow, nil)
	id, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if !errors.Is(err, domain.ErrDuplicateUpload) || id != "dup-existing" {
		t.Fatalf("esperava id dup-existing, obteve %s", id)
	}
}

func TestIngestAlarms_ReadAndParse_StorageOpenError(t *testing.T) {
	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", openErr: errors.New("open disk error")}
	parserMock := &customAlarmParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, err := uc.GetStatus(context.Background(), uploadID)
	if err != nil || status.Status != "failed" || !strings.Contains(status.ErrorLog, "open disk error") {
		t.Fatalf("esperava status failed com erro de abertura, obteve %+v", status)
	}
}

func TestIngestAlarms_ReadAndParse_ParserError(t *testing.T) {
	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customAlarmParserMock{err: errors.New("parse syntax error")}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, _ := uc.GetStatus(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "parse syntax error") {
		t.Fatalf("esperava status failed por erro de parser")
	}
}

func TestIngestAlarms_ReadAndParse_EmptyRecords(t *testing.T) {
	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customAlarmParserMock{records: []port.RawAlarmRecord{}}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, _ := uc.GetStatus(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "arquivo vazio") {
		t.Fatalf("esperava status failed por arquivo vazio")
	}
}

func TestIngestAlarms_ConvertRaw_FallbackSourceAndInvalidAlarm(t *testing.T) {
	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customAlarmParserMock{
		records: []port.RawAlarmRecord{
			{
				Tag:                    "SP04-6-PSD-FAL-P1-PP01",
				StationCode:            "FAL",
				Platform:               "P1",
				DoorNumber:             1,
				AlarmType:              "Falha",
				DurationSeconds:        10,
				EventTimestamp:         "2026-09-01 10:00:00",
				NormalizationTimestamp: "2026-09-01 10:00:10",
			},
			{
				Tag:            "TAG_INVALIDA",
				StationCode:    "FAL",
				Platform:       "INVALID",
				DoorNumber:     999,
				AlarmType:      "Erro",
				EventTimestamp: "2026-09-01 10:00:00",
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, nil)
	uploadID, err := uc.StartIngestion(context.Background(), "arquivo_sem_extensao", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, _ := uc.GetStatus(context.Background(), uploadID)
	if status.Status != "completed" || status.RowsProcessed != 2 || status.RowsInserted != 1 {
		t.Fatalf("esperava 2 processados e 1 inserido, obteve %+v", status)
	}
	if len(repo.alarms) != 1 || repo.alarms[0].Source != "telemetry" {
		t.Fatalf("esperava source telemetry por fallback")
	}
}

func TestIngestAlarms_BatchInsertion_Over500AndBatchInsertError(t *testing.T) {
	records := make([]port.RawAlarmRecord, 505)
	for i := 0; i < 505; i++ {
		records[i] = port.RawAlarmRecord{
			Tag:            fmt.Sprintf("SP04-6-PSD-FAL-P1-PP%02d", (i%24)+1),
			StationCode:    "FAL",
			Platform:       "P1",
			DoorNumber:     (i % 24) + 1,
			AlarmType:      "Teste",
			EventTimestamp: "2026-09-01 10:00:00",
		}
	}

	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customAlarmParserMock{records: records}
	runner := scheduler.NewSyncJobRunner()
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, nil)
	uploadID, err := uc.StartIngestion(context.Background(), "lote_grande.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("esperava sucesso no lote grande")
	}

	status, _ := uc.GetStatus(context.Background(), uploadID)
	if status.Status != "completed" || status.RowsInserted != 505 {
		t.Fatalf("esperava 505 inseridos em multiplos lotes, obteve %+v", status)
	}
}

func TestIngestAlarms_FinalizeUpload_GetByID_Error(t *testing.T) {
	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customAlarmParserMock{
		records: []port.RawAlarmRecord{
			{
				Tag:            "SP04-6-PSD-FAL-P1-PP01",
				StationCode:    "FAL",
				Platform:       "P1",
				DoorNumber:     1,
				AlarmType:      "Falha",
				EventTimestamp: "2026-09-01 10:00:00",
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo, uowGetIDErr: errors.New("uow get id error")}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao")
	}

	status, _ := uc.GetStatus(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "uow get id error") {
		t.Fatalf("esperava status failed com erro de get id no uow")
	}
}

func TestIngestAlarms_FinalizeUpload_Update_Error(t *testing.T) {
	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customAlarmParserMock{
		records: []port.RawAlarmRecord{
			{
				Tag:            "SP04-6-PSD-FAL-P1-PP01",
				StationCode:    "FAL",
				Platform:       "P1",
				DoorNumber:     1,
				AlarmType:      "Falha",
				EventTimestamp: "2026-09-01 10:00:00",
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo, uowUpdErr: errors.New("uow update error")}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao")
	}

	status, _ := uc.GetStatus(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "uow update error") {
		t.Fatalf("esperava status failed com erro de update no uow")
	}
}

func TestIngestAlarms_FailUpload_GetUploadByID_Error(t *testing.T) {
	repo := newMockAlarmRepo()
	repo.getUploadByIDErr = errors.New("get failed")
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", openErr: errors.New("open failed")}
	parserMock := &customAlarmParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	_, _ = uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if len(logger.errors) == 0 {
		t.Fatalf("esperava erro registrado no logger")
	}
}

func TestIngestAlarms_FailUpload_UpdateUpload_Error(t *testing.T) {
	repo := newMockAlarmRepo()
	repo.failUpdateUpload = true
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", openErr: errors.New("open failed")}
	parserMock := &customAlarmParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	_, _ = uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if len(logger.errors) == 0 {
		t.Fatalf("esperava erro registrado no logger")
	}
}

func TestIngestAlarms_ProcessFileAsync_StorageDeleteError(t *testing.T) {
	repo := newMockAlarmRepo()
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", deleteErr: errors.New("delete failed")}
	parserMock := &customAlarmParserMock{
		records: []port.RawAlarmRecord{
			{
				Tag:            "SP04-6-PSD-FAL-P1-PP01",
				StationCode:    "FAL",
				Platform:       "P1",
				DoorNumber:     1,
				AlarmType:      "Falha",
				EventTimestamp: "2026-09-01 10:00:00",
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: repo}}

	uc := usecase.NewIngestAlarmsUseCase(repo, repo, stor, parserMock, runner, uow, logger)
	_, _ = uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"))
	if len(logger.warns) == 0 {
		t.Fatalf("esperava warning registrado no logger ao falhar delete")
	}
}

func TestIngestMaintenance_CheckDuplicateHash_DeleteError(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	uploadRepo.uploads["existing-id"] = &domain.FileUpload{ID: "existing-id", FileHash: "maint-hash"}
	stor := &customStorageMock{storeKey: "key1", storeHash: "maint-hash", deleteErr: errors.New("delete failed")}
	parserMock := &customMaintParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: &mockMaintRepo{}}}

	uc := usecase.NewIngestMaintenanceUseCase(&mockMaintRepo{}, uploadRepo, stor, parserMock, runner, uow, logger)
	_, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "")
	if !errors.Is(err, domain.ErrDuplicateUpload) {
		t.Fatalf("esperava ErrDuplicateUpload")
	}
	if len(logger.warns) == 0 {
		t.Fatalf("esperava aviso no logger")
	}
}

func TestIngestMaintenance_HandleCreateUploadError_DeleteErrorAndHashNotFound(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	uploadRepo.createUploadErr = domain.ErrDuplicateUpload
	uploadRepo.getUploadByHashErr = errors.New("get hash failed")
	stor := &customStorageMock{storeKey: "key1", storeHash: "newhash", deleteErr: errors.New("delete failed")}
	parserMock := &customMaintParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: &mockMaintRepo{}}}

	uc := usecase.NewIngestMaintenanceUseCase(&mockMaintRepo{}, uploadRepo, stor, parserMock, runner, uow, logger)
	id, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "ZM01")
	if !errors.Is(err, domain.ErrDuplicateUpload) || id != "" {
		t.Fatalf("esperava id vazio e ErrDuplicateUpload")
	}
	if len(logger.warns) == 0 {
		t.Fatalf("esperava log de aviso ao falhar delete")
	}
}

func TestIngestMaintenance_HandleCreateUploadError_DuplicateFound(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	stepRepo := &stepUploadRepoMock{
		inner:       uploadRepo,
		createErr:   domain.ErrDuplicateUpload,
		foundUpload: &domain.FileUpload{ID: "dup-existing", FileHash: "hash-match"},
	}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash-match"}
	parserMock := &customMaintParserMock{}
	runner := scheduler.NewSyncJobRunner()
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: &mockMaintRepo{}}}

	uc := usecase.NewIngestMaintenanceUseCase(&mockMaintRepo{}, stepRepo, stor, parserMock, runner, uow, nil)
	id, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "ZM01")
	if !errors.Is(err, domain.ErrDuplicateUpload) || id != "dup-existing" {
		t.Fatalf("esperava id dup-existing, obteve %s", id)
	}
}

func TestIngestMaintenance_ReadAndParse_StorageOpenError(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", openErr: errors.New("open disk error")}
	parserMock := &customMaintParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "ZM01")
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, err := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if err != nil || status.Status != "failed" || !strings.Contains(status.ErrorLog, "open disk error") {
		t.Fatalf("esperava status failed com erro de abertura, obteve %+v", status)
	}
}

func TestIngestMaintenance_ReadAndParse_ParserError(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customMaintParserMock{err: errors.New("parse syntax error")}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "ZM01")
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, _ := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "parse syntax error") {
		t.Fatalf("esperava status failed por erro de parser")
	}
}

func TestIngestMaintenance_ReadAndParse_EmptyRecords(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customMaintParserMock{records: []port.RawMaintenanceRecord{}}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "ZM01")
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, _ := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "arquivo vazio") {
		t.Fatalf("esperava status failed por arquivo vazio")
	}
}

func TestIngestMaintenance_ConvertRaw_FallbackSourceAndInvalidEvent(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customMaintParserMock{
		records: []port.RawMaintenanceRecord{
			{
				SAPOrderNumber:       "ORD-1",
				OrderType:            "ZM01",
				StationCode:          "FAL",
				InstallationLocation: "PLATAFORMA 1",
				StartTimestamp:       "2026-09-01 10:00:00",
				EndTimestamp:         "2026-09-01 11:00:00",
				DurationHours:        1.0,
				Description:          "Troca",
			},
			{
				SAPOrderNumber: "ORD-INVALID",
				StartTimestamp: "",
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, nil)
	uploadID, err := uc.StartIngestion(context.Background(), "arquivo_sem_extensao", strings.NewReader("data"), "ZM01")
	if err != nil {
		t.Fatalf("inicio da ingestao deveria ter sucesso")
	}

	status, _ := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if status.Status != "completed" || status.RowsProcessed != 2 || status.RowsInserted != 1 {
		t.Fatalf("esperava 2 processados e 1 inserido, obteve %+v", status)
	}
	if len(maintRepo.events) != 1 || maintRepo.events[0].Source != "maintenance" {
		t.Fatalf("esperava source maintenance por fallback")
	}
}

func TestIngestMaintenance_BatchInsertion_Over500(t *testing.T) {
	records := make([]port.RawMaintenanceRecord, 505)
	for i := 0; i < 505; i++ {
		records[i] = port.RawMaintenanceRecord{
			SAPOrderNumber: fmt.Sprintf("ORD-%04d", i),
			OrderType:      "ZM01",
			StationCode:    "FAL",
			StartTimestamp: "2026-09-01 10:00:00",
			DurationHours:  1.0,
			Description:    "Manutencao teste",
		}
	}

	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customMaintParserMock{records: records}
	runner := scheduler.NewSyncJobRunner()
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, nil)
	uploadID, err := uc.StartIngestion(context.Background(), "lote_grande.csv", strings.NewReader("data"), "ZM01")
	if err != nil {
		t.Fatalf("esperava sucesso no lote grande")
	}

	status, _ := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if status.Status != "completed" || status.RowsInserted != 505 {
		t.Fatalf("esperava 505 inseridos em multiplos lotes, obteve %+v", status)
	}
}

func TestIngestMaintenance_FinalizeUpload_GetByID_Error(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customMaintParserMock{
		records: []port.RawMaintenanceRecord{
			{
				SAPOrderNumber: "ORD-1",
				OrderType:      "ZM01",
				StartTimestamp: "2026-09-01 10:00:00",
				DurationHours:  1.0,
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo, uowGetIDErr: errors.New("uow get id error")}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao")
	}

	status, _ := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "uow get id error") {
		t.Fatalf("esperava status failed com erro de get id no uow")
	}
}

func TestIngestMaintenance_FinalizeUpload_Update_Error(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1"}
	parserMock := &customMaintParserMock{
		records: []port.RawMaintenanceRecord{
			{
				SAPOrderNumber: "ORD-1",
				OrderType:      "ZM01",
				StartTimestamp: "2026-09-01 10:00:00",
				DurationHours:  1.0,
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo, uowUpdErr: errors.New("uow update error")}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	uploadID, err := uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "")
	if err != nil {
		t.Fatalf("falha ao iniciar ingestao")
	}

	status, _ := uploadRepo.GetUploadByID(context.Background(), uploadID)
	if status.Status != "failed" || !strings.Contains(status.ErrorLog, "uow update error") {
		t.Fatalf("esperava status failed com erro de update no uow")
	}
}

func TestIngestMaintenance_FailUpload_GetUploadByID_Error(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	uploadRepo.getUploadByIDErr = errors.New("get failed")
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", openErr: errors.New("open failed")}
	parserMock := &customMaintParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	_, _ = uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "")
	if len(logger.errors) == 0 {
		t.Fatalf("esperava erro registrado no logger")
	}
}

func TestIngestMaintenance_FailUpload_UpdateUpload_Error(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	uploadRepo.failUpdateUpload = true
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", openErr: errors.New("open failed")}
	parserMock := &customMaintParserMock{}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	_, _ = uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "")
	if len(logger.errors) == 0 {
		t.Fatalf("esperava erro registrado no logger")
	}
}

func TestIngestMaintenance_ProcessFileAsync_StorageDeleteError(t *testing.T) {
	uploadRepo := newMockAlarmRepo()
	maintRepo := &mockMaintRepo{}
	stor := &customStorageMock{storeKey: "key1", storeHash: "hash1", deleteErr: errors.New("delete failed")}
	parserMock := &customMaintParserMock{
		records: []port.RawMaintenanceRecord{
			{
				SAPOrderNumber: "ORD-1",
				OrderType:      "ZM01",
				StartTimestamp: "2026-09-01 10:00:00",
				DurationHours:  1.0,
			},
		},
	}
	runner := scheduler.NewSyncJobRunner()
	logger := &mockLogger{}
	uow := &customUOWMock{store: &customUOWStoreMock{alarmRepo: uploadRepo, maintRepo: maintRepo}}

	uc := usecase.NewIngestMaintenanceUseCase(maintRepo, uploadRepo, stor, parserMock, runner, uow, logger)
	_, _ = uc.StartIngestion(context.Background(), "test.csv", strings.NewReader("data"), "")
	if len(logger.warns) == 0 {
		t.Fatalf("esperava warning registrado no logger ao falhar delete")
	}
}
