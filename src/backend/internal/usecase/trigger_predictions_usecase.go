package usecase

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
	"github.com/google/uuid"
)

type TriggerPredictionsUseCase struct {
	predRepo  port.PredictionRepository
	mlClient  port.MLClient
	eventBus  port.EventBus
	uow       port.UnitOfWork
	mlopsRepo port.MLOpsRepository
}

func NewTriggerPredictionsUseCase(
	predRepo port.PredictionRepository,
	mlClient port.MLClient,
	eventBus port.EventBus,
	uow port.UnitOfWork,
) *TriggerPredictionsUseCase {
	return &TriggerPredictionsUseCase{
		predRepo: predRepo,
		mlClient: mlClient,
		eventBus: eventBus,
		uow:      uow,
	}
}

func (uc *TriggerPredictionsUseCase) SetMLOpsRepository(repo port.MLOpsRepository) {
	uc.mlopsRepo = repo
}

func countDoorsAtRisk(items []domain.PredictionItem) int {
	doorsAtRisk := 0
	for _, item := range items {
		if item.WillFail {
			doorsAtRisk++
		}
	}
	return doorsAtRisk
}

func applyPredictionThreshold(items []domain.PredictionItem, threshold *float64) {
	if threshold == nil {
		return
	}
	for i := range items {
		if items[i].Probability != nil {
			items[i].WillFail = *items[i].Probability >= *threshold
		}
	}
}

func calculatePredictionMetrics(items []domain.PredictionItem) (average, p95, maximum *float64) {
	probabilities := make([]float64, 0, len(items))
	for _, item := range items {
		if item.Probability != nil {
			probabilities = append(probabilities, *item.Probability)
		}
	}
	if len(probabilities) == 0 {
		return nil, nil, nil
	}
	sort.Float64s(probabilities)
	sum := 0.0
	for _, probability := range probabilities {
		sum += probability
	}
	averageValue := sum / float64(len(probabilities))
	p95Index := int(math.Ceil(float64(len(probabilities))*0.95)) - 1
	p95Value := probabilities[p95Index]
	maximumValue := probabilities[len(probabilities)-1]
	return &averageValue, &p95Value, &maximumValue
}

func calculateClassificationMetrics(threshold *float64) (precision, recall, f1 *float64) {
	t := 0.48
	if threshold != nil {
		t = *threshold
	}
	p, r, f := domain.GetThresholdCalibrationMetrics(t)
	return &p, &r, &f
}

func assignItemRunIDs(items []domain.PredictionItem, runID string) {
	for i := range items {
		items[i].ID = uuid.New().String()
		items[i].RunID = runID
	}
}

func createAutomaticInspection(ctx context.Context, store port.UnitOfWorkStore, p domain.PredictionItem, runID string) error {
	predID := p.ID
	ins, err := domain.NewInspection(domain.CreateInspectionParams{
		ID:           uuid.New().String(),
		PredictionID: &predID,
		StationCode:  p.StationCode,
		Platform:     p.Platform,
		DoorNumber:   p.DoorNumber,
		Notes:        "",
	})
	if err != nil {
		return err
	}
	return store.Inspections().Create(ctx, ins)
}

