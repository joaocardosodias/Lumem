package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/gin-gonic/gin"
)

type mockHealthyDB struct{}

func (m *mockHealthyDB) PingContext(ctx context.Context) error {
	return nil
}

type mockUnhealthyDB struct{}

func (m *mockUnhealthyDB) PingContext(ctx context.Context) error {
	return errors.New("conexao recusada pelo banco")
}

func TestHealthHandler_Live(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := httpAdapter.NewHealthHandler(&mockHealthyDB{})

	r := gin.New()
	r.GET("/healthz/live", h.Live)

	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para liveness probe, obteve %d", w.Code)
	}
}

func TestHealthHandler_Ready_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := httpAdapter.NewHealthHandler(&mockHealthyDB{})

	r := gin.New()
	r.GET("/healthz/ready", h.Ready)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200 quando dependencias estao ativas, obteve %d", w.Code)
	}
}

func TestHealthHandler_Ready_DatabaseDown(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := httpAdapter.NewHealthHandler(&mockUnhealthyDB{})

	r := gin.New()
	r.GET("/healthz/ready", h.Ready)

	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava status 503 quando banco esta inativo, obteve %d", w.Code)
	}
}
