package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	Environment    string
	RunMigrations  bool
	DBDriver       string
	DBHost         string
	DBPort         string
	DBName         string
	DBUser         string
	DBPassword     string
	DBSSLMode      string
	DatabaseURL    string
	DBPath         string
	MigrationsPath string
	JWTSecret      string
	JWTExpiration  time.Duration
	VPSModelURL    string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("src/backend/.env")
	}

	dbHost := getEnv("DB_HOST", "aws-0-us-west-2.pooler.supabase.com")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "postgres")
	dbUser := getEnv("DB_USER", "postgres.yqkcxaqorhekxbctewwj")
	dbPass := getEnv("DB_PASSWORD", "souoprotagonistasigma")
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

	return &Config{
		Port:           getEnv("PORT", "8080"),
		Environment:    getEnv("ENV", "development"),
		RunMigrations:  runMig,
		DBDriver:       getEnv("DB_DRIVER", "postgres"),
		DBHost:         dbHost,
		DBPort:         dbPort,
		DBName:         dbName,
		DBUser:         dbUser,
		DBPassword:     dbPass,
		DBSSLMode:      dbSSL,
		DatabaseURL:    rawURL,
		DBPath:         getEnv("DB_PATH", "database.db"),
		MigrationsPath: migrationsPath,
		JWTSecret:      getEnv("JWT_SECRET", "42e4a580bed6d8706daa25995601237c41442170d08b8ed966fc91847a730c06"),
		JWTExpiration:  24 * time.Hour,
		VPSModelURL:    getEnv("VPS_MODEL_URL", "http://localhost:8000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}
