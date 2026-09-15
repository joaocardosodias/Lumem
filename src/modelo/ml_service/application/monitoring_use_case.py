from datetime import datetime, timezone

import numpy as np

from ml_service.domain.feature_builder import construir_features


def _distribution_metrics(values, reference: dict) -> tuple[float, float, float]:
    clean = np.asarray(values, dtype=float)
    missing_ratio = float(np.isnan(clean).mean()) if clean.size else 0.0
    clean = clean[np.isfinite(clean)]
    bins = np.asarray(reference.get('bins', []), dtype=float)
    expected = np.asarray(reference.get('proportions', []), dtype=float)
    if clean.size == 0 or bins.size < 2 or expected.size != bins.size - 1:
        raise ValueError('distribuição de referência incompleta')

    actual_counts, _ = np.histogram(clean, bins=bins)
    actual = actual_counts / max(actual_counts.sum(), 1)
    epsilon = 1e-6
    expected = np.clip(expected, epsilon, None)
    actual = np.clip(actual, epsilon, None)
    psi = float(np.sum((actual - expected) * np.log(actual / expected)))
    ks = float(np.max(np.abs(np.cumsum(actual) - np.cumsum(expected))))
    return psi, ks, missing_ratio


class MonitoringUseCase:
    def __init__(self, prediction_use_case):
        self.prediction_use_case = prediction_use_case

    def execute(self, version: str | None = None) -> dict:
        model = self.prediction_use_case.load()
        if version and version != model.version:
            raise LookupError('a versão solicitada não está carregada')

        reference = model.manifest.get('reference_distribution') or {}
        if not reference:
            return {
                'status': 'unavailable',
                'data_drift_detected': None,
                'model_version': model.version,
                'reference_dataset_sha256': model.manifest.get('dataset_sha256'),
                'evaluated_at': datetime.now(timezone.utc).isoformat(),
                'features_monitored': 0,
                'features_drift': [],
                'message': 'O artefato não possui distribuição de referência; execute um novo treinamento.',
            }

        doors = self.prediction_use_case.feature_repository.listar_portas_ativas()
        if not doors:
            raise ValueError('não há portas ativas para calcular drift')
        matrix, reference_date = construir_features(
            doors, model.bundle, self.prediction_use_case.feature_repository
        )

        features = []
        for feature, baseline in reference.items():
            if feature not in matrix.columns:
                continue
            psi, ks, missing_ratio = _distribution_metrics(matrix[feature], baseline)
            severity = 'critical' if psi >= 0.25 else ('warning' if psi >= 0.10 else 'stable')
            features.append({
                'feature': feature,
                'psi': psi,
                'ks_statistic': ks,
                'missing_ratio': missing_ratio,
                'observed_min': float(matrix[feature].min()),
                'observed_max': float(matrix[feature].max()),
                'records': len(matrix),
                'status': severity,
            })

        max_psi = max((item['psi'] for item in features), default=0.0)
        max_ks = max((item['ks_statistic'] for item in features), default=0.0)
        return {
            'status': 'critical' if max_psi >= 0.25 else ('warning' if max_psi >= 0.10 else 'healthy'),
            'data_drift_detected': max_psi >= 0.10,
            'model_version': model.version,
            'reference_dataset_sha256': model.manifest.get('dataset_sha256'),
            'reference_date': reference_date.isoformat(),
            'evaluated_at': datetime.now(timezone.utc).isoformat(),
            'psi_score': max_psi,
            'ks_statistic': max_ks,
            'features_monitored': len(features),
            'features_drift': sorted(features, key=lambda item: item['psi'], reverse=True),
        }
