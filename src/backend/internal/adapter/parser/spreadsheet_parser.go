package parser

import (
	"encoding/csv"
	"os"

	"github.com/g04/backend/internal/domain"
	"github.com/xuri/excelize/v2"
)

var _ domain.SpreadsheetParser = (*DefaultSpreadsheetParser)(nil)

type DefaultSpreadsheetParser struct{}

func NewDefaultSpreadsheetParser() *DefaultSpreadsheetParser {
	return &DefaultSpreadsheetParser{}
}

func (p *DefaultSpreadsheetParser) ReadRows(filePath, ext string) ([][]string, error) {
	if ext == ".xlsx" || ext == ".xls" {
		f, err := excelize.OpenFile(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		sheetName := f.GetSheetName(0)
		if sheetName == "" {
			sheetList := f.GetSheetList()
			if len(sheetList) > 0 {
				sheetName = sheetList[0]
			}
		}

		return f.GetRows(sheetName)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	return reader.ReadAll()
}
