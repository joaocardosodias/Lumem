package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/g04/backend/internal/domain"
)

type dummyUploadRepo struct {
	upload *domain.FileUpload
	getErr error
	updErr error
}

func (r *dummyUploadRepo) CreateUpload(ctx context.Context, upload *domain.FileUpload) error {
	return nil
}

func (r *dummyUploadRepo) UpdateUpload(ctx context.Context, upload *domain.FileUpload) error {
	if r.updErr != nil {
		return r.updErr
	}
	r.upload = upload
	return nil
}

func (r *dummyUploadRepo) GetUploadByID(ctx context.Context, id string) (*domain.FileUpload, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.upload == nil {
		r.upload = &domain.FileUpload{ID: id, Status: "pending"}
	}
	return r.upload, nil
}

func (r *dummyUploadRepo) GetUploadByHash(ctx context.Context, hash string) (*domain.FileUpload, error) {
	return nil, nil
}

func (r *dummyUploadRepo) ReconcileStaleUploads(ctx context.Context) error {
	return nil
}

type testSpyLogger struct {
	errors []string
}

func (l *testSpyLogger) Error(msg string, args ...any) {
	l.errors = append(l.errors, msg)
}

func (l *testSpyLogger) Warn(msg string, args ...any) {}
func (l *testSpyLogger) Info(msg string, args ...any) {}

func TestIngestAlarms_RecordFailure_And_FailUpload(t *testing.T) {
	repo := &dummyUploadRepo{}
	logger := &testSpyLogger{}
	uc := &IngestAlarmsUseCase{uploadRepo: repo, logger: logger}

	uc.recordFailure("up-1", "motivo", "log de falha", errors.New("causa raiz"))
	if repo.upload == nil || repo.upload.Status != "failed" || repo.upload.ErrorLog != "motivo: causa raiz" {
		t.Fatalf("esperava status failed e motivo composto, obteve %+v", repo.upload)
	}

	repo.updErr = errors.New("falha no update")
	uc.recordFailure("up-1", "motivo", "log de falha", nil)
	if len(logger.errors) == 0 {
		t.Fatalf("esperava erro logado quando update falha")
	}

	repo.getErr = errors.New("falha no get")
	err := uc.failUpload("up-1", "falha direta")
	if err == nil {
		t.Fatalf("esperava erro no failUpload ao falhar get")
	}
}

func TestIngestMaintenance_RecordFailure_And_FailUpload(t *testing.T) {
	repo := &dummyUploadRepo{}
	logger := &testSpyLogger{}
	uc := &IngestMaintenanceUseCase{uploadRepo: repo, logger: logger}

	uc.recordFailure("up-m", "motivo maint", "log de falha maint", errors.New("causa maint"))
	if repo.upload == nil || repo.upload.Status != "failed" || repo.upload.ErrorLog != "motivo maint: causa maint" {
		t.Fatalf("esperava status failed e motivo composto, obteve %+v", repo.upload)
	}

	repo.updErr = errors.New("falha no update")
	uc.recordFailure("up-m", "motivo", "log de falha", nil)
	if len(logger.errors) == 0 {
		t.Fatalf("esperava erro logado quando update falha")
	}

	repo.getErr = errors.New("falha no get")
	err := uc.failUpload("up-m", "falha direta")
	if err == nil {
		t.Fatalf("esperava erro no failUpload ao falhar get")
	}
}
