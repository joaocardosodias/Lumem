package parser

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestHelper_Internal(t *testing.T) {
	if safeGetCell([]string{"val"}, -1) != "" {
		t.Fatalf("expected empty string for negative index")
	}
	if safeGetCell([]string{"val"}, 5) != "" {
		t.Fatalf("expected empty string for out of bounds index")
	}
	if safeGetCell([]string{"  trimmed  "}, 0) != "trimmed" {
		t.Fatalf("expected trimmed value")
	}

	if safeColumnValue([]string{"val"}, false, 0) != "" {
		t.Fatalf("expected empty string when has is false")
	}
	if safeColumnValue([]string{"val"}, true, 0) != "val" {
		t.Fatalf("expected val when has is true")
	}

	if parseDuration([]string{"val"}, false, 0) != 0.0 {
		t.Fatalf("expected 0.0 when has is false")
	}
	if parseDuration([]string{"2,5"}, true, 0) != 2.5 {
		t.Fatalf("expected 2.5 parsed duration")
	}

	f := excelize.NewFile()
	_ = f.SetCellValue("Sheet1", "A1", "header")
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("failed to write excel")
	}

	rows, err := readRowsFromReader(bytes.NewReader(buf.Bytes()), "test.xls")
	if err != nil {
		t.Fatalf("failed to read xls: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected rows from xls")
	}

	rowsNoDot, err := readRowsFromReader(bytes.NewReader(buf.Bytes()), "xlsx")
	if err != nil || len(rowsNoDot) == 0 {
		t.Fatalf("expected rows from format without dot")
	}

	csvRows, err := readRowsFromReader(strings.NewReader("a,b\n1,2"), "csv")
	if err != nil || len(csvRows) != 2 {
		t.Fatalf("expected csv rows")
	}

	f2 := excelize.NewFile()
	f2.NewSheet("CustomSheet")
	_ = f2.SetCellValue("CustomSheet", "A1", "custom_val")
	f2.DeleteSheet("Sheet1")
	var buf2 bytes.Buffer
	_ = f2.Write(&buf2)
	rows2, err := readRowsFromReader(bytes.NewReader(buf2.Bytes()), ".xlsx")
	if err != nil || len(rows2) == 0 {
		t.Fatalf("expected rows from custom sheet")
	}
}
