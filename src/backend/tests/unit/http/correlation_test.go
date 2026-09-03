package http_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/gin-gonic/gin"
)

func TestCorrelationIDMiddleware_GeneratesNewID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpAdapter.CorrelationIDMiddleware())

	var capturedCtx context.Context
	r.GET("/test", func(c *gin.Context) {
		capturedCtx = c.Request.Context()
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respID := w.Header().Get("X-Request-ID")
	if respID == "" {
		t.Fatalf("esperava cabecalho X-Request-ID na resposta")
	}

	ctxID := httpAdapter.GetRequestID(capturedCtx)
	if ctxID != respID {
		t.Fatalf("esperava request_id no context igual a resposta: context=%s, header=%s", ctxID, respID)
	}
}

func TestCorrelationIDMiddleware_PreservesExistingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpAdapter.CorrelationIDMiddleware())

	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	expectedID := "meu-custom-request-id-12345"
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", expectedID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respID := w.Header().Get("X-Request-ID")
	if respID != expectedID {
		t.Fatalf("esperava preservar o id %s, obteve %s", expectedID, respID)
	}
}

func TestStructuredLoggerMiddleware_LogsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuf bytes.Buffer
	handler := slog.NewJSONHandler(&logBuf, nil)
	logger := slog.New(handler)

	r := gin.New()
	r.Use(httpAdapter.CorrelationIDMiddleware())
	r.Use(httpAdapter.StructuredLoggerMiddleware(logger))

	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200, obteve %d", w.Code)
	}

	logOutput := logBuf.String()
	if len(logOutput) == 0 {
		t.Fatalf("esperava saida de log em JSON")
	}

	if !bytes.Contains(logBuf.Bytes(), []byte("request_id")) {
		t.Fatalf("log deveria conter campo request_id, obteve: %s", logOutput)
	}

	if !bytes.Contains(logBuf.Bytes(), []byte("latency_ms")) {
		t.Fatalf("log deveria conter campo latency_ms, obteve: %s", logOutput)
	}
}
