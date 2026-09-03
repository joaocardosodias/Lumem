package http

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthChecker interface {
	PingContext(ctx context.Context) error
}

type HealthHandler struct {
	db        HealthChecker
	startTime time.Time
}

func NewHealthHandler(db HealthChecker) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
	}
}

func (h *HealthHandler) Live(c *gin.Context) {
	uptimeSeconds := time.Since(h.startTime).Seconds()
	c.JSON(http.StatusOK, gin.H{
		"status":         "live",
		"service":        "backend-linha4-go",
		"uptime_seconds": uptimeSeconds,
		"timestamp":      time.Now().Format(time.RFC3339),
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string)
	isReady := true

	if h.db != nil {
		if err := h.db.PingContext(ctx); err != nil {
			checks["database"] = "down: " + err.Error()
			isReady = false
		} else {
			checks["database"] = "up"
		}
	} else {
		checks["database"] = "down: sem conexao configurada"
		isReady = false
	}

	tmpFile, err := os.CreateTemp("", "health_check_*")
	if err != nil {
		checks["storage"] = "down: " + err.Error()
		isReady = false
	} else {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		checks["storage"] = "writable"
	}

	statusStr := "ready"
	httpCode := http.StatusOK
	if !isReady {
		statusStr = "unready"
		httpCode = http.StatusServiceUnavailable
	}

	c.JSON(httpCode, gin.H{
		"status":    statusStr,
		"service":   "backend-linha4-go",
		"checks":    checks,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
