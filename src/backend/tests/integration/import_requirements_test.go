package integration_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/parser"
	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// Controlled queue: assertions can observe processing before invoking the worker.
// This fake does not validate River, SQL transactions or concurrent workers.
type importQueue struct {
	repo *intAlarmRepo
	jobs []port.IngestionJob
}

func (q *importQueue) Enqueue(ctx context.Context, upload *domain.FileUpload, job port.IngestionJob) error {
	if err := q.repo.CreateUpload(ctx, upload); err != nil {
		return err
	}
	q.jobs = append(q.jobs, job)
	return nil
}

type importFixture struct {
	router      *gin.Engine
	store       *inMemoryStore
	queue       *importQueue
	alarm       *usecase.IngestAlarmsUseCase
	maintenance *usecase.IngestMaintenanceUseCase
	token       string
}

func newImportFixture(t *testing.T) *importFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	s := newInMemoryStore()
	ar := &intAlarmRepo{store: s}
	mr := &intMaintenanceRepo{store: s}
	queue := &importQueue{repo: ar}
	uow := &intUnitOfWork{store: &intUOWStore{alarmRepo: ar, maintRepo: mr, uploadRepo: ar}}
	raw := storage.NewLocalTempStorage()
	f := &importFixture{store: s, queue: queue}
	f.alarm = usecase.NewIngestAlarmsUseCaseWithQueue(ar, ar, raw, parser.NewDefaultAlarmParser(), queue, uow, nil)
	f.maintenance = usecase.NewIngestMaintenanceUseCaseWithQueue(mr, ar, raw, parser.NewDefaultMaintenanceParser(), queue, uow, nil)
	h := httpAdapter.NewUploadHandler(f.alarm, f.maintenance)
	f.router = gin.New()
	f.router.Use(httpAdapter.AuthMiddleware("import-test-key"))
	f.router.POST("/uploads/alarms", h.UploadAlarms)
	f.router.POST("/uploads/maintenance", h.UploadMaintenance)
	f.router.GET("/uploads/:id", h.GetUploadStatus)
	var err error
	f.token, err = security.NewJWTTokenGenerator("import-test-key").Generate(map[string]any{"sub": "import-user", "role": "operator"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, job := range queue.jobs {
			if err := raw.Delete(context.Background(), job.StorageKey); err != nil {
				t.Error(err)
			}
		}
	})
	return f
}

