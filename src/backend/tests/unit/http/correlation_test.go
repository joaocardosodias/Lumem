package http_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
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

func TestGetRequestID_NilAndEmpty(t *testing.T) {
	if got := httpAdapter.GetRequestID(nil); got != "" {
		t.Fatalf("esperava vazio para context nil, obteve %s", got)
	}
	if got := httpAdapter.GetRequestID(context.Background()); got != "" {
		t.Fatalf("esperava vazio para context sem valor, obteve %s", got)
	}
}

func TestStructuredLoggerMiddleware_WarnAndErrorLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuf bytes.Buffer
	handler := slog.NewJSONHandler(&logBuf, nil)
	logger := slog.New(handler)

	r := gin.New()
	r.Use(httpAdapter.CorrelationIDMiddleware())
	r.Use(httpAdapter.StructuredLoggerMiddleware(logger))

	r.GET("/client-err", func(c *gin.Context) {
		_ = c.Error(errors.New("client error occurred"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})
	r.GET("/server-err", func(c *gin.Context) {
		_ = c.Error(errors.New("server error occurred"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	})

	req1 := httptest.NewRequest(http.MethodGet, "/client-err?foo=bar", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusBadRequest {
		t.Fatalf("esperava status 400, obteve %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/server-err", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("esperava status 500, obteve %d", w2.Code)
	}

	logStr := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("requisicao finalizada com erro de cliente")) {
		t.Fatalf("esperava log de warn para 400, obteve %s", logStr)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("requisicao finalizada com erro de servidor")) {
		t.Fatalf("esperava log de error para 500, obteve %s", logStr)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte("foo=bar")) {
		t.Fatalf("esperava query no log, obteve %s", logStr)
	}
}

func TestStructuredLoggerMiddleware_WithTraceSpan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuf bytes.Buffer
	handler := slog.NewJSONHandler(&logBuf, nil)
	logger := slog.New(handler)

	r := gin.New()
	r.Use(httpAdapter.CorrelationIDMiddleware())
	r.Use(httpAdapter.StructuredLoggerMiddleware(logger))

	r.GET("/traced", func(c *gin.Context) {
		c.String(http.StatusOK, "traced")
	})

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	req := httptest.NewRequest(http.MethodGet, "/traced", nil)
	req = req.WithContext(trace.ContextWithSpanContext(req.Context(), spanContext))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200, obteve %d", w.Code)
	}
	logStr := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("trace_id")) || !bytes.Contains(logBuf.Bytes(), []byte("span_id")) {
		t.Fatalf("esperava trace_id e span_id no log, obteve: %s", logStr)
	}
}
