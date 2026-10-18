package main

import (
	"context"
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
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestCheckLiveness(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()
	if err := checkLiveness(context.Background(), healthy.URL); err != nil {
		t.Fatalf("healthcheck saudavel retornou erro: %v", err)
	}

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()
	if err := checkLiveness(context.Background(), unhealthy.URL); err == nil {
		t.Fatal("healthcheck deveria rejeitar status nao saudavel")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := checkLiveness(ctx, healthy.URL); err == nil {
		t.Fatal("healthcheck deveria respeitar contexto cancelado")
	}
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
	if err := checkDatabaseConnection(db, cfgNoMig, logger); err != nil {
		t.Fatalf("nao esperava erro sem migracoes: %v", err)
	}

	cfgMig := &config.Config{
		RunMigrations:  true,
		MigrationsPath: "../../migrations",
	}
	if err := checkDatabaseConnection(db, cfgMig, logger); err != nil {
		t.Fatalf("nao esperava erro com migracoes: %v", err)
	}

	closedDB, _ := sql.Open("pgx", "postgres://invalido:invalido@localhost:54321/db?sslmode=disable")
	if err := checkDatabaseConnection(closedDB, cfgMig, logger); err == nil {
		t.Fatal("esperava erro para banco indisponivel")
	}
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
		Environment:       "production",
		LoadTest:          false,
		EnableRateLimiter: true,
		RateLimitRPS:      100,
		RateLimitBurst:    100,
	}

	cfgLoadTest := &config.Config{
		Environment: "development",
		LoadTest:    true,
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

	protectedEndpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/uploads/alarms"},
		{http.MethodPost, "/api/v1/uploads/maintenance"},
		{http.MethodGet, "/api/v1/uploads/upload-id"},
		{http.MethodGet, "/api/v1/maintenance"},
		{http.MethodPost, "/api/v1/maintenance"},
		{http.MethodPost, "/api/v1/inspections"},
		{http.MethodGet, "/api/v1/inspections"},
		{http.MethodGet, "/api/v1/inspections/insp-id"},
		{http.MethodPatch, "/api/v1/inspections/insp-id/status"},
		{http.MethodGet, "/api/v1/dashboard/summary"},
		{http.MethodGet, "/api/v1/doors/1/timeline"},
		{http.MethodGet, "/api/v1/stations/PIN/doors/status"},
		{http.MethodPost, "/api/v1/predictions/trigger"},
		{http.MethodGet, "/api/v1/predictions/latest"},
		{http.MethodGet, "/api/v1/ml/prediction-schedule"},
		{http.MethodPut, "/api/v1/ml/prediction-schedule"},
		{http.MethodGet, "/api/v1/auth/me"},
	}
	for _, endpoint := range protectedEndpoints {
		req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
		response := httptest.NewRecorder()
		rDev.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("esperava status 401 para %s %s sem token, obteve %d", endpoint.method, endpoint.path, response.Code)
		}
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
