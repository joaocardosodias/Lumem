package http

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type DocsHandler struct {
	specPath string
}

func NewDocsHandler(specPath string) *DocsHandler {
	return &DocsHandler{specPath: specPath}
}

func (h *DocsHandler) OpenAPISpec(c *gin.Context) {
	data, err := os.ReadFile(h.specPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "especificacao openapi nao encontrada"})
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}

func (h *DocsHandler) SwaggerUI(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>Documentacao da API - Linha 4 Amarela</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; padding: 0; background: #fafafa; }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIBundle.SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
