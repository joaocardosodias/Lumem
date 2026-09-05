package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

func TestRespondProblem_RFC7807Structure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(httpAdapter.CorrelationIDMiddleware())

	r.GET("/trigger-problem", func(c *gin.Context) {
		httpAdapter.RespondProblem(c, http.StatusBadRequest, "ERR_TEST_CODE", "Titulo de Teste", "Detalhe explicativo do erro")
	})

	req := httptest.NewRequest(http.MethodGet, "/trigger-problem", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("esperava status 400, obteve %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		t.Fatalf("esperava Content-Type application/problem+json, obteve %s", contentType)
	}

	var p httpAdapter.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("falha ao desserializar ProblemDetails: %v", err)
	}

	if p.Status != http.StatusBadRequest {
		t.Errorf("esperava status 400, obteve %d", p.Status)
	}
	if p.Code != "ERR_TEST_CODE" {
		t.Errorf("esperava ERR_TEST_CODE, obteve %s", p.Code)
	}
	if p.Title != "Titulo de Teste" {
		t.Errorf("esperava Titulo de Teste, obteve %s", p.Title)
	}
	if p.Instance != "/trigger-problem" {
		t.Errorf("esperava instance /trigger-problem, obteve %s", p.Instance)
	}
	if p.RequestID == "" {
		t.Errorf("esperava request_id preenchido")
	}
}

func TestRespondDomainError_DuplicateUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/duplicate", func(c *gin.Context) {
		httpAdapter.RespondDomainError(c, domain.ErrDuplicateUpload)
	})

	req := httptest.NewRequest(http.MethodGet, "/duplicate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("esperava status 409 para ErrDuplicateUpload, obteve %d", w.Code)
	}

	var p httpAdapter.ProblemDetails
	_ = json.Unmarshal(w.Body.Bytes(), &p)

	if p.Code != "ERR_DUPLICATE_UPLOAD" {
		t.Errorf("esperava code ERR_DUPLICATE_UPLOAD, obteve %s", p.Code)
	}
}

func TestRespondDomainError_NotFoundErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/inspection-not-found", func(c *gin.Context) {
		httpAdapter.RespondDomainError(c, domain.ErrInspectionNotFound)
	})
	r.GET("/maintenance-not-found", func(c *gin.Context) {
		httpAdapter.RespondDomainError(c, domain.ErrMaintenanceNotFound)
	})

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/inspection-not-found", nil))
	if w1.Code != http.StatusNotFound {
		t.Fatalf("esperava status 404 para ErrInspectionNotFound, obteve %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/maintenance-not-found", nil))
	if w2.Code != http.StatusNotFound {
		t.Fatalf("esperava status 404 para ErrMaintenanceNotFound, obteve %d", w2.Code)
	}
}

func TestRespondDomainError_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/validation", func(c *gin.Context) {
		httpAdapter.RespondDomainError(c, domain.ErrInvalidPlatform)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/validation", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("esperava status 400 para ErrInvalidPlatform, obteve %d", w.Code)
	}

	var p httpAdapter.ProblemDetails
	_ = json.Unmarshal(w.Body.Bytes(), &p)
	if p.Code != "ERR_VALIDATION" {
		t.Errorf("esperava code ERR_VALIDATION, obteve %s", p.Code)
	}
}
