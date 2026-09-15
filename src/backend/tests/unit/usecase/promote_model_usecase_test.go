package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
)

type modelPromotionServiceStub struct {
	active       *domain.MLModelRun
	models       map[string]*domain.MLModelRun
	promoteCalls []string
	promoteError map[string]error
}

func (s *modelPromotionServiceStub) GetActiveModel(context.Context) (*domain.MLModelRun, error) {
	if s.active == nil {
		return nil, domain.ErrModelNotFound
	}
	model := *s.active
	return &model, nil
}

func (s *modelPromotionServiceStub) PromoteModel(_ context.Context, version, _ string) (*domain.MLModelRun, error) {
	s.promoteCalls = append(s.promoteCalls, version)
	if err := s.promoteError[version]; err != nil {
		return nil, err
	}
	model, ok := s.models[version]
	if !ok {
		return nil, domain.ErrModelNotFound
	}
	active := *model
	active.Status = "production"
	s.active = &active
	return &active, nil
}

type officialPredictionPublisherStub struct {
	run                  *usecase.PredictionRunOutputDTO
	err                  error
	receivedHorizonDays  int
	receivedModelVersion string
}

func (s *officialPredictionPublisherStub) TriggerPredictionsForModel(
	_ context.Context,
	horizonDays int,
	modelVersion string,
) (*usecase.PredictionRunOutputDTO, error) {
	s.receivedHorizonDays = horizonDays
	s.receivedModelVersion = modelVersion
	return s.run, s.err
}

func TestPromoteModelAndPublishUseCasePublishesOfficialRun(t *testing.T) {
	models := &modelPromotionServiceStub{
		active: &domain.MLModelRun{Version: "v1", Status: "production"},
		models: map[string]*domain.MLModelRun{
			"v1": {Version: "v1", Status: "production"},
			"v2": {Version: "v2", Status: "candidate"},
		},
		promoteError: map[string]error{},
	}
	run := &usecase.PredictionRunOutputDTO{
		ID:           "official-run-v2",
		ModelVersion: "v2",
		PublishedAt:  timePointer(time.Now()),
	}
	predictions := &officialPredictionPublisherStub{run: run}
	uc := usecase.NewPromoteModelAndPublishUseCase(models, predictions)

	result, err := uc.Execute(context.Background(), "v2", "engineer-1")
	if err != nil {
		t.Fatalf("esperava promocao e publicacao com sucesso: %v", err)
	}
	if result.Model.Version != "v2" || result.PredictionRun.ID != run.ID {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if predictions.receivedHorizonDays != 7 || predictions.receivedModelVersion != "v2" {
		t.Fatalf("rodada recebeu horizonte/modelo incorretos: %d, %s", predictions.receivedHorizonDays, predictions.receivedModelVersion)
	}
	if len(models.promoteCalls) != 1 || models.promoteCalls[0] != "v2" {
		t.Fatalf("promocoes inesperadas: %v", models.promoteCalls)
	}
}

func TestPromoteModelAndPublishUseCaseRestoresPreviousModelWhenRunFails(t *testing.T) {
	publishErr := domain.ErrPredictionServiceUnavailable
	models := &modelPromotionServiceStub{
		active: &domain.MLModelRun{Version: "v1", Status: "production"},
		models: map[string]*domain.MLModelRun{
			"v1": {Version: "v1", Status: "retired"},
			"v2": {Version: "v2", Status: "candidate"},
		},
		promoteError: map[string]error{},
	}
	predictions := &officialPredictionPublisherStub{err: publishErr}
	uc := usecase.NewPromoteModelAndPublishUseCase(models, predictions)

	result, err := uc.Execute(context.Background(), "v2", "engineer-1")
	if result != nil || !errors.Is(err, publishErr) {
		t.Fatalf("esperava falha de publicacao preservada, resultado=%+v erro=%v", result, err)
	}
	if models.active.Version != "v1" {
		t.Fatalf("modelo anterior nao foi restaurado: %+v", models.active)
	}
	if len(models.promoteCalls) != 2 || models.promoteCalls[0] != "v2" || models.promoteCalls[1] != "v1" {
		t.Fatalf("sequencia de promocao e restauracao inesperada: %v", models.promoteCalls)
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
