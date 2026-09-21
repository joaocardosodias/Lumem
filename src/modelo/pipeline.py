"""Fachada depreciada para clientes antigos; a implementação vive em ml_service/."""

from ml_service.adapters.model_registry import registry_from_environment
from ml_service.adapters.postgres_feature_repository import PostgresFeatureRepository
from ml_service.application.monitoring_use_case import MonitoringUseCase
from ml_service.application.predict_use_case import PredictUseCase

_use_case = None


def _prediction_use_case():
    global _use_case
    if _use_case is None:
        _use_case = PredictUseCase(PostgresFeatureRepository(), registry_from_environment())
    return _use_case


def processar_e_prever(portas: list[dict], threshold: float | None = None) -> list[dict]:
    result = _prediction_use_case().execute(portas)
    chosen = threshold
    if chosen is None:
        chosen = _prediction_use_case().load().bundle.get('threshold', 0.5)
    return [dict(item, will_fail=item['probability'] >= float(chosen)) for item in result['predictions']]


def obter_metadados_modelo() -> dict:
    return _prediction_use_case().metadata()


def obter_metricas_drift() -> dict:
    return MonitoringUseCase(_prediction_use_case()).execute()
