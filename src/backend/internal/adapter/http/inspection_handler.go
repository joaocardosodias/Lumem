package http

import (
	"net/http"
	"strconv"
	"strings"

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
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Dados Invalidos", "Corpo da requisicao JSON incorreto.")
		return
	}

	dto, err := h.uc.Create(c.Request.Context(), input)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto)
}

func (h *InspectionHandler) UpdateStatus(c *gin.Context) {
	idParam := strings.TrimSpace(c.Param("id"))
	if idParam == "" {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_ID", "ID Invalido", "O parametro ID e obrigatorio.")
		return
	}

	var input usecase.UpdateInspectionStatusInputDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Dados Invalidos", "Corpo da requisicao JSON incorreto.")
		return
	}

	dto, err := h.uc.UpdateStatus(c.Request.Context(), idParam, input)
	if err != nil {
		RespondDomainError(c, err)
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
		RespondDomainError(c, err)
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
	idParam := strings.TrimSpace(c.Param("id"))
	if idParam == "" {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_ID", "ID Invalido", "O parametro ID e obrigatorio.")
		return
	}

	dto, err := h.uc.GetByID(c.Request.Context(), idParam)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto)
}
