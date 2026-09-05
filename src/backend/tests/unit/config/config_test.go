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

	os.Setenv("MIGRATIONS_PATH", "nonexistent_dir_for_test")
	_ = os.MkdirAll("src/backend/migrations", 0755)
	defer os.RemoveAll("src")
	cfgFallback := config.LoadConfig()
	if cfgFallback.MigrationsPath != "src/backend/migrations" {
		t.Fatalf("expected migrations path src/backend/migrations, got %s", cfgFallback.MigrationsPath)
	}
}
