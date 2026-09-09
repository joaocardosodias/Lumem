package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total de requisicoes HTTP processadas pela API",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duracao das requisicoes HTTP em segundos",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	rateLimitRejectionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_rejections_total",
			Help: "Total de requisicoes rejeitadas por limite de taxa",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(rateLimitRejectionsTotal)
}

func PrometheusMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" || c.Request.URL.Path == "/healthz/live" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, statusStr).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)

		if c.Writer.Status() == 429 {
			rateLimitRejectionsTotal.Inc()
		}
	}
}

func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
