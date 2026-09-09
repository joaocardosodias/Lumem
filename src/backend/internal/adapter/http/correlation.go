package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type correlationKey struct{}

const RequestIDHeader = "X-Request-ID"

func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		c.Header(RequestIDHeader, reqID)
		c.Set(RequestIDHeader, reqID)

		ctx := context.WithValue(c.Request.Context(), correlationKey{}, reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(correlationKey{}).(string); ok {
		return val
	}
	return ""
}

func StructuredLoggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		reqID := c.GetString(RequestIDHeader)

		fields := []any{
			slog.String("request_id", reqID),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Float64("latency_ms", float64(latency.Nanoseconds())/1e6),
			slog.String("client_ip", c.ClientIP()),
		}
		spanContext := trace.SpanContextFromContext(c.Request.Context())
		if spanContext.IsValid() {
			fields = append(fields,
				slog.String("trace_id", spanContext.TraceID().String()),
				slog.String("span_id", spanContext.SpanID().String()),
			)
		}

		if query != "" {
			fields = append(fields, slog.String("query", query))
		}

		if len(c.Errors) > 0 {
			fields = append(fields, slog.String("errors", c.Errors.String()))
		}

		switch {
		case status >= http.StatusInternalServerError:
			logger.Error("requisicao finalizada com erro de servidor", fields...)
		case status >= http.StatusBadRequest:
			logger.Warn("requisicao finalizada com erro de cliente", fields...)
		default:
			logger.Info("requisicao finalizada com sucesso", fields...)
		}
	}
}
