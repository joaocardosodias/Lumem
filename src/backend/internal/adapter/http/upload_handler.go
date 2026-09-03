package http

import (
	"net/http"
	"strconv"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo obrigatorio no campo 'file'"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo enviado"})
		return
	}
	defer file.Close()

	uploadID, err := h.ingestAlarmsUC.StartIngestion(c.Request.Context(), fileHeader.Filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo obrigatorio no campo 'file'"})
		return
	}

	orderType := c.DefaultPostForm("order_type", "")

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo enviado"})
		return
	}
	defer file.Close()

	uploadID, err := h.ingestMaintUC.StartIngestion(c.Request.Context(), fileHeader.Filename, file, orderType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"upload_id":  uploadID,
		"order_type": orderType,
		"status":     "processing",
		"message":    "arquivo de manutencao recebido com sucesso. processamento iniciado em segundo plano.",
	})
}

func (h *UploadHandler) GetUploadStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	upload, err := h.ingestAlarmsUC.GetStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "upload nao encontrado"})
		return
	}

	c.JSON(http.StatusOK, upload)
}
