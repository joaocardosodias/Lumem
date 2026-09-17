package domain

import (
	"errors"
	"time"
)

var (
	ErrModelNotFound = errors.New("modelo nao encontrado")
	ErrInvalidConfig = errors.New("configuracao de mlops invalida")
)

type MLConfig struct {
	ID                 string    `json:"id"`
	Threshold          float64   `json:"threshold"`
	DefaultHorizonDays int       `json:"default_horizon_days"`
	AutoRetrainEnabled bool      `json:"auto_retrain_enabled"`
	UpdatedAt          time.Time `json:"updated_at"`
	UpdatedBy          string    `json:"updated_by"`
}

type MLModelRun struct {
	ID                   string            `json:"id"`
	Version              string            `json:"version"`
	Algorithm            string            `json:"algorithm"`
	Status               string            `json:"status"`
	TrainMetrics         map[string]any    `json:"train_metrics"`
	TestMetrics          map[string]any    `json:"test_metrics"`
	FeatureImportance    []map[string]any  `json:"feature_importance"`
	ThresholdMetrics     []ThresholdMetric `json:"threshold_metrics,omitempty"`
	ArtifactPath         string            `json:"artifact_path"`
	ArtifactURI          string            `json:"artifact_uri"`
	ArtifactSHA256       string            `json:"artifact_sha256"`
	DatasetSHA256        string            `json:"dataset_sha256"`
	TargetName           string            `json:"target"`
	TrainEnd             *time.Time        `json:"train_end,omitempty"`
	TestStart            *time.Time        `json:"test_start,omitempty"`
	TestEnd              *time.Time        `json:"test_end,omitempty"`
	TestRows             int               `json:"test_rows"`
	PositiveRows         int               `json:"positive_rows"`
	RecommendedThreshold *float64          `json:"recommended_threshold,omitempty"`
	Manifest             map[string]any    `json:"manifest"`
	StartedAt            time.Time         `json:"started_at"`
	FinishedAt           time.Time         `json:"finished_at"`
	ActivatedAt          *time.Time        `json:"activated_at,omitempty"`
	CreatedBy            string            `json:"created_by"`
}

type ThresholdMetric struct {
	Threshold      float64 `json:"threshold"`
	TruePositives  int     `json:"true_positives"`
	FalsePositives int     `json:"false_positives"`
	FalseNegatives int     `json:"false_negatives"`
	TrueNegatives  int     `json:"true_negatives"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	F1Score        float64 `json:"f1_score"`
	AlertCount     int     `json:"alert_count"`
	AlertRate      float64 `json:"alert_rate"`
}

type MLTrainingJob struct {
	ID               string             `json:"id"`
	RequestedVersion string             `json:"version"`
	Status           string             `json:"status"`
	RequestedBy      string             `json:"requested_by"`
	Parameters       map[string]any     `json:"parameters"`
	Manifest         map[string]any     `json:"manifest,omitempty"`
	Progress         MLTrainingProgress `json:"progress"`
	ErrorMessage     string             `json:"error,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	StartedAt        *time.Time         `json:"started_at,omitempty"`
	FinishedAt       *time.Time         `json:"finished_at,omitempty"`
}

type MLTrainingProgress struct {
	CurrentStage   string            `json:"current_stage"`
	CompletedSteps int               `json:"completed_steps"`
	TotalSteps     int               `json:"total_steps"`
	Percent        int               `json:"percent"`
	UpdatedAt      string            `json:"updated_at"`
	Stages         []MLTrainingStage `json:"stages"`
}

type MLTrainingStage struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	DurationMS *int64 `json:"duration_ms,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

type MLDriftReport struct {
	Status                 string           `json:"status"`
	DataDriftDetected      *bool            `json:"data_drift_detected"`
	ModelVersion           string           `json:"model_version"`
	ReferenceDatasetSHA256 string           `json:"reference_dataset_sha256"`
	PSIScore               *float64         `json:"psi_score"`
	KSStatistic            *float64         `json:"ks_statistic"`
	EvaluatedAt            string           `json:"evaluated_at"`
	FeaturesMonitored      int              `json:"features_monitored"`
	FeaturesDrift          []map[string]any `json:"features_drift"`
	Message                string           `json:"message,omitempty"`
}

type ProductionPerformance struct {
	ModelVersion      string   `json:"model_version"`
	Threshold         *float64 `json:"threshold,omitempty"`
	MatureRuns        int      `json:"mature_runs"`
	EvaluatedItems    int      `json:"evaluated_predictions"`
	TotalItems        int      `json:"total_predictions"`
	LabelCoverage     float64  `json:"label_coverage"`
	TruePositives     int      `json:"true_positives"`
	FalsePositives    int      `json:"false_positives"`
	FalseNegatives    int      `json:"false_negatives"`
	TrueNegatives     int      `json:"true_negatives"`
	Precision         *float64 `json:"precision,omitempty"`
	Recall            *float64 `json:"recall,omitempty"`
	F1Score           *float64 `json:"f1_score,omitempty"`
	LastEvaluatedAt   *string  `json:"last_evaluated_at,omitempty"`
	FailureDefinition string   `json:"failure_definition"`
}
