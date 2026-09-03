package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/adapter/parser"
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

	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("inicializando aplicacao",
		slog.String("environment", cfg.Environment),
		slog.String("db_path", cfg.DBPath),
	)

	logger.Info("aplicando migracoes no banco de dados",
		slog.String("db_path", cfg.DBPath),
		slog.String("migrations_path", cfg.MigrationsPath),
	)
	if err := runMigrations(cfg.DBPath, cfg.MigrationsPath); err != nil {
		logger.Error("falha ao aplicar migracoes", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("migracoes aplicadas com sucesso")

	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		logger.Error("falha ao conectar no sqlite", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	userRepo := repository.NewSQLiteUserRepository(db)
	alarmRepo := repository.NewSQLiteAlarmRepository(db)
	maintRepo := repository.NewSQLiteMaintenanceRepository(db)
	insRepo := repository.NewSQLiteInspectionRepository(db)
	dashRepo := repository.NewSQLiteDashboardRepository(db)
	cachedDashRepo := repository.NewCachedDashboardRepository(dashRepo, 30*time.Second)
	predRepo := repository.NewSQLitePredictionRepository(db)

	spreadsheetParser := parser.NewDefaultSpreadsheetParser()
	mlClient := mlclient.NewClient(cfg.VPSModelURL)

	loginUC := usecase.NewLoginUseCase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	ingestAlarmsUC := usecase.NewIngestAlarmsUseCase(alarmRepo)
	ingestMaintUC := usecase.NewIngestMaintenanceUseCase(maintRepo, alarmRepo, spreadsheetParser)
	createMaintUC := usecase.NewCreateMaintenanceUseCase(maintRepo)
	insUC := usecase.NewInspectionUseCase(insRepo)
	dashUC := usecase.NewDashboardUseCase(cachedDashRepo, predRepo, mlClient)

	authHandler := httpAdapter.NewAuthHandler(loginUC)
	uploadHandler := httpAdapter.NewUploadHandler(ingestAlarmsUC, ingestMaintUC)
	maintHandler := httpAdapter.NewMaintenanceHandler(createMaintUC)
	insHandler := httpAdapter.NewInspectionHandler(insUC)
	dashHandler := httpAdapter.NewDashboardHandler(dashUC)
	healthHandler := httpAdapter.NewHealthHandler(db)

	limiter := httpAdapter.NewRateLimiter(50, 100)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpAdapter.CorrelationIDMiddleware())
	router.Use(httpAdapter.StructuredLoggerMiddleware(logger))
	router.Use(httpAdapter.RateLimiterMiddleware(limiter))

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/healthz/live", healthHandler.Live)
	router.GET("/healthz/ready", healthHandler.Ready)
	router.GET("/health", healthHandler.Ready)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/uploads/alarms", uploadHandler.UploadAlarms)
		v1.POST("/uploads/maintenance", uploadHandler.UploadMaintenance)
		v1.GET("/uploads/:id", uploadHandler.GetUploadStatus)

		v1.GET("/maintenance", maintHandler.List)
		v1.POST("/maintenance", maintHandler.Create)

		v1.POST("/inspections", insHandler.Create)
		v1.GET("/inspections", insHandler.List)
		v1.GET("/inspections/:id", insHandler.GetByID)
		v1.PATCH("/inspections/:id/status", insHandler.UpdateStatus)

		v1.GET("/dashboard/summary", dashHandler.GetSummary)
		v1.GET("/doors/:id/timeline", dashHandler.GetDoorTimeline)
		v1.GET("/stations/:code/doors/status", dashHandler.GetStationDoorsStatus)
		v1.POST("/predictions/trigger", dashHandler.TriggerPredictions)
		v1.GET("/predictions/latest", dashHandler.GetLatestPredictions)
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
		logger.Info("servidor backend iniciado", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("erro no servidor http", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("sinal de encerramento recebido. desligando servidor graciosamente...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forcando encerramento do servidor", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("servidor finalizado com sucesso")
}
