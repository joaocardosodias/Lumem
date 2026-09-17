package usecase

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type MLOpsUseCase struct {
	repo     port.MLOpsRepository
	client   port.MLOpsClient
	outcomes port.PredictionOutcomeRepository
}

func (uc *MLOpsUseCase) SetPredictionOutcomeRepository(repository port.PredictionOutcomeRepository) {
	uc.outcomes = repository
}

func NewMLOpsUseCase(repo port.MLOpsRepository, clients ...port.MLOpsClient) *MLOpsUseCase {
	var client port.MLOpsClient
	if len(clients) > 0 {
		client = clients[0]
	}
	return &MLOpsUseCase{repo: repo, client: client}
}

func (uc *MLOpsUseCase) GetConfig(ctx context.Context) (*domain.MLConfig, error) {
	return uc.repo.GetConfig(ctx)
}

func (uc *MLOpsUseCase) UpdateConfig(ctx context.Context, threshold float64, horizon int, userID string) (*domain.MLConfig, error) {
	if threshold < 0.05 || threshold > 0.95 {
		return nil, fmt.Errorf("%w: threshold deve estar entre 0.05 e 0.95", domain.ErrInvalidConfig)
	}
	if math.Abs(threshold*100-math.Round(threshold*100)) > 1e-9 {
		return nil, fmt.Errorf("%w: threshold deve usar incrementos de 0.01", domain.ErrInvalidConfig)
	}
	if horizon != 7 {
		return nil, fmt.Errorf("%w: o modelo ativo suporta apenas o horizonte de 7 dias", domain.ErrInvalidConfig)
	}
	active, err := uc.repo.GetActiveModel(ctx)
	if err != nil {
		return nil, err
	}
	metric, err := uc.repo.GetThresholdMetric(ctx, active.Version, threshold)
	if err != nil || metric == nil || math.Abs(metric.Threshold-threshold) > 1e-6 {
		return nil, fmt.Errorf("%w: modelo ativo sem avaliacao para o threshold informado", domain.ErrInvalidConfig)
	}
	config := &domain.MLConfig{
		ID:                 "active",
		Threshold:          threshold,
		DefaultHorizonDays: horizon,
		AutoRetrainEnabled: false,
		UpdatedAt:          time.Now(),
		UpdatedBy:          userID,
	}
	if err := uc.repo.UpdateConfig(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (uc *MLOpsUseCase) GetActiveModel(ctx context.Context) (*domain.MLModelRun, error) {
	return uc.repo.GetActiveModel(ctx)
}

func (uc *MLOpsUseCase) ListModelRuns(ctx context.Context, limit int) ([]domain.MLModelRun, error) {
	return uc.repo.ListModelRuns(ctx, limit)
}

func (uc *MLOpsUseCase) GetThresholdMetrics(ctx context.Context, version string) ([]domain.ThresholdMetric, error) {
	if version == "" {
		return nil, domain.ErrModelNotFound
	}
	return uc.repo.GetThresholdMetrics(ctx, version)
}

func (uc *MLOpsUseCase) TriggerRetrain(ctx context.Context, userID string) (*domain.MLTrainingJob, error) {
	if uc.client == nil {
		return nil, domain.ErrPredictionServiceUnavailable
	}
	job, err := uc.client.StartTraining(ctx, userID)
	if err != nil {
		return nil, err
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.RequestedBy = userID
	if job.Parameters == nil {
		job.Parameters = map[string]any{}
	}
	if err := uc.repo.SaveTrainingJob(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (uc *MLOpsUseCase) ResumeTraining(ctx context.Context, id, userID string) (*domain.MLTrainingJob, error) {
	stored, err := uc.repo.GetTrainingJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if stored.Status != "failed" {
		return nil, domain.ErrInvalidConfig
	}
	failedStage := ""
	for index := range stored.Progress.Stages {
		if stored.Progress.Stages[index].Status == "failed" {
			failedStage = stored.Progress.Stages[index].Key
			break
		}
	}
	if failedStage == "" {
		return nil, domain.ErrInvalidConfig
	}

	if failedStage == "catalog_registration" {
		if err := uc.ensureCompletedTrainingRegistered(ctx, stored); err != nil {
			return nil, err
		}
		stored.Status = "completed"
		stored.ErrorMessage = ""
		now := time.Now().UTC()
		stored.FinishedAt = &now
		completeTrainingRegistration(stored)
		if err := uc.repo.SaveTrainingJob(ctx, stored); err != nil {
			return nil, err
		}
		return stored, nil
	}

	if uc.client == nil {
		return nil, domain.ErrPredictionServiceUnavailable
	}
	job, err := uc.client.ResumeTraining(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.RequestedBy = userID
	if job.Parameters == nil {
		job.Parameters = map[string]any{}
	}
	if err := uc.repo.SaveTrainingJob(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (uc *MLOpsUseCase) GetTrainingJob(ctx context.Context, id string) (*domain.MLTrainingJob, error) {
	stored, err := uc.repo.GetTrainingJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if stored.Status == "completed" {
		if err := uc.ensureCompletedTrainingRegistered(ctx, stored); err != nil {
			return nil, err
		}
		if completeTrainingRegistration(stored) {
			if err := uc.repo.SaveTrainingJob(ctx, stored); err != nil {
				return nil, err
			}
		}
		return stored, nil
	}
	if stored.Status == "failed" || uc.client == nil {
		return stored, nil
	}
	current, err := uc.client.GetTrainingJob(ctx, id)
	if err != nil {
		return nil, err
	}
	current.RequestedBy = stored.RequestedBy
	current.Parameters = stored.Parameters
	if current.CreatedAt.IsZero() {
		current.CreatedAt = stored.CreatedAt
	}
	if current.Status == "completed" {
		if err := uc.ensureCompletedTrainingRegistered(ctx, current); err != nil {
			current.Status = "failed"
			current.ErrorMessage = "falha ao registrar modelo candidato: " + err.Error()
			failTrainingRegistration(current, current.ErrorMessage)
		} else {
			completeTrainingRegistration(current)
		}
	}
	if err := uc.repo.SaveTrainingJob(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

func completeTrainingRegistration(job *domain.MLTrainingJob) bool {
	return updateTrainingRegistrationStage(job, "completed", "Modelo candidato registrado no catálogo.")
}

func failTrainingRegistration(job *domain.MLTrainingJob, detail string) {
	updateTrainingRegistrationStage(job, "failed", detail)
}

func updateTrainingRegistrationStage(job *domain.MLTrainingJob, status, detail string) bool {
	if len(job.Progress.Stages) == 0 {
		return false
	}
	now := time.Now().UTC()
	changed := false
	for index := range job.Progress.Stages {
		stage := &job.Progress.Stages[index]
		if stage.Key != "catalog_registration" {
			continue
		}
		if stage.Status == status && stage.Detail == detail {
			return false
		}
		if stage.StartedAt == "" {
			stage.StartedAt = now.Format(time.RFC3339Nano)
		}
		stage.Status = status
		stage.Detail = detail
		stage.FinishedAt = now.Format(time.RFC3339Nano)
		if startedAt, err := time.Parse(time.RFC3339Nano, stage.StartedAt); err == nil {
			duration := now.Sub(startedAt).Milliseconds()
			stage.DurationMS = &duration
		}
		changed = true
		break
	}
	if !changed {
		return false
	}
	completed := 0
	for index := range job.Progress.Stages {
		if job.Progress.Stages[index].Status == "completed" || job.Progress.Stages[index].Status == "skipped" {
			completed++
		}
	}
	job.Progress.CurrentStage = "catalog_registration"
	job.Progress.CompletedSteps = completed
	job.Progress.TotalSteps = len(job.Progress.Stages)
	if job.Progress.TotalSteps > 0 {
		job.Progress.Percent = int(math.Round(float64(completed) * 100 / float64(job.Progress.TotalSteps)))
	}
	job.Progress.UpdatedAt = now.Format(time.RFC3339Nano)
	return true
}

func (uc *MLOpsUseCase) ListTrainingJobs(ctx context.Context, limit int) ([]domain.MLTrainingJob, error) {
	return uc.repo.ListTrainingJobs(ctx, limit)
}

func (uc *MLOpsUseCase) ensureCompletedTrainingRegistered(ctx context.Context, job *domain.MLTrainingJob) error {
	version := stringValue(job.Manifest, "model_version")
	if version != "" {
		model, err := uc.repo.GetModelByVersion(ctx, version)
		if err == nil && model.ArtifactSHA256 == stringValue(job.Manifest, "model_sha256") && len(model.ThresholdMetrics) > 0 {
			return nil
		}
		if err != nil && !errors.Is(err, domain.ErrModelNotFound) {
			return err
		}
	}
	return uc.registerCompletedTraining(ctx, job)
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func integer(value any) (int, bool) {
	numberValue, ok := number(value)
	if !ok || numberValue < 0 || numberValue != math.Trunc(numberValue) {
		return 0, false
	}
	return int(numberValue), true
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func mapValue(values map[string]any, key string) map[string]any {
	value, _ := values[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func parseDate(value string) *time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &parsed
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func parseThresholdMetrics(manifest map[string]any) ([]domain.ThresholdMetric, error) {
	raw, ok := manifest["threshold_metrics"].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("%w: manifest sem curva de thresholds", domain.ErrInvalidConfig)
	}
	metrics := make([]domain.ThresholdMetric, 0, len(raw))
	seen := make(map[float64]struct{}, len(raw))
	for _, item := range raw {
		values, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: ponto da curva invalido", domain.ErrInvalidConfig)
		}
		threshold, thresholdOK := number(values["threshold"])
		precision, precisionOK := number(values["precision"])
		recall, recallOK := number(values["recall"])
		f1, f1OK := number(values["f1_score"])
		alertRate, alertRateOK := number(values["alert_rate"])
		tp, tpOK := integer(values["true_positives"])
		fp, fpOK := integer(values["false_positives"])
		fn, fnOK := integer(values["false_negatives"])
		tn, tnOK := integer(values["true_negatives"])
		alerts, alertsOK := integer(values["alert_count"])
		if !thresholdOK || !precisionOK || !recallOK || !f1OK || !alertRateOK ||
			!tpOK || !fpOK || !fnOK || !tnOK || !alertsOK || threshold < 0 || threshold > 1 ||
			precision < 0 || precision > 1 || recall < 0 || recall > 1 || f1 < 0 || f1 > 1 ||
			alertRate < 0 || alertRate > 1 {
			return nil, fmt.Errorf("%w: valores invalidos na curva de thresholds", domain.ErrInvalidConfig)
		}
		if _, duplicated := seen[threshold]; duplicated {
			return nil, fmt.Errorf("%w: threshold duplicado na curva", domain.ErrInvalidConfig)
		}
		seen[threshold] = struct{}{}
		expectedPrecision := 0.0
		if tp+fp > 0 {
			expectedPrecision = float64(tp) / float64(tp+fp)
		}
		expectedRecall := 0.0
		if tp+fn > 0 {
			expectedRecall = float64(tp) / float64(tp+fn)
		}
		expectedF1 := 0.0
		if expectedPrecision+expectedRecall > 0 {
			expectedF1 = 2 * expectedPrecision * expectedRecall / (expectedPrecision + expectedRecall)
		}
		total := tp + fp + fn + tn
		expectedAlertRate := 0.0
		if total > 0 {
			expectedAlertRate = float64(alerts) / float64(total)
		}
		if total == 0 || math.Abs(precision-expectedPrecision) > 1e-6 || math.Abs(recall-expectedRecall) > 1e-6 ||
			math.Abs(f1-expectedF1) > 1e-6 || alerts != tp+fp || math.Abs(alertRate-expectedAlertRate) > 1e-6 {
			return nil, fmt.Errorf("%w: metricas divergentes da matriz de confusao", domain.ErrInvalidConfig)
		}
		metrics = append(metrics, domain.ThresholdMetric{
			Threshold: threshold, TruePositives: tp, FalsePositives: fp,
			FalseNegatives: fn, TrueNegatives: tn, Precision: precision,
			Recall: recall, F1Score: f1, AlertCount: alerts, AlertRate: alertRate,
		})
	}
	if len(metrics) != 91 {
		return nil, fmt.Errorf("%w: curva deve conter 91 thresholds", domain.ErrInvalidConfig)
	}
	for step := 5; step <= 95; step++ {
		expected := float64(step) / 100
		found := false
		for _, metric := range metrics {
			if math.Abs(metric.Threshold-expected) <= 1e-6 {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: curva sem o threshold %.2f", domain.ErrInvalidConfig, expected)
		}
	}
	return metrics, nil
}

func (uc *MLOpsUseCase) registerCompletedTraining(ctx context.Context, job *domain.MLTrainingJob) error {
	manifest := job.Manifest
	version := stringValue(manifest, "model_version")
	artifactURI := stringValue(manifest, "artifact_uri")
	artifactSHA := stringValue(manifest, "model_sha256")
	datasetSHA := stringValue(manifest, "dataset_sha256")
	algorithm := stringValue(manifest, "algorithm")
	target := stringValue(manifest, "target")
	trainEnd := parseDate(stringValue(manifest, "train_end"))
	testStart := parseDate(stringValue(manifest, "test_start"))
	testEnd := parseDate(stringValue(manifest, "test_end"))
	features, featuresOK := manifest["features"].([]any)
	if version == "" || artifactURI == "" || !validSHA256(artifactSHA) || !validSHA256(datasetSHA) ||
		algorithm == "" || target == "" || trainEnd == nil || testStart == nil || testEnd == nil ||
		!featuresOK || len(features) == 0 {
		return fmt.Errorf("%w: manifest sem versao, URI ou hashes validos", domain.ErrInvalidConfig)
	}
	featureNames := make(map[string]struct{}, len(features))
	for _, rawFeature := range features {
		feature, ok := rawFeature.(string)
		if !ok || feature == "" {
			return fmt.Errorf("%w: lista de features invalida", domain.ErrInvalidConfig)
		}
		if _, duplicated := featureNames[feature]; duplicated {
			return fmt.Errorf("%w: feature duplicada no manifest", domain.ErrInvalidConfig)
		}
		featureNames[feature] = struct{}{}
	}
	if !trainEnd.Before(*testStart) || testStart.After(*testEnd) {
		return fmt.Errorf("%w: periodos de treino e teste invalidos", domain.ErrInvalidConfig)
	}
	metrics, err := parseThresholdMetrics(manifest)
	if err != nil {
		return err
	}
	startedAt := job.CreatedAt
	if job.StartedAt != nil {
		startedAt = *job.StartedAt
	}
	finishedAt := time.Now()
	if job.FinishedAt != nil {
		finishedAt = *job.FinishedAt
	}
	recommended, recommendedOK := number(manifest["recommended_threshold"])
	testRows, testRowsOK := integer(manifest["test_rows"])
	positiveRows, positiveRowsOK := integer(manifest["positive_rows"])
	if !recommendedOK || recommended < 0.05 || recommended > 0.95 || !testRowsOK || testRows == 0 ||
		!positiveRowsOK || positiveRows > testRows {
		return fmt.Errorf("%w: procedencia da avaliacao incompleta", domain.ErrInvalidConfig)
	}
	recommendedFound := false
	for _, metric := range metrics {
		if math.Abs(metric.Threshold-recommended) <= 1e-6 {
			recommendedFound = true
			break
		}
	}
	if !recommendedFound {
		return fmt.Errorf("%w: threshold recomendado ausente da curva", domain.ErrInvalidConfig)
	}
	for _, metric := range metrics {
		if metric.TruePositives+metric.FalsePositives+metric.FalseNegatives+metric.TrueNegatives != testRows ||
			metric.TruePositives+metric.FalseNegatives != positiveRows {
			return fmt.Errorf("%w: curva pertence a outro conjunto de teste", domain.ErrInvalidConfig)
		}
	}
	featureImportance := make([]map[string]any, 0)
	if raw, ok := manifest["feature_importance"].([]any); ok {
		for _, item := range raw {
			if value, ok := item.(map[string]any); ok {
				featureImportance = append(featureImportance, value)
			}
		}
	}
	run := &domain.MLModelRun{
		ID:                   "run_" + job.ID,
		Version:              version,
		Algorithm:            algorithm,
		Status:               "candidate",
		TrainMetrics:         mapValue(manifest, "train_metrics"),
		TestMetrics:          mapValue(manifest, "test_metrics"),
		FeatureImportance:    featureImportance,
		ArtifactURI:          artifactURI,
		ArtifactSHA256:       artifactSHA,
		DatasetSHA256:        datasetSHA,
		TargetName:           target,
		TrainEnd:             trainEnd,
		TestStart:            testStart,
		TestEnd:              testEnd,
		TestRows:             testRows,
		PositiveRows:         positiveRows,
		RecommendedThreshold: &recommended,
		Manifest:             manifest,
		StartedAt:            startedAt,
		FinishedAt:           finishedAt,
		CreatedBy:            job.RequestedBy,
	}
	if err := uc.repo.SaveModelRun(ctx, run); err != nil {
		return err
	}
	stored, err := uc.repo.GetModelByVersion(ctx, run.Version)
	if err != nil {
		return err
	}
	return uc.repo.SaveThresholdMetrics(ctx, stored.ID, metrics)
}

func (uc *MLOpsUseCase) SyncActiveModel(ctx context.Context) error {
	if uc.client == nil {
		return domain.ErrPredictionServiceUnavailable
	}
	manifest, err := uc.client.GetActiveModelMetadata(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	job := &domain.MLTrainingJob{
		ID:         "sync-" + stringValue(manifest, "model_version"),
		Status:     "completed",
		Manifest:   manifest,
		CreatedAt:  now,
		StartedAt:  &now,
		FinishedAt: &now,
	}
	if err := uc.registerCompletedTraining(ctx, job); err != nil {
		return err
	}
	return uc.repo.PromoteModel(ctx, stringValue(manifest, "model_version"), "")
}

func (uc *MLOpsUseCase) GetDrift(ctx context.Context) (*domain.MLDriftReport, error) {
	if uc.client == nil {
		return nil, domain.ErrPredictionServiceUnavailable
	}
	model, err := uc.repo.GetActiveModel(ctx)
	if err != nil {
		return nil, err
	}
	report, err := uc.client.GetDrift(ctx, model.Version)
	if err != nil {
		return nil, err
	}
	if report.ModelVersion != model.Version {
		return nil, fmt.Errorf("%w: relatorio pertence a outra versao do modelo", domain.ErrInvalidConfig)
	}
	if err := uc.repo.SaveDriftReport(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

func (uc *MLOpsUseCase) PromoteModel(ctx context.Context, version, userID string) (*domain.MLModelRun, error) {
	if uc.client == nil {
		return nil, domain.ErrPredictionServiceUnavailable
	}
	target, err := uc.repo.GetModelByVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	if !validSHA256(target.ArtifactSHA256) || len(target.ThresholdMetrics) == 0 {
		return nil, fmt.Errorf("%w: modelo candidato sem artefato ou curva validos", domain.ErrInvalidConfig)
	}
	previous, err := uc.repo.GetActiveModel(ctx)
	if err != nil && err != domain.ErrModelNotFound {
		return nil, err
	}
	if err := uc.client.ActivateModel(ctx, version); err != nil {
		return nil, err
	}
	if err := uc.repo.PromoteModel(ctx, version, userID); err != nil {
		if previous != nil && previous.Version != "" {
			_ = uc.client.ActivateModel(context.WithoutCancel(ctx), previous.Version)
		}
		return nil, err
	}
	return uc.repo.GetActiveModel(ctx)
}

func (uc *MLOpsUseCase) GetProductionPerformance(ctx context.Context) (*domain.ProductionPerformance, error) {
	if uc.outcomes == nil {
		return nil, domain.ErrNoPredictions
	}
	return uc.outcomes.GetProductionPerformance(ctx)
}
