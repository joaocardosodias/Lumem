package domain

import (
	"errors"
	"time"
)

var (
	ErrUploadNotFound = errors.New("arquivo de upload nao encontrado")
)

type FileUpload struct {
	ID            string
	FileName      string
	FileType      string
	FileHash      string
	RowsProcessed int
	RowsInserted  int
	Status        string
	ErrorLog      string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}
