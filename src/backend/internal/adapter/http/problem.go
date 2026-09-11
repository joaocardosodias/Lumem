package http

import (
	"errors"
	"net/http"

	"github.com/g04/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type ProblemDetails struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

func RespondProblem(c *gin.Context, status int, code, title, detail string) {
	reqID := GetRequestID(c.Request.Context())
	if reqID == "" {
		reqID = c.GetString(RequestIDHeader)
	}

	problem := ProblemDetails{
		Type:      "https://motiva.com.br/errors/" + code,
		Title:     title,
		Status:    status,
		Detail:    detail,
		Instance:  c.Request.URL.Path,
		Code:      code,
		RequestID: reqID,
	}

	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, problem)
}

func RespondDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		RespondProblem(c, http.StatusUnauthorized, "ERR_INVALID_CREDENTIALS", "Credenciais Invalidas", "Email ou senha incorretos.")
	case errors.Is(err, domain.ErrPredictionServiceUnavailable):
		RespondProblem(c, http.StatusServiceUnavailable, "ERR_PREDICTION_UNAVAILABLE", "Servico de Predicao Indisponivel", "O servico de inferencia preditiva esta indisponivel.")
	case errors.Is(err, domain.ErrUserNotFound):
		RespondProblem(c, http.StatusNotFound, "ERR_NOT_FOUND", "Recurso Nao Encontrado", "O registro solicitado nao foi localizado.")
	case errors.Is(err, domain.ErrInspectionNotFound):
		RespondProblem(c, http.StatusNotFound, "ERR_NOT_FOUND", "Inspecao Nao Encontrada", "A inspecao solicitada nao foi localizada.")
	case errors.Is(err, domain.ErrMaintenanceNotFound):
		RespondProblem(c, http.StatusNotFound, "ERR_NOT_FOUND", "Manutencao Nao Encontrada", "O registro de manutencao solicitado nao foi localizado.")
	case errors.Is(err, domain.ErrUploadNotFound):
		RespondProblem(c, http.StatusNotFound, "ERR_NOT_FOUND", "Upload Nao Encontrado", "O upload solicitado nao foi localizado.")
	case errors.Is(err, domain.ErrDoorNotFound):
		RespondProblem(c, http.StatusNotFound, "ERR_NOT_FOUND", "Porta Nao Encontrada", "A porta solicitada nao foi localizada.")
	case errors.Is(err, domain.ErrNoPredictions):
		RespondProblem(c, http.StatusNotFound, "ERR_NOT_FOUND", "Sem Previsoes", "Nenhuma rodada de previsao calculada ate o momento.")
	case errors.Is(err, domain.ErrUserInactive):
		RespondProblem(c, http.StatusForbidden, "ERR_USER_INACTIVE", "Usuario Inativo", "O operador esta desativado no sistema.")
	case errors.Is(err, domain.ErrDuplicateUpload):
		RespondProblem(c, http.StatusConflict, "ERR_DUPLICATE_UPLOAD", "Arquivo Duplicado", "Arquivo ja processado anteriormente pelo sistema.")
	case errors.Is(err, domain.ErrEmailPasswordRequired),
		errors.Is(err, domain.ErrStationRequired),
		errors.Is(err, domain.ErrInvalidDoorID),
		errors.Is(err, domain.ErrNoActiveDoors),
		errors.Is(err, domain.ErrStationCodeRequired),
		errors.Is(err, domain.ErrInvalidPlatform),
		errors.Is(err, domain.ErrInvalidDoorNumber),
		errors.Is(err, domain.ErrInvalidInspectionStatus),
		errors.Is(err, domain.ErrInspectionIDRequired),
		errors.Is(err, domain.ErrOrderTypeRequired),
		errors.Is(err, domain.ErrMaintenanceStationRequired),
		errors.Is(err, domain.ErrStartTimestampRequired),
		errors.Is(err, domain.ErrInvalidStatusTransition):
		RespondProblem(c, http.StatusBadRequest, "ERR_VALIDATION", "Falha de Validacao", err.Error())
	default:
		RespondProblem(c, http.StatusInternalServerError, "ERR_INTERNAL_SERVER", "Erro Interno", err.Error())
	}
}
