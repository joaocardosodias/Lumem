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
	ID                string           `json:"id"`
	Version           string           `json:"version"`
	Algorithm         string           `json:"algorithm"`
	Status            string           `json:"status"`
	TrainMetrics      map[string]any   `json:"train_metrics"`
	TestMetrics       map[string]any   `json:"test_metrics"`
	FeatureImportance []map[string]any `json:"feature_importance"`
	ArtifactPath      string           `json:"artifact_path"`
	StartedAt         time.Time        `json:"started_at"`
	FinishedAt        time.Time        `json:"finished_at"`
	CreatedBy         string           `json:"created_by"`
}

type MLDriftReport struct {
	Status            string           `json:"status"`
	DataDriftDetected bool             `json:"data_drift_detected"`
	PSIScore          float64          `json:"psi_score"`
	KSStatistic       float64          `json:"ks_statistic"`
	EvaluatedAt       string           `json:"evaluated_at"`
	FeaturesMonitored int              `json:"features_monitored"`
	FeaturesDrift     []map[string]any `json:"features_drift"`
	SensorHealth      map[string]any   `json:"sensor_health"`
}
