package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	Environment    string 
	DBPath         string
	MigrationsPath string
	JWTSecret      string
	JWTExpiration  time.Duration
	VPSModelURL    string 
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: arquivo .env nao encontrado, utilizando variaveis de ambiente do sistema ou padroes.")
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		Environment:    getEnv("ENV", "development"),
		DBPath:         getEnv("DB_PATH", "database.db"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "migrations"),
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