func (uc *TriggerPredictionsUseCase) saveRunAndInspections(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem) error {
	return uc.uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		if err := store.Predictions().SaveRun(ctx, run, items); err != nil {
			return err
		}
		for _, p := range items {
			if p.WillFail {
				if err := createAutomaticInspection(ctx, store, p, run.ID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (uc *TriggerPredictionsUseCase) publishPredictionEvents(ctx context.Context, run *domain.PredictionRun, items []domain.PredictionItem, timestamp time.Time) {
	if uc.eventBus == nil {
		return
	}

	for _, p := range items {
		if p.WillFail {
			uc.eventBus.Publish(ctx, domain.DoorRiskDetectedEvent{
				RunID:       run.ID,
				StationCode: p.StationCode,
				Platform:    p.Platform,
				DoorNumber:  p.DoorNumber,
				Timestamp:   timestamp,
			})
		}
	}

	uc.eventBus.Publish(ctx, domain.PredictionRunCompletedEvent{
		RunID:          run.ID,
		HorizonDays:    run.HorizonDays,
		TotalEvaluated: run.TotalDoorsEvaluated,
		TotalAtRisk:    run.TotalDoorsAtRisk,
		Timestamp:      timestamp,
	})
}

func toPredictionItemDTOs(items []domain.PredictionItem) []PredictionItemOutputDTO {
	itemsDTO := make([]PredictionItemOutputDTO, 0, len(items))
	for _, p := range items {
		itemsDTO = append(itemsDTO, PredictionItemOutputDTO{
			ID:          p.ID,
			RunID:       p.RunID,
			StationCode: p.StationCode,
			Platform:    p.Platform,
			DoorNumber:  p.DoorNumber,
			WillFail:    p.WillFail,
			Probability: p.Probability,
		})
	}
	return itemsDTO
}

func (uc *TriggerPredictionsUseCase) Execute(ctx context.Context, horizonDays int) (*PredictionRunOutputDTO, error) {
	return uc.execute(ctx, horizonDays, nil, false)
}

func (uc *TriggerPredictionsUseCase) ExecuteWithThreshold(ctx context.Context, horizonDays int, customThreshold *float64) (*PredictionRunOutputDTO, error) {
	return uc.execute(ctx, horizonDays, customThreshold, false)
}

func (uc *TriggerPredictionsUseCase) ExecuteExperiment(ctx context.Context, threshold float64) (*PredictionRunOutputDTO, error) {
	if threshold < 0.05 || threshold > 0.95 {
		return nil, domain.ErrInvalidConfig
	}
	return uc.execute(ctx, 7, &threshold, true)
}

func (uc *TriggerPredictionsUseCase) execute(ctx context.Context, horizonDays int, customThreshold *float64, experiment bool) (*PredictionRunOutputDTO, error) {
	horizonDays = 7
	threshold := customThreshold
	modelVersion := ""
	if !experiment && threshold == nil && uc.mlopsRepo != nil {
		if cfg, err := uc.mlopsRepo.GetConfig(ctx); err == nil && cfg != nil {
			threshold = &cfg.Threshold
		}
	}
	if uc.mlopsRepo != nil {
		if model, err := uc.mlopsRepo.GetActiveModel(ctx); err == nil && model != nil {
			modelVersion = model.Version
		}
	}

	doors, err := uc.predRepo.ListDoorsForInference(ctx)
	if err != nil {
		return nil, err
	}

	if len(doors) == 0 {
		return nil, domain.ErrNoActiveDoors
	}

	predictedItems, err := uc.mlClient.PredictBatch(ctx, doors, threshold)
	if err != nil {
		return nil, err
	}
	applyPredictionThreshold(predictedItems, threshold)
	averageProbability, p95Probability, maximumProbability := calculatePredictionMetrics(predictedItems)
	estimatedPrecision, estimatedRecall, estimatedF1 := calculateClassificationMetrics(threshold)

	now := time.Now()
	var publishedAt *time.Time
	if !experiment {
		publishedAt = &now
	}
	runID := uuid.New().String()
	run := &domain.PredictionRun{
		ID:                  runID,
		HorizonDays:         horizonDays,
		ThresholdUsed:       threshold,
		ModelVersion:        modelVersion,
		IsExperiment:        experiment,
		PublishedAt:         publishedAt,
		TotalDoorsEvaluated: len(doors),
		TotalDoorsAtRisk:    countDoorsAtRisk(predictedItems),
		AverageProbability:  averageProbability,
		P95Probability:      p95Probability,
		MaximumProbability:  maximumProbability,
		EstimatedPrecision:  estimatedPrecision,
		EstimatedRecall:     estimatedRecall,
		EstimatedF1:         estimatedF1,
		CreatedAt:           now,
	}

	assignItemRunIDs(predictedItems, runID)

	var saveErr error
	if experiment {
		saveErr = uc.uow.Do(ctx, func(store port.UnitOfWorkStore) error {
			return store.Predictions().SaveRun(ctx, run, predictedItems)
		})
	} else {
		saveErr = uc.saveRunAndInspections(ctx, run, predictedItems)
	}
	if saveErr != nil {
		return nil, saveErr
	}

	if !experiment {
		uc.publishPredictionEvents(ctx, run, predictedItems, now)
	}

	return &PredictionRunOutputDTO{
		ID:                  run.ID,
		HorizonDays:         run.HorizonDays,
		ThresholdUsed:       run.ThresholdUsed,
		ModelVersion:        run.ModelVersion,
		IsExperiment:        run.IsExperiment,
		PublishedAt:         run.PublishedAt,
		TotalDoorsEvaluated: run.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    run.TotalDoorsAtRisk,
		AverageProbability:  run.AverageProbability,
		P95Probability:      run.P95Probability,
		MaximumProbability:  run.MaximumProbability,
		EstimatedPrecision:  run.EstimatedPrecision,
		EstimatedRecall:     run.EstimatedRecall,
		EstimatedF1:         run.EstimatedF1,
		CreatedAt:           now,
		Predictions:         toPredictionItemDTOs(predictedItems),
	}, nil
}

func (uc *TriggerPredictionsUseCase) ListExperiments(ctx context.Context, limit int) ([]PredictionRunOutputDTO, error) {
	runs, err := uc.predRepo.ListExperiments(ctx, limit)
	if err != nil {
		return nil, err
	}
	result := make([]PredictionRunOutputDTO, 0, len(runs))
	for _, run := range runs {
		result = append(result, PredictionRunOutputDTO{
			ID:                  run.ID,
			HorizonDays:         run.HorizonDays,
			ThresholdUsed:       run.ThresholdUsed,
			ModelVersion:        run.ModelVersion,
			IsExperiment:        true,
			TotalDoorsEvaluated: run.TotalDoorsEvaluated,
			TotalDoorsAtRisk:    run.TotalDoorsAtRisk,
			AverageProbability:  run.AverageProbability,
			P95Probability:      run.P95Probability,
			MaximumProbability:  run.MaximumProbability,
			EstimatedPrecision:  run.EstimatedPrecision,
			EstimatedRecall:     run.EstimatedRecall,
			EstimatedF1:         run.EstimatedF1,
			CreatedAt:           run.CreatedAt,
		})
	}
	return result, nil
}

func (uc *TriggerPredictionsUseCase) PromoteExperiment(ctx context.Context, id, userID string) (*PredictionRunOutputDTO, error) {
	var promoted *domain.PredictionRun
	err := uc.uow.Do(ctx, func(store port.UnitOfWorkStore) error {
		run, err := store.Predictions().GetRunByID(ctx, id)
		if err != nil {
			return err
		}
		if !run.IsExperiment {
			return domain.ErrExperimentNotFound
		}
		run.AverageProbability, run.P95Probability, run.MaximumProbability = calculatePredictionMetrics(run.Predictions)
		run.EstimatedPrecision, run.EstimatedRecall, run.EstimatedF1 = calculateClassificationMetrics(run.ThresholdUsed)
		if err := store.Predictions().PromoteExperiment(ctx, id, userID); err != nil {
			return err
		}
		for _, item := range run.Predictions {
			if item.WillFail {
				if err := createAutomaticInspection(ctx, store, item, run.ID); err != nil {
					return err
				}
			}
		}
		now := time.Now()
		run.IsExperiment = false
		run.PublishedAt = &now
		promoted = run
		return nil
	})
	if err != nil {
		return nil, err
	}
	uc.publishPredictionEvents(ctx, promoted, promoted.Predictions, *promoted.PublishedAt)
	return &PredictionRunOutputDTO{
		ID:                  promoted.ID,
		HorizonDays:         promoted.HorizonDays,
		ThresholdUsed:       promoted.ThresholdUsed,
		ModelVersion:        promoted.ModelVersion,
		PublishedAt:         promoted.PublishedAt,
		TotalDoorsEvaluated: promoted.TotalDoorsEvaluated,
		TotalDoorsAtRisk:    promoted.TotalDoorsAtRisk,
		AverageProbability:  promoted.AverageProbability,
		P95Probability:      promoted.P95Probability,
		MaximumProbability:  promoted.MaximumProbability,
		EstimatedPrecision:  promoted.EstimatedPrecision,
		EstimatedRecall:     promoted.EstimatedRecall,
		EstimatedF1:         promoted.EstimatedF1,
		CreatedAt:           promoted.CreatedAt,
		Predictions:         toPredictionItemDTOs(promoted.Predictions),
	}, nil
}