func (f *importFixture) upload(t *testing.T, kind, filename, orderType string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if filename != "" {
		part, err := w.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.WriteField("order_type", orderType); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/uploads/"+kind, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+f.token)
	res := httptest.NewRecorder()
	f.router.ServeHTTP(res, req)
	return res
}

func (f *importFixture) status(t *testing.T, id string) httpAdapter.UploadStatusResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/uploads/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+f.token)
	res := httptest.NewRecorder()
	f.router.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}
	var result httpAdapter.UploadStatusResponse
	if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func (f *importFixture) process(t *testing.T) {
	t.Helper()
	if len(f.queue.jobs) != 1 {
		t.Fatalf("expected exactly one job, got %d", len(f.queue.jobs))
	}
	j := f.queue.jobs[0]
	ctx := context.Background()
	var err error
	if j.Type == port.IngestionJobAlarms {
		err = f.alarm.ProcessJob(ctx, j)
		if err != nil {
			err = f.alarm.FailJob(ctx, j, err.Error())
		}
	} else {
		err = f.maintenance.ProcessJob(ctx, j)
		if err != nil {
			err = f.maintenance.FailJob(ctx, j, err.Error())
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(j.StorageKey); !os.IsNotExist(err) {
		t.Fatalf("temporary upload was not removed: %v", err)
	}
}

func importData(t *testing.T, format string, rows [][]string) []byte {
	t.Helper()
	var data bytes.Buffer
	if format == "csv" {
		w := csv.NewWriter(&data)
		if err := w.WriteAll(rows); err != nil {
			t.Fatal(err)
		}
	} else {
		book := excelize.NewFile()
		defer book.Close()
		for i, row := range rows {
			cells := make([]any, len(row))
			for j := range row {
				cells[j] = row[j]
			}
			if err := book.SetSheetRow(book.GetSheetName(0), fmt.Sprintf("A%d", i+1), &cells); err != nil {
				t.Fatal(err)
			}
		}
		if err := book.Write(&data); err != nil {
			t.Fatal(err)
		}
	}
	return data.Bytes()
}

func importRows(kind, orderType string) [][]string {
	if kind == "alarms" {
		return [][]string{
			{"tag", "estacao", "plataforma", "porta", "alarme", "horario_alarme", "tempo_normalizacao_seg"},
			{"SP04-6-PSD-FAL-P1-PP04", "FAL", "P1", "4", "OBSTACULO", "2026-09-01 10:00:00", "12"},
			{"SP04-6-PSD-PIN-P2-PP08", "PIN", "P2", "8", "FALHA", "2026-09-01 11:00:00", "20"},
		}
	}
	return [][]string{
		{"ordem", "tipo_de_ordem", "local_de_instalacao", "estacao", "inicio_avaria_datetime", "fim_avaria_datetime", "duracao_avaria_horas", "descricao", "sintoma", "causa", "acao"},
		{"1001", orderType, "SP04-6-PSD-FAL-P1-PP04", "FAL", "2026-09-01 10:00:00", "2026-09-01 11:30:00", "1.5", "Reparo", "Ruido", "Desgaste", "Troca"},
		{"1002", orderType, "SP04-6-PSD-PIN-P2-PP08", "PIN", "2026-09-01 12:00:00", "2026-09-01 13:30:00", "1.5", "Reparo", "Ruido", "Desgaste", "Troca"},
	}
}

func TestRF03RF04_ImportFormatsAndTypes(t *testing.T) {
	for _, orderType := range []string{"", "ZM01", "ZM02", "ZM03"} {
		kind := "maintenance"
		if orderType == "" {
			kind = "alarms"
		}
		for _, format := range []string{"csv", "xlsx"} {
			t.Run(kind+orderType+"/"+format, func(t *testing.T) {
				f := newImportFixture(t)
				rows := importRows(kind, orderType)
				data := importData(t, format, rows)
				res := f.upload(t, kind, "valid."+format, orderType, data)
				if res.Code != 202 {
					t.Fatalf("expected 202, got %d: %s", res.Code, res.Body.String())
				}
				var accepted struct {
					ID     string `json:"upload_id"`
					Status string `json:"status"`
				}
				if err := json.Unmarshal(res.Body.Bytes(), &accepted); err != nil {
					t.Fatal(err)
				}
				if accepted.ID == "" || accepted.Status != "processing" {
					t.Fatalf("invalid acceptance: %+v", accepted)
				}
				before := f.status(t, accepted.ID)
				if before.Status != "processing" || before.RowsInserted != 0 || len(f.store.alarms)+len(f.store.maintenances) != 0 {
					t.Fatalf("processed before worker: %+v", before)
				}
				// Duplicate must be rejected both while queued and after completion.
				if duplicate := f.upload(t, kind, "renamed."+format, orderType, data); duplicate.Code != 409 {
					t.Fatalf("queued duplicate: %d", duplicate.Code)
				}
				f.process(t)
				after := f.status(t, accepted.ID)
				if after.Status != "completed" || after.RowsProcessed != 2 || after.RowsInserted != 2 || after.CompletedAt == nil || after.ErrorLog != "" {
					t.Fatalf("unexpected final status: %+v", after)
				}
				if kind == "alarms" {
					if len(f.store.alarms) != 2 {
						t.Fatal("wrong alarm count")
					}
					for i, a := range f.store.alarms {
						r := rows[i+1]
						if a.UploadID != accepted.ID || a.Tag != r[0] || a.StationCode != r[1] || a.Platform != r[2] || fmt.Sprint(a.DoorNumber) != r[3] || a.AlarmType != r[4] || a.EventTimestamp != r[5] || fmt.Sprint(a.DurationSeconds) != r[6] || a.Source != format {
							t.Fatalf("alarm mapping mismatch: %+v", a)
						}
					}
				} else {
					if len(f.store.maintenances) != 2 {
						t.Fatal("wrong maintenance count")
					}
					for i, m := range f.store.maintenances {
						r := rows[i+1]
						if m.UploadID != accepted.ID || m.SAPOrderNumber != r[0] || m.OrderType != orderType || m.InstallationLocation != r[2] || m.StationCode != r[3] || m.StartTimestamp != r[4] || m.EndTimestamp != r[5] || m.DurationHours != 1.5 || m.Description != r[7] || m.Symptom != r[8] || m.Cause != r[9] || m.ActionTaken != r[10] || m.Source != format {
							t.Fatalf("maintenance mapping mismatch: %+v", m)
						}
					}
				}
				if duplicate := f.upload(t, kind, "again."+format, orderType, data); duplicate.Code != 409 {
					t.Fatalf("completed duplicate: %d", duplicate.Code)
				}
				// Redelivery of a completed job must not insert the rows again.
				f.process(t)
				if len(f.queue.jobs) != 1 || len(f.store.uploads) != 1 || len(f.store.alarms)+len(f.store.maintenances) != 2 {
					t.Fatal("duplicate changed persisted data or enqueued another job")
				}
			})
		}
	}
}

func TestRF03RF04_InvalidFiles(t *testing.T) {
	for _, kind := range []string{"alarms", "maintenance"} {
		for _, tc := range []struct {
			name, filename string
			data           []byte
			status         int
			code           string
		}{
			{"missing_file", "", nil, 400, "ERR_MISSING_FILE"},
			{"unsupported_extension", "file.pdf", importData(t, "csv", importRows(kind, "ZM01")), 415, "ERR_UNSUPPORTED_FILE_TYPE"},
			{"legacy_xls", "file.xls", []byte("legacy"), 415, "ERR_UNSUPPORTED_FILE_TYPE"},
			{"no_extension", "file", []byte("data"), 415, "ERR_UNSUPPORTED_FILE_TYPE"},
			{"empty", "empty.csv", nil, 202, ""},
			{"missing_columns", "invalid.csv", []byte("unrelated\nvalue\n"), 202, ""},
			{"header_only", "header.csv", importData(t, "csv", importRows(kind, "ZM01")[:1]), 202, ""},
			{"corrupt_xlsx", "corrupt.xlsx", []byte("not an excel file"), 202, ""},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				f := newImportFixture(t)
				res := f.upload(t, kind, tc.filename, "ZM01", tc.data)
				if res.Code != tc.status {
					t.Fatalf("expected %d, got %d: %s", tc.status, res.Code, res.Body.String())
				}
				if tc.status != 202 {
					var problem httpAdapter.ProblemDetails
					if err := json.Unmarshal(res.Body.Bytes(), &problem); err != nil {
						t.Fatal(err)
					}
					if problem.Code != tc.code || len(f.queue.jobs) != 0 || len(f.store.uploads) != 0 {
						t.Fatalf("rejection mismatch: %+v", problem)
					}
				} else {
					f.process(t)
					status := f.status(t, f.queue.jobs[0].UploadID)
					if status.Status != "failed" || status.ErrorLog == "" || status.RowsInserted != 0 || status.CompletedAt == nil {
						t.Fatalf("invalid file must fail visibly: %+v", status)
					}
				}
				if len(f.store.alarms)+len(f.store.maintenances) != 0 {
					t.Fatal("invalid file persisted records")
				}
			})
		}
	}
}

