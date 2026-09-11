package parser

import (
	"context"
	"errors"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/xuri/excelize/v2"
	"io"
	"strings"
	"testing"
)

type observedReader struct {
	io.Reader
	eof bool
}

func (r *observedReader) Read(b []byte) (int, error) {
	n, err := r.Reader.Read(b)
	if err == io.EOF {
		r.eof = true
	}
	return n, err
}

func TestCSVYieldsBeforeEOFAndStopsOnConsumerError(t *testing.T) {
	r := &observedReader{Reader: strings.NewReader("tag,estacao,porta,alarme,horario_alarme\n" + strings.Repeat("T,FAL,1,Falha,2026-09-01 10:00:00\n", 10000))}
	sentinel := errors.New("stop")
	count := 0
	err := NewDefaultAlarmParser().StreamAlarms(context.Background(), r, "csv", func(port.RawAlarmRecord) error {
		count++
		if r.eof {
			t.Fatal("parser buffered the entire input")
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) || count != 1 {
		t.Fatalf("consumer error ignored: %d %v", count, err)
	}
}

func TestExcelStreamsRowsAndHonorsCancellation(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	f := excelize.NewFile()
	defer f.Close()
	values := []any{"ordem", "inicio_avaria_datetime"}
	if err := f.SetSheetRow("Sheet1", "A1", &values); err != nil {
		t.Fatal(err)
	}
	for i := 2; i <= 1202; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i)
		row := []any{"ORD", "2026-09-01 10:00:00"}
		if err := f.SetSheetRow("Sheet1", cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	count := 0
	err = NewDefaultMaintenanceParser().StreamMaintenance(ctx, strings.NewReader(buf.String()), "xlsx", "ZM01", func(port.RawMaintenanceRecord) error {
		count++
		if count == 501 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || count != 501 {
		t.Fatalf("cancellation not honored: %d %v", count, err)
	}
}
