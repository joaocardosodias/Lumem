package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Environment        string
	RunMigrations      bool
	DBDriver           string
	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	DBSSLMode          string
	DatabaseURL        string
	DBPath             string
	MigrationsPath     string
	JWTSecret          string
	JWTExpiration      time.Duration
	VPSModelURL        string
	CORSAllowedOrigins string
	EnableRateLimiter  bool
	RateLimitRPS       int
	RateLimitBurst     int
	LoadTest           bool
	StorageDriver      string
	S3Endpoint         string
	S3Region           string
	S3Bucket           string
	S3AccessKeyID      string
	S3SecretAccessKey  string
	S3Prefix           string
	RiverPollOnly      bool
	RiverMaxWorkers    int
	RiverMaxAttempts   int
	OTelEnabled        bool
	OTelEndpoint       string
	OTelInsecure       bool
	OTelServiceName    string
	OTelWorkerName     string
}

func (c *Config) Validate() error {
	if c.Environment == "production" {
		if c.DBPassword == "" {
			return errors.New("DB_PASSWORD e obrigatorio em ambiente de producao")
		}
		if c.JWTSecret == "" {
			return errors.New("JWT_SECRET e obrigatorio em ambiente de producao")
		}
	}

	storageDriver := strings.ToLower(strings.TrimSpace(c.StorageDriver))
	if storageDriver == "" {
		storageDriver = "local"
	}
	if storageDriver != "local" && storageDriver != "s3" {
		return errors.New("STORAGE_DRIVER deve ser local ou s3")
	}
	if storageDriver == "s3" {
		missing := make([]string, 0, 5)
		required := []struct {
			name  string
			value string
		}{
			{name: "S3_ENDPOINT", value: c.S3Endpoint},
			{name: "S3_REGION", value: c.S3Region},
			{name: "S3_BUCKET", value: c.S3Bucket},
			{name: "S3_ACCESS_KEY_ID", value: c.S3AccessKeyID},
			{name: "S3_SECRET_ACCESS_KEY", value: c.S3SecretAccessKey},
		}
		for _, field := range required {
			if strings.TrimSpace(field.value) == "" {
				missing = append(missing, field.name)
			}
		}
		if len(missing) > 0 {
			return fmt.Errorf("configuracao S3 incompleta; variaveis obrigatorias: %s", strings.Join(missing, ", "))
		}
	}
	if c.OTelEnabled {
		if strings.TrimSpace(c.OTelEndpoint) == "" {
			return errors.New("OTEL_EXPORTER_OTLP_ENDPOINT e obrigatorio quando OTEL_ENABLED=true")
		}
		if strings.TrimSpace(c.OTelServiceName) == "" || strings.TrimSpace(c.OTelWorkerName) == "" {
			return errors.New("OTEL_SERVICE_NAME e OTEL_WORKER_SERVICE_NAME sao obrigatorios quando OTEL_ENABLED=true")
		}
	}
	return nil
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("src/backend/.env")
	}

	dbHost := getEnv("DB_HOST", "aws-0-us-west-2.pooler.supabase.com")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "postgres")
	dbUser := getEnv("DB_USER", "postgres.yqkcxaqorhekxbctewwj")
	dbPass := getEnv("DB_PASSWORD", "")
	dbSSL := getEnv("DB_SSLMODE", "require")

	defaultURL := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPass, dbHost, dbPort, dbName, dbSSL)

	rawURL := getEnv("DATABASE_URL", defaultURL)
	if strings.Contains(rawURL, "sua-senha-aqui") && dbPass != "" {
		rawURL = strings.ReplaceAll(rawURL, "sua-senha-aqui", dbPass)
	}

	migrationsPath := getEnv("MIGRATIONS_PATH", "migrations")
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		if _, err := os.Stat("src/backend/migrations"); err == nil {
			migrationsPath = "src/backend/migrations"
		}
	}

	runMig := strings.ToLower(getEnv("RUN_MIGRATIONS", "true")) == "true"
	enableRateLimiter := strings.ToLower(getEnv("ENABLE_RATE_LIMITER", "true")) == "true"
	rps, _ := strconv.Atoi(getEnv("RATE_LIMIT_RPS", "50"))
	if rps <= 0 {
		rps = 50
	}
	burst, _ := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "100"))
	if burst <= 0 {
		burst = 100
	}
	riverMaxWorkers, _ := strconv.Atoi(getEnv("RIVER_MAX_WORKERS", "2"))
	if riverMaxWorkers <= 0 {
		riverMaxWorkers = 2
	}
	riverMaxAttempts, _ := strconv.Atoi(getEnv("RIVER_MAX_ATTEMPTS", "5"))
	if riverMaxAttempts <= 0 {
		riverMaxAttempts = 5
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		Environment:        getEnv("ENV", "development"),
		RunMigrations:      runMig,
		DBDriver:           getEnv("DB_DRIVER", "postgres"),
		DBHost:             dbHost,
		DBPort:             dbPort,
		DBName:             dbName,
		DBUser:             dbUser,
		DBPassword:         dbPass,
		DBSSLMode:          dbSSL,
		DatabaseURL:        rawURL,
		DBPath:             getEnv("DB_PATH", "database.db"),
		MigrationsPath:     migrationsPath,
		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTExpiration:      24 * time.Hour,
		VPSModelURL:        getEnv("VPS_MODEL_URL", "http://localhost:8000"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
		EnableRateLimiter:  enableRateLimiter,
		RateLimitRPS:       rps,
		RateLimitBurst:     burst,
		LoadTest:           strings.ToLower(getEnv("LOAD_TEST", "false")) == "true",
		StorageDriver:      strings.ToLower(strings.TrimSpace(getEnv("STORAGE_DRIVER", "local"))),
		S3Endpoint:         getEnv("S3_ENDPOINT", ""),
		S3Region:           getEnv("S3_REGION", ""),
		S3Bucket:           getEnv("S3_BUCKET", "ingestion-uploads"),
		S3AccessKeyID:      getEnv("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey:  getEnv("S3_SECRET_ACCESS_KEY", ""),
		S3Prefix:           getEnv("S3_PREFIX", "ingestion"),
		RiverPollOnly:      strings.EqualFold(getEnv("RIVER_POLL_ONLY", "true"), "true"),
		RiverMaxWorkers:    riverMaxWorkers,
		RiverMaxAttempts:   riverMaxAttempts,
		OTelEnabled:        strings.EqualFold(getEnv("OTEL_ENABLED", "false"), "true"),
		OTelEndpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		OTelInsecure:       strings.EqualFold(getEnv("OTEL_EXPORTER_OTLP_INSECURE", "true"), "true"),
		OTelServiceName:    getEnv("OTEL_SERVICE_NAME", "g04-api"),
		OTelWorkerName:     getEnv("OTEL_WORKER_SERVICE_NAME", "g04-worker"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}
