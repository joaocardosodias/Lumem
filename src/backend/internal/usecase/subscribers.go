package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type CacheInvalidator interface {
	Invalidate()
}

func RegisterEventSubscribers(
	bus port.EventBus,
	insRepo port.InspectionRepository,
	cache CacheInvalidator,
	logger *slog.Logger,
) {
	bus.Subscribe("DoorRiskDetected", func(ctx context.Context, e domain.Event) error {
		evt, ok := e.(domain.DoorRiskDetectedEvent)
		if !ok {
			return nil
		}

		if insRepo != nil {
			existing, _ := insRepo.List(ctx, 50, 0, "")
			alreadyExists := false
			for _, item := range existing {
				if item.PredictionID != nil && *item.PredictionID == evt.RunID && item.StationCode == evt.StationCode && item.DoorNumber == evt.DoorNumber {
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				notes := fmt.Sprintf("Ordem gerada automaticamente pela IA (Rodada #%s)", evt.RunID)
				ins, err := domain.NewInspection(
					uuid.New().String(),
					&evt.RunID,
					evt.StationCode,
					evt.Platform,
					evt.DoorNumber,
					"",
					"critical",
					notes,
				)
				if err != nil {
					if logger != nil {
						logger.Error("falha ao instanciar inspecao de dominio", slog.String("error", err.Error()))
					}
					return err
				}
				if err := insRepo.Create(ctx, ins); err != nil {
					if logger != nil {
						logger.Error("falha ao persistir inspecao automatica", slog.String("error", err.Error()))
					}
					return err
				}
			}
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
