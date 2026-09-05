package http

import (
	"net/http"
	"strconv"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type MaintenanceHandler struct {
	createUC *usecase.CreateMaintenanceUseCase
}

func NewMaintenanceHandler(createUC *usecase.CreateMaintenanceUseCase) *MaintenanceHandler {
	return &MaintenanceHandler{createUC: createUC}
}

func (h *MaintenanceHandler) Create(c *gin.Context) {
	var input usecase.CreateMaintenanceInputDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Dados Invalidos", "Corpo da requisicao JSON incorreto.")
		return
	}

	dto, err := h.createUC.Execute(c.Request.Context(), input)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto)
}

func (h *MaintenanceHandler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	dtos, err := h.createUC.List(c.Request.Context(), limit, offset)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	if dtos == nil {
		dtos = make([]*usecase.MaintenanceOutputDTO, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(dtos),
		"data":  dtos,
	})
}
