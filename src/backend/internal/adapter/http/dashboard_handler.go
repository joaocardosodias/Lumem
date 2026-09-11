package http

import (
	"net/http"
	"strconv"

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
	ID          string `json:"id"`
	RunID       string `json:"run_id"`
	StationCode string `json:"station_code"`
	Platform    string `json:"platform"`
	DoorNumber  int    `json:"door_number"`
	WillFail    bool   `json:"will_fail"`
}

type PredictionRunResponse struct {
	ID                  string                   `json:"id"`
	HorizonDays         int                      `json:"horizon_days"`
	TotalDoorsEvaluated int                      `json:"total_doors_evaluated"`
	TotalDoorsAtRisk    int                      `json:"total_doors_at_risk"`
	CreatedAt           string                   `json:"created_at"`
	Predictions         []PredictionItemResponse `json:"predictions,omitempty"`
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
		})
	}

	return PredictionRunResponse{
		ID:                  r.ID,
		HorizonDays:         r.HorizonDays,
		TotalDoorsEvaluated: r.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    r.TotalDoorsAtRisk,
		CreatedAt:           r.CreatedAt.Format(defaultDateTimeLayout),
		Predictions:         items,
	}
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
		HorizonDays int `json:"horizon_days"`
	}
	_ = c.ShouldBindJSON(&input)

	run, err := h.uc.TriggerPredictions(c.Request.Context(), input.HorizonDays)
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
