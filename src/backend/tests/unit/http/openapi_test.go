package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/gin-gonic/gin"
)

func TestDocsHandler_OpenAPISpec(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := httpAdapter.NewDocsHandler("../../../docs/openapi.json")

	r := gin.New()
	r.GET("/openapi.json", handler.OpenAPISpec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para /openapi.json, obteve %d", w.Code)
	}

	var spec map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("falha ao interpretar openapi.json: %v", err)
	}

	if spec["openapi"] != "3.0.3" {
		t.Errorf("esperava versao 3.0.3, obteve %v", spec["openapi"])
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Fatalf("esperava endpoints definidos em paths")
	}

	if _, exists := paths["/api/v1/dashboard/summary"]; !exists {
		t.Errorf("endpoint /api/v1/dashboard/summary deveria estar documentado")
	}
}

func TestDocsHandler_SwaggerUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := httpAdapter.NewDocsHandler("../../../docs/openapi.json")

	r := gin.New()
	r.GET("/docs", handler.SwaggerUI)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava status 200 para /docs, obteve %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("esperava content-type html, obteve %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "SwaggerUIBundle") {
		t.Fatalf("pagina deveria carregar o script SwaggerUIBundle")
	}
}
