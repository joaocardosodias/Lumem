package domain

type SpreadsheetParser interface {
	ReadRows(filePath, ext string) ([][]string, error)
}
