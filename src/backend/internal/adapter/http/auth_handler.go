package http

import (
	"net/http"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	loginUC *usecase.LoginUseCase
}

func NewAuthHandler(loginUC *usecase.LoginUseCase) *AuthHandler {
	return &AuthHandler{loginUC: loginUC}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input usecase.LoginInputDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Requisicao Invalida", "Formato de dados JSON incorreto.")
		return
	}

	output, err := h.loginUC.Execute(c.Request.Context(), input)
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, output)
}