func TestRF03RF04_InvalidRowsAreNotPersisted(t *testing.T) {
	for _, kind := range []string{"alarms", "maintenance"} {
		for _, format := range []string{"csv", "xlsx"} {
			t.Run(kind+"/"+format, func(t *testing.T) {
				f := newImportFixture(t)
				rows := importRows(kind, "ZM03")
				if kind == "alarms" {
					rows[2][3] = "99" // Door outside the valid range.
				} else {
					rows[2][2], rows[2][3] = "", "" // No station or location fallback.
				}
				res := f.upload(t, kind, "mixed."+format, "ZM03", importData(t, format, rows))
				if res.Code != 202 {
					t.Fatalf("expected acceptance, got %d", res.Code)
				}
				f.process(t)
				status := f.status(t, f.queue.jobs[0].UploadID)
				if status.Status != "completed" || status.RowsProcessed != 2 || status.RowsInserted != 1 {
					t.Fatalf("invalid row accounting: %+v", status)
				}
				if kind == "alarms" {
					if len(f.store.alarms) != 1 || f.store.alarms[0].StationCode != "FAL" || f.store.alarms[0].DoorNumber != 4 {
						t.Fatal("wrong row persisted")
					}
				} else if len(f.store.maintenances) != 1 || f.store.maintenances[0].SAPOrderNumber != "1001" || f.store.maintenances[0].StationCode != "FAL" {
					t.Fatal("wrong maintenance persisted")
				}
			})
		}
	}
}

func TestRF04_OrderTypesFromFile(t *testing.T) {
	for _, format := range []string{"csv", "xlsx"} {
		t.Run(format, func(t *testing.T) {
			f := newImportFixture(t)
			rows := importRows("maintenance", "ZM01")
			rows[2][1] = "ZM02"
			third := append([]string(nil), rows[1]...)
			third[0], third[1] = "1003", "ZM03"
			rows = append(rows, third)
			// Uppercase extensions are valid; omit order_type to preserve each row's type.
			name := "mixed.CSV"
			if format == "xlsx" {
				name = "mixed.XLSX"
			}
			res := f.upload(t, "maintenance", name, "", importData(t, format, rows))
			if res.Code != 202 {
				t.Fatalf("expected acceptance, got %d", res.Code)
			}
			f.process(t)
			status := f.status(t, f.queue.jobs[0].UploadID)
			if status.Status != "completed" || status.RowsInserted != 3 || len(f.store.maintenances) != 3 {
				t.Fatalf("wrong mixed import: %+v", status)
			}
			for i, want := range []string{"ZM01", "ZM02", "ZM03"} {
				if f.store.maintenances[i].OrderType != want || f.store.maintenances[i].SAPOrderNumber != rows[i+1][0] {
					t.Fatalf("type/order mismatch: %+v", f.store.maintenances[i])
				}
			}
		})
	}
}
