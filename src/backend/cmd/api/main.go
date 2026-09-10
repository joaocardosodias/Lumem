package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/g04/backend/internal/adapter/eventbus"
	httpAdapter "github.com/g04/backend/internal/adapter/http"
	"github.com/g04/backend/internal/adapter/jobqueue"
	"github.com/g04/backend/internal/adapter/mlclient"
	"github.com/g04/backend/internal/adapter/parser"
	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/adapter/security"
	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/adapter/telemetry"
	"github.com/g04/backend/internal/config"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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

func checkLiveness(ctx context.Context, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.New("healthcheck retornou status nao saudavel")
	}
	return nil
}

func checkDatabaseConnection(db *sql.DB, cfg *config.Config, logger *slog.Logger) {
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	if err := db.PingContext(pingCtx); err != nil {
		logger.Warn("aviso: conexao inicial com supabase indisponivel ou aguardando credenciais",
			slog.String("error", err.Error()),
		)
		return
	}

	logger.Info("conexao com supabase postgres estabelecida com sucesso")
	if !cfg.RunMigrations {
		logger.Info("execucao de migracoes desativada por configuracao (RUN_MIGRATIONS=false)")
		return
	}

	logger.Info("aplicando migracoes no banco de dados", slog.String("migrations_path", cfg.MigrationsPath))
	if err := runMigrations(db, cfg.MigrationsPath); err != nil {
		logger.Error("falha ao aplicar migracoes", slog.String("error", err.Error()))
		return
	}
	logger.Info("migracoes aplicadas com sucesso")
}

func initDatabase(cfg *config.Config, logger *slog.Logger) *sql.DB {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("falha ao conectar no postgres/supabase", slog.String("error", err.Error()))
		os.Exit(1)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	checkDatabaseConnection(db, cfg, logger)
	return db
}

func initIngestionStorage(ctx context.Context, cfg *config.Config) (port.IngestionStorage, error) {
	if cfg.StorageDriver != "s3" {
		return storage.NewLocalTempStorage(), nil
	}

	return storage.NewS3Storage(ctx, storage.S3Config{
		Endpoint:        cfg.S3Endpoint,
		Region:          cfg.S3Region,
		Bucket:          cfg.S3Bucket,
		AccessKeyID:     cfg.S3AccessKeyID,
		SecretAccessKey: cfg.S3SecretAccessKey,
		Prefix:          cfg.S3Prefix,
	})
}

type appHandlers struct {
	auth      *httpAdapter.AuthHandler
	upload    *httpAdapter.UploadHandler
	maint     *httpAdapter.MaintenanceHandler
	ins       *httpAdapter.InspectionHandler
	dash      *httpAdapter.DashboardHandler
	health    *httpAdapter.HealthHandler
	docs      *httpAdapter.DocsHandler
	jwtSecret string
}

func registerRoutes(router *gin.Engine, h appHandlers) {
	router.GET("/openapi.json", h.docs.OpenAPISpec)
	router.GET("/docs", h.docs.SwaggerUI)
	router.GET("/swagger", h.docs.SwaggerUI)

	router.GET("/healthz/live", h.health.Live)
	router.GET("/healthz/ready", h.health.Ready)
	router.GET("/health", h.health.Ready)
	router.GET("/metrics", httpAdapter.MetricsHandler())

	v1 := router.Group("/api/v1")
	v1.POST("/auth/login", h.auth.Login)

	protected := v1.Group("")
	protected.Use(httpAdapter.AuthMiddleware(h.jwtSecret))
	{
		protected.POST("/uploads/alarms", h.upload.UploadAlarms)
		protected.POST("/uploads/maintenance", h.upload.UploadMaintenance)
		protected.GET("/uploads/:id", h.upload.GetUploadStatus)

		protected.GET("/maintenance", h.maint.List)
		protected.POST("/maintenance", h.maint.Create)

		protected.POST("/inspections", h.ins.Create)
		protected.GET("/inspections", h.ins.List)
		protected.GET("/inspections/:id", h.ins.GetByID)
		protected.PATCH("/inspections/:id/status", h.ins.UpdateStatus)

		protected.GET("/dashboard/summary", h.dash.GetSummary)
		protected.GET("/doors/:id/timeline", h.dash.GetDoorTimeline)
		protected.GET("/stations/:code/doors/status", h.dash.GetStationDoorsStatus)
		protected.POST("/predictions/trigger", h.dash.TriggerPredictions)
		protected.GET("/predictions/latest", h.dash.GetLatestPredictions)

		protected.GET("/auth/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"id":    c.GetString("userID"),
				"email": c.GetString("userEmail"),
				"role":  c.GetString("userRole"),
			})
		})
	}
}

func buildRouter(cfg *config.Config, logger *slog.Logger, h appHandlers) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	if cfg.OTelEnabled {
		router.Use(otelgin.Middleware(cfg.OTelServiceName))
	}
	router.Use(httpAdapter.CorrelationIDMiddleware())
	router.Use(httpAdapter.StructuredLoggerMiddleware(logger))
	router.Use(httpAdapter.PrometheusMetricsMiddleware())

	if cfg.LoadTest {
		logger.Warn("modo LOAD_TEST ativado: rate limiter desativado para teste de carga")
	} else if cfg.EnableRateLimiter {
		limiter := httpAdapter.NewRateLimiter(float64(cfg.RateLimitRPS), cfg.RateLimitBurst)
		router.Use(httpAdapter.RateLimiterMiddleware(limiter))
	}

	registerRoutes(router, h)
	return router
}

