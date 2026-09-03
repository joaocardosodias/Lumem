package http

import (
	"net/http"
	"strconv"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type InspectionHandler struct {
	uc *usecase.InspectionUseCase
}

func NewInspectionHandler(uc *usecase.InspectionUseCase) *InspectionHandler {
	return &InspectionHandler{uc: uc}
}

func (h *InspectionHandler) Create(c *gin.Context) {
	var input usecase.CreateInspectionInputDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados invalidos na requisicao"})
		return
	}

	dto, err := h.uc.Create(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto)
}

func (h *InspectionHandler) UpdateStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var input usecase.UpdateInspectionStatusInputDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados invalidos na requisicao"})
		return
	}

	dto, err := h.uc.UpdateStatus(c.Request.Context(), id, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

func (h *InspectionHandler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	status := c.Query("status")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	dtos, err := h.uc.List(c.Request.Context(), limit, offset, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar inspecoes"})
		return
	}

	if dtos == nil {
		dtos = make([]*usecase.InspectionOutputDTO, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(dtos),
		"data":  dtos,
	})
}

func (h *InspectionHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	dto, err := h.uc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "inspecao nao encontrada"})
		return
	}

	c.JSON(http.StatusOK, dto)
}
