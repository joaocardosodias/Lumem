package usecase

import (
	"context"
	"log/slog"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type CacheInvalidator interface {
	Invalidate()
}

func RegisterEventSubscribers(
	bus port.EventBus,
	cache CacheInvalidator,
	logger *slog.Logger,
) {
	bus.Subscribe("DoorRiskDetected", func(ctx context.Context, e domain.Event) error {
		evt, ok := e.(domain.DoorRiskDetectedEvent)
		if !ok {
			return nil
		}

		if logger != nil {
			logger.Warn("alerta de risco critico de falha detectado pela ia",
				slog.String("event", "DoorRiskDetected"),
				slog.String("run_id", evt.RunID),
				slog.String("station", evt.StationCode),
				slog.String("platform", evt.Platform),
				slog.Int("door", evt.DoorNumber),
			)
		}
		return nil
	})

	bus.Subscribe("PredictionRunCompleted", func(ctx context.Context, e domain.Event) error {
		if cache != nil {
			cache.Invalidate()
		}
		if logger != nil {
			evt, ok := e.(domain.PredictionRunCompletedEvent)
			if ok {
				logger.Info("rodada de predicao concluida",
					slog.String("event", "PredictionRunCompleted"),
					slog.String("run_id", evt.RunID),
					slog.Int("evaluated", evt.TotalEvaluated),
					slog.Int("at_risk", evt.TotalAtRisk),
				)
			}
		}
		return nil
	})

	bus.Subscribe("FileUploadCompleted", func(ctx context.Context, e domain.Event) error {
		if cache != nil {
			cache.Invalidate()
		}
		return nil
	})
}
