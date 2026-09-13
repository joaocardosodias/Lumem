package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func createValidToken(secret, userID, email, role string) string {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

func setupTestRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpAdapter.AuthMiddleware(secret))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.GetString("userID"),
			"role":    c.GetString("userRole"),
		})
	})
	return r
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "test-secret"
	token := createValidToken(secret, "usr_123", "teste@motiva.com.br", "operator")

	router := setupTestRouter(secret)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200, obteve %d", w.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	router := setupTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("esperava status 401 para cabecalho ausente, obteve %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	router := setupTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "TokenInvalidoSemBearer")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("esperava status 401 para formato invalido, obteve %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	fakeToken := createValidToken("secret-errado", "usr_123", "teste@motiva.com.br", "operator")

	router := setupTestRouter("secret-correto")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+fakeToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("esperava status 401 para assinatura adulterada, obteve %d", w.Code)
	}
}

func TestAuthMiddleware_UnexpectedSigningMethod(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub":   "usr_123",
		"email": "teste@motiva.com.br",
		"role":  "operator",
	})
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	router := setupTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("esperava status 401 para metodo none, obteve %d", w.Code)
	}
}

func setupRoleRouter(secret string, allowedRoles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpAdapter.AuthMiddleware(secret))
	r.POST("/engineer-only", httpAdapter.RequireRole(allowedRoles...), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestRequireRole_AllowedRole(t *testing.T) {
	secret := "test-secret"
	router := setupRoleRouter(secret, "enginner")

	token := createValidToken(secret, "usr_eng_1", "engenheiro@motiva.com.br", "enginner")
	req := httptest.NewRequest(http.MethodPost, "/engineer-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para enginner, obteve %d", w.Code)
	}
}

func TestRequireRole_ForbiddenRole(t *testing.T) {
	secret := "test-secret"
	router := setupRoleRouter(secret, "enginner")

	token := createValidToken(secret, "usr_op_1", "operador@motiva.com.br", "operator")
	req := httptest.NewRequest(http.MethodPost, "/engineer-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("esperava status 403 para operator, obteve %d", w.Code)
	}
}

func TestRequireRole_MissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/role-check", httpAdapter.RequireRole("enginner"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodPost, "/role-check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("esperava status 403 para contexto sem role, obteve %d", w.Code)
	}
}

