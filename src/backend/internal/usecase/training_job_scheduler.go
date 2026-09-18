package usecase

import (
	"context"
	"log/slog"
	"time"
)

func RunTrainingJobReconciler(ctx context.Context, uc *MLOpsUseCase, interval time.Duration, logger *slog.Logger) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	reconcile := func() {
		jobs, err := uc.ListTrainingJobs(ctx, 100)
		if err != nil {
			logger.Error("falha ao listar jobs de treinamento", slog.String("error", err.Error()))
			return
		}
		for index := range jobs {
			if jobs[index].Status == "failed" || jobs[index].Status == "cancelled" {
				continue
			}
			jobCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			_, err := uc.GetTrainingJob(jobCtx, jobs[index].ID)
			cancel()
			if err != nil {
				logger.Warn("falha ao reconciliar job de treinamento",
					slog.String("job_id", jobs[index].ID),
					slog.String("error", err.Error()),
				)
			}
		}
	}

	reconcile()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile()
		}
	}
}
