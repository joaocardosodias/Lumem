package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type CreateMaintenanceRequest struct {
	SAPOrderNumber       string  `json:"sap_order_number"`
	OrderType            string  `json:"order_type"`
	StationCode          string  `json:"station_code"`
	InstallationLocation string  `json:"installation_location"`
	StartTimestamp       string  `json:"start_timestamp"`
	EndTimestamp         string  `json:"end_timestamp"`
	DurationHours        float64 `json:"duration_hours"`
	Description          string  `json:"description"`
	Symptom              string  `json:"symptom"`
	Cause                string  `json:"cause"`
	ActionTaken          string  `json:"action_taken"`
}

type MaintenanceResponse struct {
	ID                   string    `json:"id"`
	SAPOrderNumber       string    `json:"sap_order_number,omitempty"`
	OrderType            string    `json:"order_type"`
	StationCode          string    `json:"station_code"`
	InstallationLocation string    `json:"installation_location,omitempty"`
	StartTimestamp       string    `json:"start_timestamp"`
	EndTimestamp         string    `json:"end_timestamp,omitempty"`
	DurationHours        float64   `json:"duration_hours,omitempty"`
	Description          string    `json:"description,omitempty"`
	Symptom              string    `json:"symptom,omitempty"`
	Cause                string    `json:"cause,omitempty"`
	ActionTaken          string    `json:"action_taken,omitempty"`
	Source               string    `json:"source"`
	CreatedAt            time.Time `json:"created_at"`
}

type MaintenanceHandler struct {
	createUC *usecase.CreateMaintenanceUseCase
}

func NewMaintenanceHandler(createUC *usecase.CreateMaintenanceUseCase) *MaintenanceHandler {
	return &MaintenanceHandler{createUC: createUC}
}

func toMaintenanceResponse(dto *usecase.MaintenanceOutputDTO) MaintenanceResponse {
	return MaintenanceResponse{
		ID:                   dto.ID,
		SAPOrderNumber:       dto.SAPOrderNumber,
		OrderType:            dto.OrderType,
		StationCode:          dto.StationCode,
		InstallationLocation: dto.InstallationLocation,
		StartTimestamp:       dto.StartTimestamp,
		EndTimestamp:         dto.EndTimestamp,
		DurationHours:        dto.DurationHours,
		Description:          dto.Description,
		Symptom:              dto.Symptom,
		Cause:                dto.Cause,
		ActionTaken:          dto.ActionTaken,
		Source:               dto.Source,
		CreatedAt:            dto.CreatedAt,
	}
}

func (h *MaintenanceHandler) Create(c *gin.Context) {
	var req CreateMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Dados Invalidos", "Corpo da requisicao JSON incorreto.")
		return
	}

	dto, err := h.createUC.Execute(c.Request.Context(), usecase.CreateMaintenanceInputDTO{
		SAPOrderNumber:       req.SAPOrderNumber,
		OrderType:            req.OrderType,
		StationCode:          req.StationCode,
		InstallationLocation: req.InstallationLocation,
		StartTimestamp:       req.StartTimestamp,
		EndTimestamp:         req.EndTimestamp,
		DurationHours:        req.DurationHours,
		Description:          req.Description,
		Symptom:              req.Symptom,
		Cause:                req.Cause,
		ActionTaken:          req.ActionTaken,
	})
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toMaintenanceResponse(dto))
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

	responses := make([]MaintenanceResponse, 0, len(dtos))
	for _, dto := range dtos {
		responses = append(responses, toMaintenanceResponse(dto))
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(responses),
		"data":  responses,
	})
}
