package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type memoryUserRepo struct {
	users map[string]*domain.User
}

func (m *memoryUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *memoryUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (m *memoryUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func setupIntegrationServer(jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo := &memoryUserRepo{users: make(map[string]*domain.User)}
	hash, _ := bcrypt.GenerateFromPassword([]byte("senha123"), bcrypt.MinCost)
	repo.users["operador@motiva.com"] = &domain.User{
		ID:           "usr_int_01",
		Name:         "Operador Integracao",
		Email:        "operador@motiva.com",
		PasswordHash: string(hash),
		Role:         "operator",
		IsActive:     true,
	}

	hasher := security.NewBcryptPasswordHasher(bcrypt.MinCost)
	tokenGen := security.NewJWTTokenGenerator(jwtSecret)
	uc := usecase.NewLoginUseCase(repo, hasher, tokenGen, time.Hour)
	handler := httpAdapter.NewAuthHandler(uc)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", handler.Login)
	}

	protected := v1.Group("")
	protected.Use(httpAdapter.AuthMiddleware(jwtSecret))
	{
		protected.GET("/auth/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"id": c.GetString("userID")})
		})
	}

	return r
}

func TestIntegration_HealthEndpoint(t *testing.T) {
	r := setupIntegrationServer("secret")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}
}

func TestIntegration_FullAuthFlow(t *testing.T) {
	secret := "chave-integracao"
	r := setupIntegrationServer(secret)

	loginBody := map[string]string{
		"email":    "operador@motiva.com",
		"password": "senha123",
	}
	bodyBytes, _ := json.Marshal(loginBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("falha no login de integracao: codigo %d", w.Code)
	}

	var resp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Token == "" {
		t.Fatalf("token vazio retornado")
	}

	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+resp.Token)
	wMe := httptest.NewRecorder()

	r.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("falha ao acessar rota protegida com token gerado: %d", wMe.Code)
	}
}
