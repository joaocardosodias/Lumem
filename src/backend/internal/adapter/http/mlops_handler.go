package http

import (
	"net/http"
	"strconv"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type UpdateMLConfigRequest struct {
	Threshold          float64 `json:"threshold"`
	DefaultHorizonDays int     `json:"default_horizon_days"`
}

type UpdatePredictionScheduleRequest struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"interval_minutes"`
}

type MLOpsHandler struct {
	mlopsUC   *usecase.MLOpsUseCase
	promoteUC *usecase.PromoteModelAndPublishUseCase
}

type PromoteModelResponse struct {
	Model         *domain.MLModelRun    `json:"model"`
	PredictionRun PredictionRunResponse `json:"prediction_run"`
}

func NewMLOpsHandler(mlopsUC *usecase.MLOpsUseCase, promoteUC *usecase.PromoteModelAndPublishUseCase) *MLOpsHandler {
	return &MLOpsHandler{mlopsUC: mlopsUC, promoteUC: promoteUC}
}

func (h *MLOpsHandler) GetConfig(c *gin.Context) {
	cfg, err := h.mlopsUC.GetConfig(c.Request.Context())
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *MLOpsHandler) UpdateConfig(c *gin.Context) {
	var req UpdateMLConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Requisicao Invalida", "Formato de dados JSON incorreto.")
		return
	}

	userID := c.GetString("userID")
	cfg, err := h.mlopsUC.UpdateConfig(c.Request.Context(), req.Threshold, req.DefaultHorizonDays, userID)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *MLOpsHandler) GetPredictionSchedule(c *gin.Context) {
	schedule, err := h.mlopsUC.GetPredictionSchedule(c.Request.Context())
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (h *MLOpsHandler) UpdatePredictionSchedule(c *gin.Context) {
	var req UpdatePredictionScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Requisicao Invalida", "Informe se o agendamento esta ativo e o intervalo em minutos.")
		return
	}
	schedule, err := h.mlopsUC.UpdatePredictionSchedule(
		c.Request.Context(), req.Enabled, req.IntervalMinutes, c.GetString("userID"),
	)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, schedule)
}

func (h *MLOpsHandler) GetActiveModel(c *gin.Context) {
	model, err := h.mlopsUC.GetActiveModel(c.Request.Context())
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, model)
}

func (h *MLOpsHandler) ListModelHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	runs, err := h.mlopsUC.ListModelRuns(c.Request.Context(), limit)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, runs)
}

func (h *MLOpsHandler) GetThresholdMetrics(c *gin.Context) {
	metrics, err := h.mlopsUC.GetThresholdMetrics(c.Request.Context(), c.Param("version"))
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *MLOpsHandler) TriggerRetrain(c *gin.Context) {
	userID := c.GetString("userID")
	run, err := h.mlopsUC.TriggerRetrain(c.Request.Context(), userID)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, run)
}

func (h *MLOpsHandler) ResumeTraining(c *gin.Context) {
	userID := c.GetString("userID")
	run, err := h.mlopsUC.ResumeTraining(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, run)
}

func (h *MLOpsHandler) CancelTraining(c *gin.Context) {
	job, err := h.mlopsUC.CancelTraining(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *MLOpsHandler) GetDrift(c *gin.Context) {
	drift, err := h.mlopsUC.GetDrift(c.Request.Context())
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, drift)
}

func (h *MLOpsHandler) GetTrainingJob(c *gin.Context) {
	job, err := h.mlopsUC.GetTrainingJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *MLOpsHandler) ListTrainingJobs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	jobs, err := h.mlopsUC.ListTrainingJobs(c.Request.Context(), limit)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, jobs)
}

func (h *MLOpsHandler) PromoteModel(c *gin.Context) {
	result, err := h.promoteUC.Execute(c.Request.Context(), c.Param("version"), c.GetString("userID"))
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, PromoteModelResponse{
		Model:         result.Model,
		PredictionRun: toPredictionRunResponse(result.PredictionRun),
	})
}

func (h *MLOpsHandler) GetProductionPerformance(c *gin.Context) {
	performance, err := h.mlopsUC.GetProductionPerformance(c.Request.Context())
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, performance)
}
