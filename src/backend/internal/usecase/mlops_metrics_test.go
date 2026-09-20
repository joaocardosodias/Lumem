package usecase

import "testing"

func validThresholdManifest() map[string]any {
	metrics := make([]any, 0, 91)
	for step := 5; step <= 95; step++ {
		metrics = append(metrics, map[string]any{
			"threshold":       float64(step) / 100,
			"true_positives":  8.0,
			"false_positives": 2.0,
			"false_negatives": 2.0,
			"true_negatives":  88.0,
			"precision":       0.8,
			"recall":          0.8,
			"f1_score":        0.8,
			"alert_count":     10.0,
			"alert_rate":      0.1,
		})
	}
	return map[string]any{
		"threshold_metrics": metrics,
	}
}

func TestParseThresholdMetricsValidatesConfusionMatrix(t *testing.T) {
	metrics, err := parseThresholdMetrics(validThresholdManifest())
	if err != nil {
		t.Fatalf("valid curve rejected: %v", err)
	}
	if len(metrics) != 91 || metrics[0].TruePositives != 8 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func TestParseThresholdMetricsRejectsInventedF1(t *testing.T) {
	manifest := validThresholdManifest()
	manifest["threshold_metrics"].([]any)[0].(map[string]any)["f1_score"] = 0.99
	if _, err := parseThresholdMetrics(manifest); err == nil {
		t.Fatal("expected inconsistent F1 to be rejected")
	}
}

func TestParseThresholdMetricsRejectsIncompleteCurve(t *testing.T) {
	manifest := validThresholdManifest()
	manifest["threshold_metrics"] = manifest["threshold_metrics"].([]any)[:90]
	if _, err := parseThresholdMetrics(manifest); err == nil {
		t.Fatal("expected incomplete threshold curve to be rejected")
	}
}
