package parser

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func getFirstIndex(m map[string]int, keys ...string) (int, bool) {
	for _, k := range keys {
		if idx, ok := m[k]; ok {
			return idx, true
		}
	}
	return -1, false
}

func safeGetCell(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

// walkRows keeps only the current row in memory. Excel XML larger than 4 MiB
// is spilled to disk by Excelize; expanded workbooks are capped at 256 MiB.
func walkRows(ctx context.Context, r io.Reader, format string, yield func([]string) error) error {
	ext := strings.ToLower(filepath.Ext(format))
	if ext == "" {
		ext = "." + strings.ToLower(strings.TrimPrefix(format, "."))
	}
	if ext == ".xlsx" || ext == ".xls" {
		return walkExcelRows(ctx, r, yield)
	}
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := yield(row); err != nil {
			return err
		}
	}
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

func walkExcelRows(ctx context.Context, r io.Reader, yield func([]string) error) error {
	tmp, err := os.CreateTemp("", "ingestion-parser-*.xlsx")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := io.Copy(tmp, contextReader{ctx, r}); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	f, err := excelize.OpenFile(tmp.Name(), excelize.Options{UnzipXMLSizeLimit: 4 << 20, UnzipSizeLimit: 256 << 20})
	if err != nil {
		return err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return errors.New("planilha sem abas")
	}
	rows, err := f.Rows(sheets[0])
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		row, err := rows.Columns()
		if err != nil {
			return err
		}
		if err := yield(row); err != nil {
			return err
		}
	}
	return rows.Error()
}
