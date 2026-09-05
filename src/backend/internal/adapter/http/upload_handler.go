package http

import (
	"net/http"
	"strings"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	ingestAlarmsUC *usecase.IngestAlarmsUseCase
	ingestMaintUC  *usecase.IngestMaintenanceUseCase
}

func NewUploadHandler(ingestAlarmsUC *usecase.IngestAlarmsUseCase, ingestMaintUC *usecase.IngestMaintenanceUseCase) *UploadHandler {
	return &UploadHandler{
		ingestAlarmsUC: ingestAlarmsUC,
		ingestMaintUC:  ingestMaintUC,
	}
}

func (h *UploadHandler) UploadAlarms(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_MISSING_FILE", "Arquivo Obrigatorio", "O envio do arquivo no campo multipart 'file' e obrigatorio.")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondProblem(c, http.StatusInternalServerError, "ERR_FILE_OPEN", "Falha ao Abrir Arquivo", err.Error())
		return
	}
	defer file.Close()

	uploadID, err := h.ingestAlarmsUC.StartIngestion(c.Request.Context(), fileHeader.Filename, file)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"upload_id": uploadID,
		"status":    "processing",
		"message":   "arquivo recebido com sucesso. processamento iniciado em segundo plano.",
	})
}

func (h *UploadHandler) UploadMaintenance(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_MISSING_FILE", "Arquivo Obrigatorio", "O envio do arquivo no campo multipart 'file' e obrigatorio.")
		return
	}

	orderType := c.DefaultPostForm("order_type", "")

	file, err := fileHeader.Open()
	if err != nil {
		RespondProblem(c, http.StatusInternalServerError, "ERR_FILE_OPEN", "Falha ao Abrir Arquivo", err.Error())
		return
	}
	defer file.Close()

	uploadID, err := h.ingestMaintUC.StartIngestion(c.Request.Context(), fileHeader.Filename, file, orderType)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"upload_id":  uploadID,
		"order_type": orderType,
		"status":     "processing",
		"message":    "arquivo de manutencao recebido com sucesso. processamento iniciado em segundo plano.",
	})
}

type UploadStatusResponse struct {
	ID            string  `json:"id"`
	FileName      string  `json:"file_name"`
	FileType      string  `json:"file_type"`
	Status        string  `json:"status"`
	RowsProcessed int     `json:"rows_processed"`
	RowsInserted  int     `json:"rows_inserted"`
	ErrorLog      string  `json:"error_log,omitempty"`
	CreatedAt     string  `json:"created_at"`
	CompletedAt   *string `json:"completed_at,omitempty"`
}

func (h *UploadHandler) GetUploadStatus(c *gin.Context) {
	idParam := strings.TrimSpace(c.Param("id"))
	if idParam == "" {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_ID", "ID Invalido", "O parametro ID e obrigatorio.")
		return
	}

	upload, err := h.ingestAlarmsUC.GetStatus(c.Request.Context(), idParam)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	var completedStr *string
	if upload.CompletedAt != nil {
		s := upload.CompletedAt.Format("2006-01-02 15:04:05")
		completedStr = &s
	}

	c.JSON(http.StatusOK, UploadStatusResponse{
		ID:            upload.ID,
		FileName:      upload.FileName,
		FileType:      upload.FileType,
		Status:        upload.Status,
		RowsProcessed: upload.RowsProcessed,
		RowsInserted:  upload.RowsInserted,
		ErrorLog:      upload.ErrorLog,
		CreatedAt:     upload.CreatedAt.Format("2006-01-02 15:04:05"),
		CompletedAt:   completedStr,
	})
}
