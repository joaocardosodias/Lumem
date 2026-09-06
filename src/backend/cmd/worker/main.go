package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/g04/backend/internal/adapter/jobqueue"
	"github.com/g04/backend/internal/adapter/parser"
	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/adapter/storage"
	"github.com/g04/backend/internal/adapter/telemetry"
	"github.com/g04/backend/internal/config"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func ingestionStorage(ctx context.Context, cfg *config.Config) (port.IngestionStorage, error) {
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

func main() {
	cfg := config.LoadConfig()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := cfg.Validate(); err != nil {
		logger.Error("falha na validacao da configuracao", slog.String("error", err.Error()))
		os.Exit(1)
	}

	shutdownTelemetry, err := telemetry.Init(context.Background(), telemetry.Config{
		Enabled:     cfg.OTelEnabled,
		Endpoint:    cfg.OTelEndpoint,
		ServiceName: cfg.OTelWorkerName,
		Environment: cfg.Environment,
		Insecure:    cfg.OTelInsecure,
	})
	if err != nil {
		logger.Error("falha ao inicializar OpenTelemetry", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			logger.Error("falha ao encerrar OpenTelemetry", slog.String("error", err.Error()))
		}
	}()

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("falha ao configurar banco do worker", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.RiverMaxWorkers + 2)
	db.SetMaxIdleConns(2)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()
	if err := db.PingContext(startupCtx); err != nil {
		logger.Error("falha ao conectar worker ao banco", slog.String("error", err.Error()))
		os.Exit(1)
	}

	riverPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("falha ao configurar pool River", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer riverPool.Close()
	if cfg.RunMigrations {
		if err := jobqueue.Migrate(startupCtx, riverPool, logger); err != nil {
			logger.Error("falha ao aplicar migracoes River", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	storedFiles, err := ingestionStorage(context.Background(), cfg)
	if err != nil {
		logger.Error("falha ao inicializar armazenamento", slog.String("error", err.Error()))
		os.Exit(1)
	}
	uow := repository.NewPostgresUnitOfWork(db)
	uploadRepo := repository.NewPostgresAlarmRepository(db)
	alarms := usecase.NewIngestAlarmsUseCaseWithQueue(
		uploadRepo, uploadRepo, storedFiles, parser.NewDefaultAlarmParser(), nil, uow, logger,
	)
	maintenance := usecase.NewIngestMaintenanceUseCaseWithQueue(
		repository.NewPostgresMaintenanceRepository(db), uploadRepo, storedFiles,
		parser.NewDefaultMaintenanceParser(), nil, uow, logger,
	)

	worker := jobqueue.NewIngestionWorker(alarms, maintenance, logger)
	client, err := jobqueue.NewWorkerClient(riverPool, worker, jobqueue.WorkerConfig{
		MaxWorkers: cfg.RiverMaxWorkers,
		PollOnly:   cfg.RiverPollOnly,
		Logger:     logger,
	})
	if err != nil {
		logger.Error("falha ao criar worker River", slog.String("error", err.Error()))
		os.Exit(1)
	}

	runCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	if err := client.Start(runCtx); err != nil {
		logger.Error("falha ao iniciar worker River", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("worker de ingestao iniciado",
		slog.Int("max_workers", cfg.RiverMaxWorkers),
		slog.Bool("poll_only", cfg.RiverPollOnly),
	)
	<-client.Stopped()
	logger.Info("worker de ingestao encerrado")
}
