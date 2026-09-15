package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/g04/backend/internal/domain"
)

type modelPromotionService interface {
	GetActiveModel(ctx context.Context) (*domain.MLModelRun, error)
	PromoteModel(ctx context.Context, version, userID string) (*domain.MLModelRun, error)
}

type officialPredictionPublisher interface {
	TriggerPredictionsForModel(ctx context.Context, horizonDays int, modelVersion string) (*PredictionRunOutputDTO, error)
}

type PromoteModelOutputDTO struct {
	Model         *domain.MLModelRun
	PredictionRun *PredictionRunOutputDTO
}

type PromoteModelAndPublishUseCase struct {
	models      modelPromotionService
	predictions officialPredictionPublisher
}

func NewPromoteModelAndPublishUseCase(
	models modelPromotionService,
	predictions officialPredictionPublisher,
) *PromoteModelAndPublishUseCase {
	return &PromoteModelAndPublishUseCase{models: models, predictions: predictions}
}

func (uc *PromoteModelAndPublishUseCase) Execute(
	ctx context.Context,
	version,
	userID string,
) (*PromoteModelOutputDTO, error) {
	previous, err := uc.models.GetActiveModel(ctx)
	if err != nil && !errors.Is(err, domain.ErrModelNotFound) {
		return nil, err
	}

	model, err := uc.models.PromoteModel(ctx, version, userID)
	if err != nil {
		return nil, err
	}

	run, err := uc.predictions.TriggerPredictionsForModel(ctx, 7, model.Version)
	if err != nil {
		return nil, uc.restorePreviousModel(ctx, previous, model, userID, err)
	}

	return &PromoteModelOutputDTO{Model: model, PredictionRun: run}, nil
}

func (uc *PromoteModelAndPublishUseCase) restorePreviousModel(
	ctx context.Context,
	previous,
	current *domain.MLModelRun,
	userID string,
	publishErr error,
) error {
	if previous == nil || previous.Version == "" || previous.Version == current.Version {
		return publishErr
	}

	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if _, err := uc.models.PromoteModel(rollbackCtx, previous.Version, userID); err != nil {
		return errors.Join(
			publishErr,
			fmt.Errorf("falha ao restaurar o modelo %s apos erro na rodada oficial: %w", previous.Version, err),
		)
	}

	return fmt.Errorf("rodada oficial falhou; modelo %s restaurado: %w", previous.Version, publishErr)
}
