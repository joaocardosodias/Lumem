import sys
from datetime import date, timedelta
from pathlib import Path

import pandas as pd
import pytest

MODEL_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(MODEL_ROOT))

from app import create_app
from ml_service.adapters.model_registry import LocalModelRegistry
from ml_service.application.predict_use_case import PredictUseCase


class FakeFeatureRepository:
    def obter_data_referencia(self, estacoes):
        return date.today()

    def buscar_telemetria_recente(self, estacoes, hoje, janela_dias=60):
        return pd.DataFrame([
            {
                'station_code': 'BUT',
                'platform': 'BUT1',
                'door_number': 8,
                'alarm_type': 'CA',
                'duration_seconds': 12.0,
                'event_timestamp': hoje - timedelta(days=1),
            }
        ])

    def buscar_manutencoes(self, estacoes):
        return pd.DataFrame([
            {
                'order_type': 'ZM01',
                'station_code': 'BUT',
                'installation_location': 'SP04-6-PSD-BUT-P1S1-PP08PDM',
                'start_timestamp': '2026-06-01',
                'end_timestamp': '2026-06-01',
            }
        ])

    def listar_portas_ativas(self):
        return [{'station_code': 'BUT', 'platform': 'BUT1', 'door_number': 8}]


class FakeTrainingService:
    def start(self, requested_by=''):
        return {'id': 'job-1', 'version': 'v2', 'status': 'queued', 'requested_by': requested_by}

    def get(self, job_id):
        if job_id != 'job-1':
            raise LookupError('não encontrado')
        return {'id': job_id, 'version': 'v2', 'status': 'running'}

    def resume(self, job_id, requested_by=''):
        if job_id != 'job-1':
            raise LookupError('não encontrado')
        return {
            'id': 'job-2',
            'version': 'v2',
            'status': 'queued',
            'requested_by': requested_by,
            'parameters': {'resumed_from_job_id': job_id},
        }


artifact = MODEL_ROOT / 'artifacts' / 'modelo_5.joblib'
app = create_app(
    FakeFeatureRepository(),
    LocalModelRegistry(str(artifact), str(MODEL_ROOT / 'artifacts' / 'manifest.json')),
    FakeTrainingService(),
)
client = app.test_client()


def test_v1_retorna_apenas_probabilidade_e_versao():
    response = client.post('/v1/predictions', json={'doors': [
        {'station_code': 'BUT', 'platform': 'BUT1', 'door_number': 8},
        {'station_code': 'BUT', 'platform': 'BUT1', 'door_number': 99},
    ]})

    assert response.status_code == 200
    body = response.get_json()
    assert body['model_version']
    assert len(body['predictions']) == 2
    assert all('will_fail' not in item for item in body['predictions'])
    assert all(0 <= item['probability'] <= 1 for item in body['predictions'])


def test_v1_rejeita_threshold_e_campos_extras():
    response = client.post('/v1/predictions', json={'doors': [], 'threshold': 0.5})
    assert response.status_code == 400
    assert response.get_json()['status'] == 400


def test_v1_rejeita_numero_de_porta_invalido():
    response = client.post('/v1/predictions', json={'doors': [
        {'station_code': 'BUT', 'platform': 'BUT1', 'door_number': 0},
    ]})
    assert response.status_code == 400


def test_lista_vazia_preserva_contrato_versionado():
    response = client.post('/v1/predictions', json={'doors': []})
    assert response.status_code == 200
    assert response.get_json()['predictions'] == []


def test_healthchecks_distinguem_processo_e_modelo():
    assert client.get('/health/live').get_json() == {'status': 'ok'}
    ready = client.get('/health/ready')
    assert ready.status_code == 200
    assert ready.get_json()['model_version']


def test_metadata_e_metricas_possuem_procedencia():
    metadata = client.get('/v1/models/active').get_json()
    assert metadata['model_version']
    assert len(metadata['model_sha256']) == 64

    metrics = client.get(f"/v1/models/{metadata['model_version']}/metrics")
    assert metrics.status_code == 200
    assert metrics.get_json()['model_sha256'] == metadata['model_sha256']


def test_drift_sem_baseline_e_explicitamente_indisponivel():
    metadata = client.get('/v1/models/active').get_json()
    response = client.get(f"/v1/models/{metadata['model_version']}/drift")
    assert response.status_code == 200
    assert response.get_json()['status'] in {'unavailable', 'healthy', 'warning', 'critical'}


def test_training_job_retorna_estado_real_do_executor():
    created = client.post('/v1/training-jobs', json={'requested_by': 'engineer'})
    assert created.status_code == 202
    assert created.get_json()['status'] == 'queued'
    assert client.get('/v1/training-jobs/job-1').get_json()['status'] == 'running'


def test_training_job_pode_ser_retomado_do_checkpoint():
    response = client.post('/v1/training-jobs/job-1/resume', json={'requested_by': 'engineer'})
    assert response.status_code == 202
    assert response.get_json()['parameters']['resumed_from_job_id'] == 'job-1'


def test_endpoint_legado_marca_depreciacao():
    response = client.post('/predict', json=[])
    assert response.status_code == 200
    assert response.headers['Deprecation'] == 'true'


def test_promocao_nao_altera_ponteiro_antes_de_validar_candidata():
    class InvalidCandidateRegistry:
        def __init__(self):
            self.activated = False

        def materialize_active(self):
            raise AssertionError('não deve carregar o ativo neste teste')

        def materialize_version(self, version):
            return artifact, {'model_version': version, 'features': ['feature_inexistente']}

        def activate(self, version):
            self.activated = True
            return {'model_version': version}

    registry = InvalidCandidateRegistry()
    prediction = PredictUseCase(FakeFeatureRepository(), registry)

    with pytest.raises(ValueError, match='ordens de features'):
        prediction.activate('v-invalida')

    assert registry.activated is False
