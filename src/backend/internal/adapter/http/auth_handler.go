package http

import (
	"net/http"

	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type AuthHandler struct {
	loginUC *usecase.LoginUseCase
}

func NewAuthHandler(loginUC *usecase.LoginUseCase) *AuthHandler {
	return &AuthHandler{loginUC: loginUC}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondProblem(c, http.StatusBadRequest, "ERR_INVALID_PAYLOAD", "Requisicao Invalida", "Formato de dados JSON incorreto.")
		return
	}

	output, err := h.loginUC.Execute(c.Request.Context(), usecase.LoginInputDTO{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		RespondDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: output.Token,
		User: UserResponse{
			ID:    output.User.ID,
			Name:  output.User.Name,
			Email: output.User.Email,
			Role:  output.User.Role,
		},
	})
}
