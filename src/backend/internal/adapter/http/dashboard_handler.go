package http

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

const defaultDateTimeLayout = "2006-01-02 15:04:05"

type StationRiskResponse struct {
	StationCode string `json:"station_code"`
	DoorsAtRisk int    `json:"doors_at_risk"`
}

type DashboardSummaryResponse struct {
	TotalDoors           int                   `json:"total_doors"`
	DoorsAtRisk          int                   `json:"doors_at_risk"`
	PendingInspections   int                   `json:"pending_inspections"`
	CompletedInspections int                   `json:"completed_inspections"`
	CriticalStations     []StationRiskResponse `json:"critical_stations"`
}

type TimelineAlarmResponse struct {
	ID                     string `json:"id"`
	Tag                    string `json:"tag"`
	AlarmType              string `json:"alarm_type"`
	DurationSeconds        int    `json:"duration_seconds"`
	EventTimestamp         string `json:"event_timestamp"`
	NormalizationTimestamp string `json:"normalization_timestamp,omitempty"`
}

type TimelineMaintenanceResponse struct {
	ID             string  `json:"id"`
	OrderType      string  `json:"order_type"`
	SAPOrderNumber string  `json:"sap_order_number,omitempty"`
	StartTimestamp string  `json:"start_timestamp"`
	DurationHours  float64 `json:"duration_hours,omitempty"`
	Description    string  `json:"description,omitempty"`
	Symptom        string  `json:"symptom,omitempty"`
	Source         string  `json:"source"`
}

type TimelineInspectionResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	Notes       string  `json:"notes,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type DoorTimelineResponse struct {
	DoorID          int                           `json:"door_id"`
	StationName     string                        `json:"station_name"`
	Platform        string                        `json:"platform"`
	DoorNumber      int                           `json:"door_number"`
	SAPTag          string                        `json:"sap_tag"`
	Status          string                        `json:"status"`
	WillFail        bool                          `json:"will_fail"`
	LastEvaluatedAt *string                       `json:"last_evaluated_at,omitempty"`
	RecentAlarms    []TimelineAlarmResponse       `json:"recent_alarms"`
	Maintenances    []TimelineMaintenanceResponse `json:"maintenances"`
	Inspections     []TimelineInspectionResponse  `json:"inspections"`
}

type DoorStatusResponse struct {
	ID         int    `json:"id"`
	Platform   string `json:"platform"`
	DoorNumber int    `json:"door_number"`
	SAPTag     string `json:"sap_tag"`
	Status     string `json:"status"`
	WillFail   bool   `json:"will_fail"`
}

type PredictionItemResponse struct {
	ID          string   `json:"id"`
	RunID       string   `json:"run_id"`
	StationCode string   `json:"station_code"`
	Platform    string   `json:"platform"`
	DoorNumber  int      `json:"door_number"`
	WillFail    bool     `json:"will_fail"`
	Probability *float64 `json:"probability,omitempty"`
}

type PredictionRunResponse struct {
	ID                  string                   `json:"id"`
	HorizonDays         int                      `json:"horizon_days"`
	ThresholdUsed       *float64                 `json:"threshold_used,omitempty"`
	ModelVersion        string                   `json:"model_version,omitempty"`
	IsExperiment        bool                     `json:"is_experiment"`
	PublishedAt         *string                  `json:"published_at,omitempty"`
	TotalDoorsEvaluated int                      `json:"total_doors_evaluated"`
	TotalDoorsAtRisk    int                      `json:"total_doors_at_risk"`
	AverageProbability  *float64                 `json:"average_probability,omitempty"`
	P95Probability      *float64                 `json:"p95_probability,omitempty"`
	MaximumProbability  *float64                 `json:"maximum_probability,omitempty"`
	EstimatedPrecision  *float64                 `json:"estimated_precision,omitempty"`
	EstimatedRecall     *float64                 `json:"estimated_recall,omitempty"`
	EstimatedF1         *float64                 `json:"estimated_f1,omitempty"`
	ValidationSource    string                   `json:"validation_source,omitempty"`
	CreatedAt           string                   `json:"created_at"`
	Predictions         []PredictionItemResponse `json:"predictions,omitempty"`
}

type DoorPredictionHistoryResponse struct {
	ID            string   `json:"id"`
	RunID         string   `json:"run_id"`
	StationCode   string   `json:"station_code"`
	Platform      string   `json:"platform"`
	DoorNumber    int      `json:"door_number"`
	WillFail      bool     `json:"will_fail"`
	Probability   *float64 `json:"probability,omitempty"`
	HorizonDays   int      `json:"horizon_days"`
	ThresholdUsed *float64 `json:"threshold_used,omitempty"`
	ModelVersion  string   `json:"model_version,omitempty"`
	CreatedAt     string   `json:"created_at"`
	WindowEndsAt  string   `json:"window_ends_at"`
}

type DashboardHandler struct {
	uc *usecase.DashboardUseCase
}

func NewDashboardHandler(uc *usecase.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{uc: uc}
}

func toDashboardSummaryResponse(s *usecase.DashboardSummaryOutputDTO) DashboardSummaryResponse {
	critical := make([]StationRiskResponse, 0, len(s.CriticalStations))
	for _, c := range s.CriticalStations {
		critical = append(critical, StationRiskResponse{
			StationCode: c.StationCode,
			DoorsAtRisk: c.DoorsAtRisk,
		})
	}
	return DashboardSummaryResponse{
		TotalDoors:           s.TotalDoors,
		DoorsAtRisk:          s.DoorsAtRisk,
		PendingInspections:   s.PendingInspections,
		CompletedInspections: s.CompletedInspections,
		CriticalStations:     critical,
	}
}

func toDoorTimelineResponse(t *usecase.DoorTimelineOutputDTO) DoorTimelineResponse {
	var evalStr *string
	if t.LastEvaluatedAt != nil {
		s := t.LastEvaluatedAt.Format(defaultDateTimeLayout)
		evalStr = &s
	}

	alarms := make([]TimelineAlarmResponse, 0, len(t.RecentAlarms))
	for _, a := range t.RecentAlarms {
		alarms = append(alarms, TimelineAlarmResponse{
			ID:                     a.ID,
			Tag:                    a.Tag,
			AlarmType:              a.AlarmType,
			DurationSeconds:        a.DurationSeconds,
			EventTimestamp:         a.EventTimestamp,
			NormalizationTimestamp: a.NormalizationTimestamp,
		})
	}

	maints := make([]TimelineMaintenanceResponse, 0, len(t.Maintenances))
	for _, m := range t.Maintenances {
		maints = append(maints, TimelineMaintenanceResponse{
			ID:             m.ID,
			OrderType:      m.OrderType,
			SAPOrderNumber: m.SAPOrderNumber,
			StartTimestamp: m.StartTimestamp,
			DurationHours:  m.DurationHours,
			Description:    m.Description,
			Symptom:        m.Symptom,
			Source:         m.Source,
		})
	}

	inspections := make([]TimelineInspectionResponse, 0, len(t.Inspections))
	for _, i := range t.Inspections {
		var compStr *string
		if i.CompletedAt != nil {
			s := i.CompletedAt.Format(defaultDateTimeLayout)
			compStr = &s
		}
		inspections = append(inspections, TimelineInspectionResponse{
			ID:          i.ID,
			Status:      i.Status,
			Notes:       i.Notes,
			CompletedAt: compStr,
			CreatedAt:   i.CreatedAt.Format(defaultDateTimeLayout),
		})
	}

	return DoorTimelineResponse{
		DoorID:          t.DoorID,
		StationName:     t.StationName,
		Platform:        t.Platform,
		DoorNumber:      t.DoorNumber,
		SAPTag:          t.SAPTag,
		Status:          t.Status,
		WillFail:        t.WillFail,
		LastEvaluatedAt: evalStr,
		RecentAlarms:    alarms,
		Maintenances:    maints,
		Inspections:     inspections,
	}
}

func toPredictionRunResponse(r *usecase.PredictionRunOutputDTO) PredictionRunResponse {
	items := make([]PredictionItemResponse, 0, len(r.Predictions))
	for _, p := range r.Predictions {
		items = append(items, PredictionItemResponse{
			ID:          p.ID,
			RunID:       p.RunID,
			StationCode: p.StationCode,
			Platform:    p.Platform,
			DoorNumber:  p.DoorNumber,
			WillFail:    p.WillFail,
			Probability: p.Probability,
		})
	}

	var publishedAt *string
	if r.PublishedAt != nil {
		value := r.PublishedAt.Format(time.RFC3339)
		publishedAt = &value
	}
	return PredictionRunResponse{
		ID:                  r.ID,
		HorizonDays:         r.HorizonDays,
		ThresholdUsed:       r.ThresholdUsed,
		ModelVersion:        r.ModelVersion,
		IsExperiment:        r.IsExperiment,
		PublishedAt:         publishedAt,
		TotalDoorsEvaluated: r.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    r.TotalDoorsAtRisk,
		AverageProbability:  r.AverageProbability,
		P95Probability:      r.P95Probability,
		MaximumProbability:  r.MaximumProbability,
		EstimatedPrecision:  r.EstimatedPrecision,
		EstimatedRecall:     r.EstimatedRecall,
		EstimatedF1:         r.EstimatedF1,
		ValidationSource:    r.ValidationSource,
		CreatedAt:           r.CreatedAt.Format(defaultDateTimeLayout),
		Predictions:         items,
	}
}

func (h *DashboardHandler) RunPredictionExperiment(c *gin.Context) {
	var input struct {
		Threshold float64 `json:"threshold"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Requisicao Invalida", "Informe um threshold entre 0.05 e 0.95.")
		return
	}
	run, err := h.uc.RunPredictionExperiment(c.Request.Context(), input.Threshold)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toPredictionRunResponse(run))
}

