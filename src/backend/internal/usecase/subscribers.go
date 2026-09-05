package usecase

import (
	"context"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type CacheInvalidator interface {
	Invalidate()
}

type EventLogger = port.Logger

func RegisterEventSubscribers(
	bus port.EventBus,
	cache CacheInvalidator,
	logger EventLogger,
) {
	bus.Subscribe("DoorRiskDetected", func(ctx context.Context, e domain.Event) error {
		evt, ok := e.(domain.DoorRiskDetectedEvent)
		if !ok {
			return nil
		}

		if logger != nil {
			logger.Warn("alerta de risco critico de falha detectado pela ia",
				"event", "DoorRiskDetected",
				"run_id", evt.RunID,
				"station", evt.StationCode,
				"platform", evt.Platform,
				"door", evt.DoorNumber,
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
					"event", "PredictionRunCompleted",
					"run_id", evt.RunID,
					"evaluated", evt.TotalEvaluated,
					"at_risk", evt.TotalAtRisk,
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
