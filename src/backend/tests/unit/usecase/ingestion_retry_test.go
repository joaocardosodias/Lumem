package usecase_test

import (
	"context"
	"errors"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"io"
	"testing"
)

type terminalStorage struct {
	customStorageMock
	opened, deleted int
}

func (s *terminalStorage) Open(context.Context, string) (io.ReadCloser, error) {
	s.opened++
	return nil, errors.New("object already removed")
}
func (s *terminalStorage) Delete(context.Context, string) error { s.deleted++; return s.deleteErr }

func TestTerminalUploadRetryOnlyCleansUp(t *testing.T) {
	for _, kind := range []port.IngestionJobType{port.IngestionJobAlarms, port.IngestionJobMaintenance} {
		for _, status := range []string{"completed", "failed"} {
			t.Run(string(kind)+"/"+status, func(t *testing.T) {
				repo := newMockAlarmRepo()
				repo.uploads["upload"] = &domain.FileUpload{ID: "upload", Status: status, RowsInserted: 42, ErrorLog: "original"}
				storage := &terminalStorage{}
				var uc interface {
					ProcessJob(context.Context, port.IngestionJob) error
					FailJob(context.Context, port.IngestionJob, string) error
				}
				if kind == port.IngestionJobAlarms {
					uc = usecase.NewIngestAlarmsUseCaseWithQueue(repo, repo, storage, nil, nil, nil, nil)
				} else {
					uc = usecase.NewIngestMaintenanceUseCaseWithQueue(nil, repo, storage, nil, nil, nil, nil)
				}
				job := port.IngestionJob{Type: kind, UploadID: "upload", StorageKey: "missing.csv"}
				storage.deleteErr = errors.New("cleanup queue unavailable")
				var pending *usecase.CleanupPendingError
				if err := uc.ProcessJob(context.Background(), job); !errors.As(err, &pending) {
					t.Fatalf("expected retryable cleanup: %v", err)
				}
				storage.deleteErr = nil
				if err := uc.ProcessJob(context.Background(), job); err != nil {
					t.Fatal(err)
				}
				if err := uc.FailJob(context.Background(), job, "last attempt"); err != nil {
					t.Fatal(err)
				}
				upload := repo.uploads["upload"]
				if storage.opened != 0 || storage.deleted != 3 || upload.Status != status || upload.RowsInserted != 42 || upload.ErrorLog != "original" {
					t.Fatalf("terminal upload modified: %+v storage=%+v", upload, storage)
				}
			})
		}
	}
}
