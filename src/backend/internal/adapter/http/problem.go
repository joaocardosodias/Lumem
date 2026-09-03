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
	case errors.Is(err, domain.ErrUserNotFound):
		RespondProblem(c, http.StatusNotFound, "ERR_NOT_FOUND", "Recurso Nao Encontrado", "O registro solicitado nao foi localizado.")
	case errors.Is(err, domain.ErrUserInactive):
		RespondProblem(c, http.StatusForbidden, "ERR_USER_INACTIVE", "Usuario Inativo", "O operador esta desativado no sistema.")
	case errors.Is(err, domain.ErrDuplicateUpload):
		RespondProblem(c, http.StatusConflict, "ERR_DUPLICATE_UPLOAD", "Arquivo Duplicado", "Arquivo ja processado anteriormente pelo sistema (idempotencia sha-256).")
	default:
		RespondProblem(c, http.StatusInternalServerError, "ERR_INTERNAL_SERVER", "Erro Interno", err.Error())
	}
}
