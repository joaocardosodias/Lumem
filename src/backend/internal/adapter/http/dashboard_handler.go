package http

import (
	"net/http"
	"strconv"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	uc *usecase.DashboardUseCase
}

func NewDashboardHandler(uc *usecase.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{uc: uc}
}

func (h *DashboardHandler) GetSummary(c *gin.Context) {
	summary, err := h.uc.GetSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao carregar resumo do dashboard"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *DashboardHandler) GetDoorTimeline(c *gin.Context) {
	idParam := c.Param("id")
	doorID, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de porta invalido"})
		return
	}

	timeline, err := h.uc.GetDoorTimeline(c.Request.Context(), doorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "porta nao encontrada"})
		return
	}

	c.JSON(http.StatusOK, timeline)
}

func (h *DashboardHandler) GetStationDoorsStatus(c *gin.Context) {
	code := c.Param("code")
	items, err := h.uc.GetStationDoorsStatus(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"station_code": code,
		"total_doors":  len(items),
		"doors":        items,
	})
}

func (h *DashboardHandler) TriggerPredictions(c *gin.Context) {
	var input struct {
		HorizonDays int `json:"horizon_days"`
	}
	_ = c.ShouldBindJSON(&input)

	run, err := h.uc.TriggerPredictions(c.Request.Context(), input.HorizonDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, run)
}

func (h *DashboardHandler) GetLatestPredictions(c *gin.Context) {
	run, err := h.uc.GetLatestPredictions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nenhuma previsao encontrada"})
		return
	}
	c.JSON(http.StatusOK, run)
}
