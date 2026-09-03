package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			RespondProblem(c, http.StatusUnauthorized, "ERR_MISSING_TOKEN", "Token Ausente", "Cabecalho Authorization e obrigatorio.")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			RespondProblem(c, http.StatusUnauthorized, "ERR_INVALID_TOKEN_FORMAT", "Formato Invalido", "Formato de autorizacao deve ser 'Bearer <token>'.")
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("metodo de assinatura inesperado")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			RespondProblem(c, http.StatusUnauthorized, "ERR_INVALID_TOKEN", "Token Invalido", "Token expirado ou assinatura digital invalida.")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			RespondProblem(c, http.StatusUnauthorized, "ERR_INVALID_CLAIMS", "Claims Invalidas", "Falha ao extrair informacoes do token.")
			return
		}

		userID, _ := claims["sub"].(string)
		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		c.Set("userID", userID)
		c.Set("userEmail", email)
		c.Set("userRole", role)

		c.Next()
	}
}
