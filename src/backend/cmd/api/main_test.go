package main

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestIsOriginAllowed(t *testing.T) {
	wildcardCfg := &config.Config{
		Environment:        "production",
		CORSAllowedOrigins: "*",
	}
	if !isOriginAllowed("https://qualquer.com", wildcardCfg, []string{"*"}) {
		t.Fatalf("esperava true para wildcard")
	}

	devCfg := &config.Config{
		Environment:        "development",
		CORSAllowedOrigins: "https://app.linha4.com.br",
	}
	if !isOriginAllowed("http://localhost:3000", devCfg, []string{"https://app.linha4.com.br"}) {
		t.Fatalf("esperava true em ambiente que nao e production")
	}

	prodCfg := &config.Config{
		Environment:        "production",
		CORSAllowedOrigins: "https://app.linha4.com.br, https://painel.linha4.com.br",
	}
	allowed := []string{"https://app.linha4.com.br", " https://painel.linha4.com.br"}

	if !isOriginAllowed("https://app.linha4.com.br", prodCfg, allowed) {
		t.Fatalf("esperava true para origem cadastrada")
	}

	if !isOriginAllowed("https://painel.linha4.com.br", prodCfg, allowed) {
		t.Fatalf("esperava true para origem com espaco trim")
	}

	if isOriginAllowed("https://malicioso.com", prodCfg, allowed) {
		t.Fatalf("esperava false para origem nao cadastrada em producao")
	}
}

func TestSetupCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Environment:        "production",
		CORSAllowedOrigins: "https://app.linha4.com.br",
	}

	r := gin.New()
	r.Use(setupCORSMiddleware(cfg))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("options preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://app.linha4.com.br")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("esperava 204 No Content para OPTIONS, obteve %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "https://app.linha4.com.br" {
			t.Fatalf("cabecalho allow origin incorreto: %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("normal get with origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://app.linha4.com.br")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200, obteve %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "https://app.linha4.com.br" {
			t.Fatalf("cabecalho allow origin incorreto: %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("normal get without origin", func(t *testing.T) {
		cfgDev := &config.Config{
			Environment:        "development",
			CORSAllowedOrigins: "*",
		}
		rDev := gin.New()
		rDev.Use(setupCORSMiddleware(cfgDev))
		rDev.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		rDev.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("esperava wildcard quando origem esta vazia")
		}
	})
}

func TestWaitForUploads(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		time.Sleep(5 * time.Millisecond)
		wg.Done()
	}()

	waitForUploads(wg, logger)
}

func TestCheckDatabaseConnection_AndMigrations(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5439/postgres?sslmode=disable"
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("banco indisponivel: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("postgres indisponivel na porta 5439: %v", err)
	}

	cfgNoMig := &config.Config{
		RunMigrations: false,
	}
	checkDatabaseConnection(db, cfgNoMig, logger)

	cfgMig := &config.Config{
		RunMigrations:  true,
		MigrationsPath: "../../migrations",
	}
	checkDatabaseConnection(db, cfgMig, logger)

	closedDB, _ := sql.Open("pgx", "postgres://invalido:invalido@localhost:54321/db?sslmode=disable")
	checkDatabaseConnection(closedDB, cfgMig, logger)
	closedDB.Close()

	if err := runMigrations(db, "../../migrations"); err != nil {
		t.Fatalf("esperava sucesso ao rodar migracoes existentes: %v", err)
	}

	if err := runMigrations(db, "caminho-inexistente-de-migracao"); err == nil {
		t.Fatalf("esperava erro ao rodar migracoes de pasta invalida")
	}
}

func TestBuildRouter_AndRoutes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	jwtSecret := "test-secret-key-12345"

	cfgProd := &config.Config{
		Environment:        "production",
		CORSAllowedOrigins: "*",
		LoadTest:           false,
		EnableRateLimiter:  true,
		RateLimitRPS:       100,
		RateLimitBurst:     100,
	}

	cfgLoadTest := &config.Config{
		Environment:        "development",
		CORSAllowedOrigins: "*",
		LoadTest:           true,
	}

	handlers := appHandlers{
		auth:      httpAdapter.NewAuthHandler(nil),
		upload:    httpAdapter.NewUploadHandler(nil, nil),
		maint:     httpAdapter.NewMaintenanceHandler(nil),
		ins:       httpAdapter.NewInspectionHandler(nil),
		dash:      httpAdapter.NewDashboardHandler(nil),
		health:    httpAdapter.NewHealthHandler(nil),
		docs:      httpAdapter.NewDocsHandler("../../docs/openapi.json"),
		jwtSecret: jwtSecret,
	}

	rProd := buildRouter(cfgProd, logger, handlers)
	if rProd == nil {
		t.Fatalf("router nao deveria ser nil")
	}

	rDev := buildRouter(cfgLoadTest, logger, handlers)
	if rDev == nil {
		t.Fatalf("router nao deveria ser nil")
	}

	endpoints := []string{
		"/openapi.json",
		"/docs",
		"/swagger",
		"/healthz/live",
		"/healthz/ready",
		"/health",
		"/metrics",
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		w := httptest.NewRecorder()
		rDev.ServeHTTP(w, req)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "usr_abc",
		"email": "teste@linha4.com.br",
		"role":  "operator",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+tokenStr)
	wMe := httptest.NewRecorder()
	rDev.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para /auth/me, obteve %d", wMe.Code)
	}
}
