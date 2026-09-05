package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/g04/backend/internal/adapter/eventbus"
	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/adapter/parser"
	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/adapter/scheduler"
	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/config"
	"github.com/g04/backend/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func runMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsPath, "postgres", driver)
	if err != nil {
		return err
	}

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

	if err := cfg.Validate(); err != nil {
		logger.Error("falha na validacao de configuracao de producao", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("inicializando aplicacao",
		slog.String("environment", cfg.Environment),
		slog.Bool("run_migrations", cfg.RunMigrations),
		slog.String("db_host", cfg.DBHost),
		slog.String("db_port", cfg.DBPort),
		slog.String("db_name", cfg.DBName),
	)

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("falha ao conectar no postgres/supabase", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(10 * time.Minute)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(pingCtx); err != nil {
		logger.Warn("aviso: conexao inicial com supabase indisponivel ou aguardando credenciais",
			slog.String("error", err.Error()),
		)
	} else {
		logger.Info("conexao com supabase postgres estabelecida com sucesso")
		if cfg.RunMigrations {
			logger.Info("aplicando migracoes no banco de dados", slog.String("migrations_path", cfg.MigrationsPath))
			if err := runMigrations(db, cfg.MigrationsPath); err != nil {
				logger.Error("falha ao aplicar migracoes", slog.String("error", err.Error()))
			} else {
				logger.Info("migracoes aplicadas com sucesso")
			}
		} else {
			logger.Info("execucao de migracoes desativada por configuracao (RUN_MIGRATIONS=false)")
		}
	}
	pingCancel()

	bus := eventbus.NewInMemoryEventBus(logger)
	uow := repository.NewPostgresUnitOfWork(db)

	userRepo := repository.NewPostgresUserRepository(db)
	alarmRepo := repository.NewPostgresAlarmRepository(db)
	maintRepo := repository.NewPostgresMaintenanceRepository(db)
	insRepo := repository.NewPostgresInspectionRepository(db)
	dashRepo := repository.NewPostgresDashboardRepository(db)
	cachedDashRepo := repository.NewCachedDashboardRepository(dashRepo, 30*time.Second)
	predRepo := repository.NewPostgresPredictionRepository(db)

	if err := alarmRepo.ReconcileStaleUploads(context.Background()); err != nil {
		logger.Warn("falha ao reconciliar uploads pendentes", slog.String("error", err.Error()))
	} else {
		logger.Info("reconciliacao de uploads pendentes executada com sucesso")
	}

	usecase.RegisterEventSubscribers(bus, insRepo, cachedDashRepo, logger)

	ingestionStorage := storage.NewLocalTempStorage()
	alarmParser := parser.NewDefaultAlarmParser()
	maintParser := parser.NewDefaultMaintenanceParser()
	mlClient := mlclient.NewClient(cfg.VPSModelURL)

	hasher := security.NewBcryptPasswordHasher(0)
	tokenGen := security.NewJWTTokenGenerator(cfg.JWTSecret)

	uploadWG := &sync.WaitGroup{}
	jobRunner := scheduler.NewGoroutineJobRunner(uploadWG, logger)

	loginUC := usecase.NewLoginUseCase(userRepo, hasher, tokenGen, cfg.JWTExpiration)
	ingestAlarmsUC := usecase.NewIngestAlarmsUseCase(alarmRepo, alarmRepo, ingestionStorage, alarmParser, jobRunner)
	ingestMaintUC := usecase.NewIngestMaintenanceUseCase(maintRepo, alarmRepo, ingestionStorage, maintParser, jobRunner)
	createMaintUC := usecase.NewCreateMaintenanceUseCase(maintRepo)
	insUC := usecase.NewInspectionUseCase(insRepo)
	dashUC := usecase.NewDashboardUseCase(cachedDashRepo, predRepo, mlClient, bus, uow)

	authHandler := httpAdapter.NewAuthHandler(loginUC)
	uploadHandler := httpAdapter.NewUploadHandler(ingestAlarmsUC, ingestMaintUC)
	maintHandler := httpAdapter.NewMaintenanceHandler(createMaintUC)
	insHandler := httpAdapter.NewInspectionHandler(insUC)
	dashHandler := httpAdapter.NewDashboardHandler(dashUC)
	healthHandler := httpAdapter.NewHealthHandler(db)
	docsHandler := httpAdapter.NewDocsHandler("docs/openapi.json")

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpAdapter.CorrelationIDMiddleware())
	router.Use(httpAdapter.StructuredLoggerMiddleware(logger))
	router.Use(httpAdapter.PrometheusMetricsMiddleware())
	if cfg.LoadTest {
		logger.Warn("modo LOAD_TEST ativado: rate limiter desativado para teste de carga")
	} else if cfg.EnableRateLimiter {
		limiter := httpAdapter.NewRateLimiter(float64(cfg.RateLimitRPS), cfg.RateLimitBurst)
		router.Use(httpAdapter.RateLimiterMiddleware(limiter))
	}

	allowedOrigins := strings.Split(cfg.CORSAllowedOrigins, ",")
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allow := false
		if cfg.CORSAllowedOrigins == "*" || cfg.Environment != "production" {
			allow = true
		} else {
			for _, o := range allowedOrigins {
				if strings.TrimSpace(o) == origin {
					allow = true
					break
				}
			}
		}

		if allow {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/openapi.json", docsHandler.OpenAPISpec)
	router.GET("/docs", docsHandler.SwaggerUI)
	router.GET("/swagger", docsHandler.SwaggerUI)

	router.GET("/healthz/live", healthHandler.Live)
	router.GET("/healthz/ready", healthHandler.Ready)
	router.GET("/health", healthHandler.Ready)
	router.GET("/metrics", httpAdapter.MetricsHandler())

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

	uploadDone := make(chan struct{})
	go func() {
		uploadWG.Wait()
		close(uploadDone)
	}()

	select {
	case <-uploadDone:
		logger.Info("todos os uploads assincronos foram concluidos")
	case <-time.After(10 * time.Second):
		logger.Warn("tempo limite atingido aguardando uploads pendentes")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forcando encerramento do servidor", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("servidor finalizado com sucesso")
}