func waitForUploads(uploadWG *sync.WaitGroup, logger *slog.Logger) {
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
}

func runGracefulServer(cfg *config.Config, router *gin.Engine, logger *slog.Logger) {
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

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := checkLiveness(ctx, "http://127.0.0.1:"+port+"/healthz/live"); err != nil {
			os.Exit(1)
		}
		return
	}

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

	telemetryShutdown, err := telemetry.Init(context.Background(), telemetry.Config{
		Enabled:     cfg.OTelEnabled,
		Endpoint:    cfg.OTelEndpoint,
		ServiceName: cfg.OTelServiceName,
		Environment: cfg.Environment,
		Insecure:    cfg.OTelInsecure,
	})
	if err != nil {
		logger.Error("falha ao inicializar OpenTelemetry", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryShutdown(shutdownCtx); err != nil {
			logger.Error("falha ao encerrar OpenTelemetry", slog.String("error", err.Error()))
		}
	}()

	logger.Info("inicializando aplicacao",
		slog.String("environment", cfg.Environment),
		slog.Bool("run_migrations", cfg.RunMigrations),
		slog.String("db_host", cfg.DBHost),
		slog.String("db_port", cfg.DBPort),
		slog.String("db_name", cfg.DBName),
	)

	db := initDatabase(cfg, logger)
	defer db.Close()

	if cfg.RunMigrations {
		migrationCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := jobqueue.MigrateSQL(migrationCtx, db, logger); err != nil {
			cancel()
			logger.Error("falha ao aplicar migracoes do River", slog.String("error", err.Error()))
			os.Exit(1)
		}
		cancel()
	}
	riverClient, err := jobqueue.NewInsertClient(db, logger)
	if err != nil {
		logger.Error("falha ao inicializar cliente River", slog.String("error", err.Error()))
		os.Exit(1)
	}
	ingestionQueue := jobqueue.NewRiverQueue(db, riverClient, cfg.RiverMaxAttempts)

	bus := eventbus.NewInMemoryEventBus(logger)
	uow := repository.NewPostgresUnitOfWork(db)

	userRepo := repository.NewPostgresUserRepository(db)
	alarmRepo := repository.NewPostgresAlarmRepository(db)
	maintRepo := repository.NewPostgresMaintenanceRepository(db)
	insRepo := repository.NewPostgresInspectionRepository(db)
	dashRepo := repository.NewPostgresDashboardRepository(db)
	cachedDashRepo := repository.NewCachedDashboardRepository(dashRepo, 30*time.Second)
	predRepo := repository.NewPostgresPredictionRepository(db)

	usecase.RegisterEventSubscribers(bus, cachedDashRepo, logger)

	ingestionStorage, err := initIngestionStorage(context.Background(), cfg)
	if err != nil {
		logger.Error("falha ao inicializar armazenamento de uploads", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("armazenamento de uploads inicializado",
		slog.String("driver", cfg.StorageDriver),
		slog.String("bucket", cfg.S3Bucket),
	)
	ingestionStorage = jobqueue.NewCleanupStorage(ingestionStorage, riverClient)
	alarmParser := parser.NewDefaultAlarmParser()
	maintParser := parser.NewDefaultMaintenanceParser()
	mlClient := mlclient.NewClient(cfg.VPSModelURL)

	hasher := security.NewBcryptPasswordHasher(0)
	tokenGen := security.NewJWTTokenGenerator(cfg.JWTSecret)

	loginUC := usecase.NewLoginUseCase(userRepo, hasher, tokenGen, cfg.JWTExpiration)
	ingestAlarmsUC := usecase.NewIngestAlarmsUseCaseWithQueue(alarmRepo, alarmRepo, ingestionStorage, alarmParser, ingestionQueue, uow, logger)
	ingestMaintUC := usecase.NewIngestMaintenanceUseCaseWithQueue(maintRepo, alarmRepo, ingestionStorage, maintParser, ingestionQueue, uow, logger)
	createMaintUC := usecase.NewCreateMaintenanceUseCase(maintRepo)
	insUC := usecase.NewInspectionUseCase(insRepo)
	dashUC := usecase.NewDashboardUseCase(cachedDashRepo, predRepo, mlClient, bus, uow)

	handlers := appHandlers{
		auth:      httpAdapter.NewAuthHandler(loginUC),
		upload:    httpAdapter.NewUploadHandler(ingestAlarmsUC, ingestMaintUC),
		maint:     httpAdapter.NewMaintenanceHandler(createMaintUC),
		ins:       httpAdapter.NewInspectionHandler(insUC),
		dash:      httpAdapter.NewDashboardHandler(dashUC),
		health:    httpAdapter.NewHealthHandler(db),
		docs:      httpAdapter.NewDocsHandler("docs/openapi.json"),
		jwtSecret: cfg.JWTSecret,
	}

	router := buildRouter(cfg, logger, handlers)
	runGracefulServer(cfg, router, logger)
}
