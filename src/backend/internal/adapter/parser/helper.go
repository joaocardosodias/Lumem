package parser

import (
	"encoding/csv"
	"io"
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

func readRowsFromReader(r io.Reader, format string) ([][]string, error) {
	ext := strings.ToLower(filepath.Ext(format))
	if ext == "" {
		ext = "." + strings.ToLower(strings.TrimPrefix(format, "."))
	}
	if ext == ".xlsx" || ext == ".xls" {
		f, err := excelize.OpenReader(r)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		sheetList := f.GetSheetList()
		var sheetName string
		if len(sheetList) > 0 {
			sheetName = sheetList[0]
		}

		return f.GetRows(sheetName)
	}

	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}
