package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/config"
	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func runMigrations(dbPath, migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, "sqlite3://"+dbPath)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}

func main() {
	cfg := config.LoadConfig()

	log.Printf("Ambiente: %s | DB: %s", cfg.Environment, cfg.DBPath)
	log.Printf("Aplicando migrações em %s a partir de %s...", cfg.DBPath, cfg.MigrationsPath)
	if err := runMigrations(cfg.DBPath, cfg.MigrationsPath); err != nil {
		log.Fatalf("Falha ao aplicar migrações: %v", err)
	}
	log.Println("Migrações aplicadas com sucesso.")

	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		log.Fatalf("Falha ao conectar no SQLite: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewSQLiteUserRepository(db)
	alarmRepo := repository.NewSQLiteAlarmRepository(db)

	loginUC := usecase.NewLoginUseCase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	ingestUC := usecase.NewIngestAlarmsUseCase(alarmRepo)

	authHandler := httpAdapter.NewAuthHandler(loginUC)
	uploadHandler := httpAdapter.NewUploadHandler(ingestUC)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"service":     "backend-linha4-go",
			"environment": cfg.Environment,
			"time":        time.Now().Format(time.RFC3339),
		})
	})

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/uploads/alarms", uploadHandler.UploadAlarms)
		v1.GET("/uploads/:id", uploadHandler.GetUploadStatus)
	}

	protected := v1.Group("")
	protected.Use(httpAdapter.AuthMiddleware(cfg.JWTSecret))
	{
		protected.GET("/auth/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"id":    c.GetString("userID"),
				"email": c.GetString("userEmail"),
				"role":  c.GetString("userRole"),
			})
		})
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("Servidor backend iniciado na porta %s...", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Erro inesperado no servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Sinal de encerramento recebido. Desligando servidor graciosamente...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forçando encerramento do servidor: %v", err)
	}

	log.Println("Servidor finalizado com sucesso.")
}
