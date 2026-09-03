package usecase

type UploadStatusOutputDTO struct {
	ID            int64   `json:"id"`
	FileName      string  `json:"file_name"`
	FileType      string  `json:"file_type"`
	Status        string  `json:"status"`
	RowsProcessed int     `json:"rows_processed"`
	RowsInserted  int     `json:"rows_inserted"`
	ErrorLog      string  `json:"error_log,omitempty"`
	CreatedAt     string  `json:"created_at"`
	CompletedAt   *string `json:"completed_at,omitempty"`
}
