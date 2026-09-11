package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type CreateInspectionRequest struct {
	StationCode  string  `json:"station_code"`
	Platform     string  `json:"platform"`
	DoorNumber   int     `json:"door_number"`
	ScheduledFor string  `json:"scheduled_for"`
	Priority     string  `json:"priority"`
	Notes        string  `json:"notes"`
	PredictionID *string `json:"prediction_id"`
}

type UpdateInspectionStatusRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

type InspectionResponse struct {
	ID           string    `json:"id"`
	PredictionID *string   `json:"prediction_id,omitempty"`
	StationCode  string    `json:"station_code"`
	Platform     string    `json:"platform"`
	DoorNumber   int       `json:"door_number"`
	ScheduledFor string    `json:"scheduled_for,omitempty"`
	Priority     string    `json:"priority"`
	Status       string    `json:"status"`
	Notes        string    `json:"notes,omitempty"`
	CompletedAt  *string   `json:"completed_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type InspectionHandler struct {
	uc *usecase.InspectionUseCase
}

func NewInspectionHandler(uc *usecase.InspectionUseCase) *InspectionHandler {
	return &InspectionHandler{uc: uc}
}

func toInspectionResponse(dto *usecase.InspectionOutputDTO) InspectionResponse {
	var compStr *string
	if dto.CompletedAt != nil {
		s := dto.CompletedAt.Format("2006-01-02 15:04:05")
		compStr = &s
	}

	return InspectionResponse{
		ID:           dto.ID,
		PredictionID: dto.PredictionID,
		StationCode:  dto.StationCode,
		Platform:     dto.Platform,
		DoorNumber:   dto.DoorNumber,
		ScheduledFor: dto.ScheduledFor,
		Priority:     dto.Priority,
		Status:       dto.Status,
		Notes:        dto.Notes,
		CompletedAt:  compStr,
		CreatedAt:    dto.CreatedAt,
		UpdatedAt:    dto.UpdatedAt,
	}
}

func (h *InspectionHandler) Create(c *gin.Context) {
	var req CreateInspectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Dados Invalidos", "Corpo da requisicao JSON incorreto.")
		return
	}

	dto, err := h.uc.Create(c.Request.Context(), usecase.CreateInspectionInputDTO{
		StationCode:  req.StationCode,
		Platform:     req.Platform,
		DoorNumber:   req.DoorNumber,
		ScheduledFor: req.ScheduledFor,
		Priority:     req.Priority,
		Notes:        req.Notes,
		PredictionID: req.PredictionID,
	})
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toInspectionResponse(dto))
}

func (h *InspectionHandler) UpdateStatus(c *gin.Context) {
	idParam := strings.TrimSpace(c.Param("id"))
	if idParam == "" {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_ID", "ID Invalido", "O parametro ID e obrigatorio.")
		return
	}

	var req UpdateInspectionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Dados Invalidos", "Corpo da requisicao JSON incorreto.")
		return
	}

	dto, err := h.uc.UpdateStatus(c.Request.Context(), idParam, usecase.UpdateInspectionStatusInputDTO{
		Status: req.Status,
		Notes:  req.Notes,
	})
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, toInspectionResponse(dto))
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

	responses := make([]InspectionResponse, 0, len(dtos))
	for _, dto := range dtos {
		responses = append(responses, toInspectionResponse(dto))
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(responses),
		"data":  responses,
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

	c.JSON(http.StatusOK, toInspectionResponse(dto))
}
