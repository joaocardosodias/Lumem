package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/gin-gonic/gin"
)

func TestRateLimiterMiddleware_AllowsUnderBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := httpAdapter.NewRateLimiter(10, 5)

	r := gin.New()
	r.Use(httpAdapter.RateLimiterMiddleware(limiter))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200 na requisicao %d, obteve %d", i+1, w.Code)
		}
	}
}

func TestRateLimiterMiddleware_BlocksOverBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := httpAdapter.NewRateLimiter(1, 2)

	r := gin.New()
	r.Use(httpAdapter.RateLimiterMiddleware(limiter))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("esperava 200 na requisicao permitida %d", i+1)
		}
	}

	reqBlocked := httptest.NewRequest(http.MethodGet, "/ping", nil)
	wBlocked := httptest.NewRecorder()
	r.ServeHTTP(wBlocked, reqBlocked)

	if wBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("esperava 429 Too Many Requests, obteve %d", wBlocked.Code)
	}

	time.Sleep(1100 * time.Millisecond)

	reqAllowed := httptest.NewRequest(http.MethodGet, "/ping", nil)
	wAllowed := httptest.NewRecorder()
	r.ServeHTTP(wAllowed, reqAllowed)

	if wAllowed.Code != http.StatusOK {
		t.Fatalf("esperava 200 apos recuperacao de tokens, obteve %d", wAllowed.Code)
	}
}
