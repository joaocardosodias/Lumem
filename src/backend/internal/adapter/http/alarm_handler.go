package http

import (
	"net/http"
	"strconv"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type AlarmHandler struct {
	listUC *usecase.ListAlarmsUseCase
}

func NewAlarmHandler(listUC *usecase.ListAlarmsUseCase) *AlarmHandler {
	return &AlarmHandler{listUC: listUC}
}

func (h *AlarmHandler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	station := c.Query("station_code")
	platform := c.Query("platform")
	doorStr := c.Query("door_number")
	alarmType := c.Query("alarm_type")
	tag := c.Query("tag")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	doorNumber, _ := strconv.Atoi(doorStr)

	dtos, err := h.listUC.Execute(c.Request.Context(), usecase.ListAlarmsInputDTO{
		StationCode: station,
		Platform:    platform,
		DoorNumber:  doorNumber,
		AlarmType:   alarmType,
		Tag:         tag,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(dtos),
		"data":  dtos,
	})
}
