package http

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestApplyClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	want := jwt.MapClaims{"sub": "user-1"}
	applyClaims(ctx, &jwt.Token{Claims: want})
	if ctx.GetString("userID") != "user-1" {
		t.Fatalf("claims validas nao foram aplicadas: userID=%q", ctx.GetString("userID"))
	}

	invalidCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	invalidCtx.Request = httptest.NewRequest("GET", "/protected", nil)
	applyClaims(invalidCtx, &jwt.Token{Claims: jwt.RegisteredClaims{}})
	if !invalidCtx.IsAborted() {
		t.Fatal("claims de tipo inesperado deveriam abortar a requisicao")
	}
}
