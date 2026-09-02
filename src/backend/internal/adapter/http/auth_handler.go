package http

import (
	"errors"
	"net/http"

	"github.com/g04/backend/internal/domain"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de requisição inválido"})
		return
	}

	output, err := h.loginUC.Execute(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno do servidor"})
		}
		return
	}

	c.JSON(http.StatusOK, output)
}
