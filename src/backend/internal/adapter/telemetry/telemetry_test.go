package telemetry

import (
	"context"
	"strings"
	"testing"
)

func TestInitDisabled(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{Enabled: false})
	if err != nil {
		t.Fatalf("esperava nil, obteve %v", err)
	}
	if shutdown == nil {
		t.Fatal("esperava funcao de shutdown nao nil")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("esperava shutdown sem erro, obteve %v", err)
	}
}

func TestInitValidationErrors(t *testing.T) {
	_, err := Init(context.Background(), Config{
		Enabled:  true,
		Endpoint: "",
	})
	if err == nil || !strings.Contains(err.Error(), "OTEL_EXPORTER_OTLP_ENDPOINT") {
		t.Fatalf("esperava erro de endpoint, obteve %v", err)
	}

	_, err = Init(context.Background(), Config{
		Enabled:     true,
		Endpoint:    "localhost:4317",
		ServiceName: "",
	})
	if err == nil || !strings.Contains(err.Error(), "OTEL_SERVICE_NAME") {
		t.Fatalf("esperava erro de service name, obteve %v", err)
	}
}

func TestInitEnabledInsecure(t *testing.T) {
	ctx := context.Background()
	shutdown, err := Init(ctx, Config{
		Enabled:     true,
		Endpoint:    "127.0.0.1:4317",
		ServiceName: "test-service",
		Environment: "test",
		Insecure:    true,
	})
	if err != nil {
		t.Fatalf("esperava inicializacao bem-sucedida, obteve %v", err)
	}
	if shutdown == nil {
		t.Fatal("esperava funcao de shutdown nao nil")
	}
	_ = shutdown(ctx)
}

func TestInitEnabledSecure(t *testing.T) {
	ctx := context.Background()
	shutdown, err := Init(ctx, Config{
		Enabled:     true,
		Endpoint:    "127.0.0.1:4317",
		ServiceName: "test-service",
		Environment: "production",
		Insecure:    false,
	})
	if err != nil {
		t.Fatalf("esperava inicializacao bem-sucedida, obteve %v", err)
	}
	if shutdown == nil {
		t.Fatal("esperava funcao de shutdown nao nil")
	}
	_ = shutdown(ctx)
}
