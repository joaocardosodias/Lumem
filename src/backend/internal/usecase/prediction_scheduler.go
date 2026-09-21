package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/g04/backend/internal/usecase/port"
)

func RunPredictionScheduler(
	ctx context.Context,
	repo port.MLOpsRepository,
	predictions *TriggerPredictionsUseCase,
	pollInterval time.Duration,
	logger *slog.Logger,
) {
	if pollInterval <= 0 {
		pollInterval = 30 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}

	runDuePrediction := func() {
		claimCtx, cancelClaim := context.WithTimeout(ctx, 5*time.Second)
		schedule, err := repo.ClaimDuePredictionSchedule(claimCtx)
		cancelClaim()
		if err != nil {
			logger.Error("falha ao consultar agendamento de predicao", slog.String("error", err.Error()))
			return
		}
		if schedule == nil {
			return
		}

		logger.Info("executando predicao oficial agendada",
			slog.Int("interval_minutes", schedule.IntervalMinutes),
		)
		runCtx, cancelRun := context.WithTimeout(ctx, 10*time.Minute)
		run, runErr := predictions.Execute(runCtx, 7)
		cancelRun()

		runID := ""
		status := "completed"
		errorMessage := ""
		if runErr != nil {
			status = "failed"
			errorMessage = runErr.Error()
			logger.Error("predicao oficial agendada falhou", slog.String("error", errorMessage))
		} else {
			runID = run.ID
			logger.Info("predicao oficial agendada concluida",
				slog.String("run_id", run.ID),
				slog.Int("doors_evaluated", run.TotalDoorsEvaluated),
				slog.Int("doors_at_risk", run.TotalDoorsAtRisk),
			)
		}

		completeCtx, cancelComplete := context.WithTimeout(ctx, 5*time.Second)
		err = repo.CompletePredictionScheduleRun(completeCtx, runID, status, errorMessage)
		cancelComplete()
		if err != nil {
			logger.Error("falha ao registrar resultado da predicao agendada", slog.String("error", err.Error()))
		}
	}

	runDuePrediction()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runDuePrediction()
		}
	}
}
