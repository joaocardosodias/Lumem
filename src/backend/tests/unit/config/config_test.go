package config_test

import (
	"os"
	"testing"

	"github.com/g04/backend/internal/config"
)

func TestConfig_Validate(t *testing.T) {
	devCfg := &config.Config{
		Environment: "development",
	}
	if err := devCfg.Validate(); err != nil {
		t.Fatalf("expected development config to be valid without secrets: %v", err)
	}

	prodCfgNoPass := &config.Config{
		Environment: "production",
		DBPassword:  "",
		JWTSecret:   "secret",
	}
	if err := prodCfgNoPass.Validate(); err == nil {
		t.Fatalf("expected error on production config with empty db password")
	}

	prodCfgNoJWT := &config.Config{
		Environment: "production",
		DBPassword:  "password",
		JWTSecret:   "",
	}
	if err := prodCfgNoJWT.Validate(); err == nil {
		t.Fatalf("expected error on production config with empty jwt secret")
	}

	prodCfgValid := &config.Config{
		Environment: "production",
		DBPassword:  "password",
		JWTSecret:   "secret",
	}
	if err := prodCfgValid.Validate(); err != nil {
		t.Fatalf("expected production config to be valid: %v", err)
	}
}

func TestConfigValidateStorage(t *testing.T) {
	invalidDriver := &config.Config{StorageDriver: "ftp"}
	if err := invalidDriver.Validate(); err == nil {
		t.Fatal("expected invalid storage driver to fail validation")
	}

	incompleteS3 := &config.Config{StorageDriver: "s3", S3Bucket: "ingestion-uploads"}
	if err := incompleteS3.Validate(); err == nil {
		t.Fatal("expected incomplete S3 configuration to fail validation")
	}

	validS3 := &config.Config{
		StorageDriver:     "s3",
		S3Endpoint:        "http://minio:9000",
		S3Region:          "us-east-1",
		S3Bucket:          "ingestion-uploads",
		S3AccessKeyID:     "access-key",
		S3SecretAccessKey: "secret-key",
	}
	if err := validS3.Validate(); err != nil {
		t.Fatalf("expected complete S3 configuration to be valid: %v", err)
	}
}

func TestConfigValidateTelemetry(t *testing.T) {
	missingEndpoint := &config.Config{OTelEnabled: true, OTelServiceName: "api", OTelWorkerName: "worker"}
	if err := missingEndpoint.Validate(); err == nil {
		t.Fatal("expected enabled telemetry without endpoint to fail validation")
	}

	valid := &config.Config{
		OTelEnabled:     true,
		OTelEndpoint:    "collector:4317",
		OTelServiceName: "api",
		OTelWorkerName:  "worker",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected complete telemetry configuration to be valid: %v", err)
	}

	missingService := &config.Config{
		OTelEnabled:    true,
		OTelEndpoint:   "collector:4317",
		OTelWorkerName: "worker",
	}
	if err := missingService.Validate(); err == nil {
		t.Fatal("expected missing service name to fail validation")
	}

	missingWorker := &config.Config{
		OTelEnabled:     true,
		OTelEndpoint:    "collector:4317",
		OTelServiceName: "api",
	}
	if err := missingWorker.Validate(); err == nil {
		t.Fatal("expected missing worker name to fail validation")
	}
}

func TestConfig_LoadConfig_DefaultsAndEnv(t *testing.T) {
	cfg := config.LoadConfig()
	if cfg == nil {
		t.Fatalf("expected non-nil config from LoadConfig")
	}

	os.Setenv("PORT", "9999")
	os.Setenv("ENV", "production")
	os.Setenv("RUN_MIGRATIONS", "false")
	os.Setenv("DB_PASSWORD", "supersecret")
	os.Setenv("DATABASE_URL", "postgres://user:sua-senha-aqui@host:5432/db")
	os.Setenv("MIGRATIONS_PATH", "migrations")
	os.Setenv("RATE_LIMIT_RPS", "-5")
	os.Setenv("RATE_LIMIT_BURST", "0")
	os.Setenv("LOAD_TEST", "true")
	os.Setenv("RIVER_POLL_ONLY", "false")
	os.Setenv("RIVER_MAX_WORKERS", "4")
	os.Setenv("RIVER_MAX_ATTEMPTS", "8")
	os.Setenv("OTEL_ENABLED", "true")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("ENV")
		os.Unsetenv("RUN_MIGRATIONS")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("MIGRATIONS_PATH")
		os.Unsetenv("RATE_LIMIT_RPS")
		os.Unsetenv("RATE_LIMIT_BURST")
		os.Unsetenv("LOAD_TEST")
		os.Unsetenv("RIVER_POLL_ONLY")
		os.Unsetenv("RIVER_MAX_WORKERS")
		os.Unsetenv("RIVER_MAX_ATTEMPTS")
		os.Unsetenv("OTEL_ENABLED")
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}()

	customCfg := config.LoadConfig()
	if customCfg.Port != "9999" {
		t.Fatalf("expected port 9999, got %s", customCfg.Port)
	}
	if customCfg.Environment != "production" {
		t.Fatalf("expected production env")
	}
	if customCfg.RunMigrations {
		t.Fatalf("expected run migrations false")
	}
	if customCfg.RateLimitRPS != 50 || customCfg.RateLimitBurst != 100 {
		t.Fatalf("expected fallback RPS 50 and Burst 100 for invalid inputs")
	}
	if !customCfg.LoadTest {
		t.Fatalf("expected LoadTest true")
	}
	if customCfg.DatabaseURL != "postgres://user:supersecret@host:5432/db" {
		t.Fatalf("expected replaced password in database url, got %s", customCfg.DatabaseURL)
	}
	if customCfg.RiverPollOnly || customCfg.RiverMaxWorkers != 4 || customCfg.RiverMaxAttempts != 8 {
		t.Fatalf("unexpected River configuration: poll_only=%v workers=%d attempts=%d", customCfg.RiverPollOnly, customCfg.RiverMaxWorkers, customCfg.RiverMaxAttempts)
	}
	if !customCfg.OTelEnabled || customCfg.OTelEndpoint != "collector:4317" {
		t.Fatalf("unexpected OpenTelemetry configuration: enabled=%v endpoint=%q", customCfg.OTelEnabled, customCfg.OTelEndpoint)
	}

	os.Setenv("MIGRATIONS_PATH", "nonexistent_dir_for_test")
	_ = os.MkdirAll("src/backend/migrations", 0755)
	defer os.RemoveAll("src")
	cfgFallback := config.LoadConfig()
	if cfgFallback.MigrationsPath != "src/backend/migrations" {
		t.Fatalf("expected migrations path src/backend/migrations, got %s", cfgFallback.MigrationsPath)
	}
}

func TestConfig_LoadConfig_InvalidRiverNumbers(t *testing.T) {
	os.Setenv("RIVER_MAX_WORKERS", "invalido")
	os.Setenv("RIVER_MAX_ATTEMPTS", "invalido")
	defer func() {
		os.Unsetenv("RIVER_MAX_WORKERS")
		os.Unsetenv("RIVER_MAX_ATTEMPTS")
	}()

	cfg := config.LoadConfig()
	if cfg.RiverMaxWorkers != 2 {
		t.Fatalf("esperava fallback RiverMaxWorkers 2, obteve %d", cfg.RiverMaxWorkers)
	}
	if cfg.RiverMaxAttempts != 5 {
		t.Fatalf("esperava fallback RiverMaxAttempts 5, obteve %d", cfg.RiverMaxAttempts)
	}
}
