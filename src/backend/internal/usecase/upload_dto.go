package usecase

import "time"

type UploadStatusOutputDTO struct {
	ID            string
	FileName      string
	FileType      string
	Status        string
	RowsProcessed int
	RowsInserted  int
	ErrorLog      string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}