func (h *DashboardHandler) ListPredictionExperiments(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	runs, err := h.uc.ListPredictionExperiments(c.Request.Context(), limit)
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	response := make([]PredictionRunResponse, 0, len(runs))
	for i := range runs {
		response = append(response, toPredictionRunResponse(&runs[i]))
	}
	c.JSON(http.StatusOK, response)
}

func (h *DashboardHandler) PromotePredictionExperiment(c *gin.Context) {
	run, err := h.uc.PromotePredictionExperiment(c.Request.Context(), c.Param("id"), c.GetString("userID"))
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPredictionRunResponse(run))
}

func (h *DashboardHandler) GetSummary(c *gin.Context) {
	summary, err := h.uc.GetSummary(c.Request.Context())
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDashboardSummaryResponse(summary))
}

func (h *DashboardHandler) GetDoorTimeline(c *gin.Context) {
	idParam := c.Param("id")
	doorID, err := strconv.Atoi(idParam)
	if err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_ID", "ID Invalido", "O parametro ID da porta deve ser um numero inteiro.")
		return
	}

	timeline, err := h.uc.GetDoorTimeline(c.Request.Context(), doorID)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, toDoorTimelineResponse(timeline))
}

func (h *DashboardHandler) GetStationDoorsStatus(c *gin.Context) {
	code := c.Param("code")
	items, err := h.uc.GetStationDoorsStatus(c.Request.Context(), code)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	responses := make([]DoorStatusResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, DoorStatusResponse{
			ID:         item.ID,
			Platform:   item.Platform,
			DoorNumber: item.DoorNumber,
			SAPTag:     item.SAPTag,
			Status:     item.Status,
			WillFail:   item.WillFail,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"station_code": code,
		"total_doors":  len(responses),
		"doors":        responses,
	})
}

func (h *DashboardHandler) TriggerPredictions(c *gin.Context) {
	var input struct {
		HorizonDays *int `json:"horizon_days"`
	}
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Payload Invalido", "Informe um JSON valido com horizon_days inteiro.")
		return
	}
	horizonDays := 7
	if input.HorizonDays != nil {
		if *input.HorizonDays <= 0 {
			RespondProblem(c, http.StatusBadRequest, "ERR_VALIDATION", "Horizonte Invalido", "horizon_days deve ser um numero inteiro positivo.")
			return
		}
		horizonDays = *input.HorizonDays
	}

	run, err := h.uc.TriggerPredictions(c.Request.Context(), horizonDays)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toPredictionRunResponse(run))
}

func (h *DashboardHandler) GetLatestPredictions(c *gin.Context) {
	run, err := h.uc.GetLatestPredictions(c.Request.Context())
	if err != nil {
		RespondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPredictionRunResponse(run))
}

func (h *DashboardHandler) GetDoorPredictionHistory(c *gin.Context) {
	doorID, err := strconv.Atoi(c.Param("id"))
	if err != nil || doorID <= 0 {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_ID", "ID Invalido", "O parametro ID da porta deve ser um numero inteiro positivo.")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	items, err := h.uc.GetDoorPredictionHistory(c.Request.Context(), doorID, limit)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	response := make([]DoorPredictionHistoryResponse, 0, len(items))
	for _, item := range items {
		response = append(response, DoorPredictionHistoryResponse{
			ID:            item.ID,
			RunID:         item.RunID,
			StationCode:   item.StationCode,
			Platform:      item.Platform,
			DoorNumber:    item.DoorNumber,
			WillFail:      item.WillFail,
			Probability:   item.Probability,
			HorizonDays:   item.HorizonDays,
			ThresholdUsed: item.ThresholdUsed,
			ModelVersion:  item.ModelVersion,
			CreatedAt:     item.CreatedAt.Format(time.RFC3339),
			WindowEndsAt:  item.WindowEndsAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{"history": response})
}
