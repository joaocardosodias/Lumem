from flask import Flask, jsonify, request

from ml_service.adapters.model_registry import registry_from_environment
from ml_service.adapters.postgres_feature_repository import PostgresFeatureRepository
from ml_service.application.monitoring_use_case import MonitoringUseCase
from ml_service.application.predict_use_case import PredictUseCase
from ml_service.application.training_service import TrainingService


def _problem(status: int, title: str, detail: str):
    return jsonify({
        'type': f'https://g04.local/problems/ml-{status}',
        'title': title,
        'status': status,
        'detail': detail,
        'instance': request.path,
    }), status


def _validate_doors(payload) -> tuple[list[dict] | None, str | None]:
    if not isinstance(payload, dict) or set(payload) != {'doors'}:
        return None, 'o corpo deve conter somente o campo doors'
    doors = payload.get('doors')
    if not isinstance(doors, list):
        return None, 'doors deve ser uma lista'
    if len(doors) > 2000:
        return None, 'doors deve conter no máximo 2000 itens'
    allowed = {'station_code', 'platform', 'door_number'}
    for index, item in enumerate(doors):
        if not isinstance(item, dict) or set(item) != allowed:
            return None, f'doors[{index}] deve conter station_code, platform e door_number'
        if not isinstance(item['station_code'], str) or not item['station_code'].strip():
            return None, f'doors[{index}].station_code é inválido'
        if not isinstance(item['platform'], str) or not item['platform'].strip():
            return None, f'doors[{index}].platform é inválido'
        if not isinstance(item['door_number'], int) or isinstance(item['door_number'], bool) or item['door_number'] <= 0:
            return None, f'doors[{index}].door_number deve ser um inteiro positivo'
    return doors, None


def create_app(feature_repository=None, model_registry=None, training_service=None):
    flask_app = Flask(__name__)
    repository = feature_repository or PostgresFeatureRepository()
    registry = model_registry or registry_from_environment()
    prediction = PredictUseCase(repository, registry)
    monitoring = MonitoringUseCase(prediction)
    training = training_service or TrainingService()

    @flask_app.get('/health/live')
    def health_live():
        return jsonify({'status': 'ok'}), 200

    @flask_app.get('/health/ready')
    def health_ready():
        if prediction.ready():
            return jsonify({'status': 'ready', 'model_version': prediction.load().version}), 200
        return _problem(503, 'Serviço não está pronto', 'Nenhum artefato válido pôde ser carregado.')

    @flask_app.post('/v1/predictions')
    def predict_v1():
        doors, validation_error = _validate_doors(request.get_json(silent=True))
        if validation_error:
            return _problem(400, 'Payload inválido', validation_error)
        if not doors:
            metadata = prediction.metadata()
            return jsonify({'model_version': metadata['model_version'], 'predictions': []}), 200
        try:
            return jsonify(prediction.execute(doors)), 200
        except Exception as error:
            return _problem(503, 'Inferência indisponível', str(error))

    @flask_app.get('/v1/models/active')
    def active_model():
        try:
            return jsonify(prediction.metadata()), 200
        except Exception as error:
            return _problem(503, 'Modelo indisponível', str(error))

    @flask_app.get('/v1/models/<version>/metrics')
    def model_metrics(version):
        try:
            metadata = prediction.metadata()
            if metadata.get('model_version') != version:
                return _problem(404, 'Modelo não encontrado', 'A versão solicitada não está carregada neste serviço.')
            return jsonify({
                'model_version': version,
                'model_sha256': metadata.get('model_sha256'),
                'dataset_sha256': metadata.get('dataset_sha256'),
                'target': metadata.get('target'),
                'train_end': metadata.get('train_end'),
                'test_start': metadata.get('test_start'),
                'test_end': metadata.get('test_end'),
                'test_rows': metadata.get('test_rows'),
                'positive_rows': metadata.get('positive_rows'),
                'threshold_metrics': metadata.get('threshold_metrics', []),
            }), 200
        except Exception as error:
            return _problem(503, 'Métricas indisponíveis', str(error))

    @flask_app.post('/v1/models/<version>/activate')
    def activate_model(version):
        try:
            return jsonify(prediction.activate(version)), 200
        except FileNotFoundError as error:
            return _problem(404, 'Modelo não encontrado', str(error))
        except Exception as error:
            return _problem(409, 'Modelo não ativado', str(error))

    @flask_app.post('/v1/training-jobs')
    def start_training():
        payload = request.get_json(silent=True) or {}
        if not isinstance(payload, dict) or set(payload).difference({'requested_by'}):
            return _problem(400, 'Payload inválido', 'Somente requested_by pode ser informado.')
        try:
            return jsonify(training.start(str(payload.get('requested_by', '')))), 202
        except RuntimeError as error:
            return _problem(409, 'Treinamento não iniciado', str(error))

    @flask_app.get('/v1/training-jobs/<job_id>')
    def training_status(job_id):
        try:
            return jsonify(training.get(job_id)), 200
        except LookupError as error:
            return _problem(404, 'Treinamento não encontrado', str(error))

    @flask_app.get('/v1/models/<version>/drift')
    def model_drift(version):
        try:
            return jsonify(monitoring.execute(version)), 200
        except LookupError as error:
            return _problem(404, 'Modelo não encontrado', str(error))
        except Exception as error:
            return _problem(503, 'Drift indisponível', str(error))

    # Compatibilidade temporária. O backend Go utiliza exclusivamente /v1.
    @flask_app.get('/health')
    def legacy_health():
        response, status = health_live()
        response.headers['Deprecation'] = 'true'
        return response, status

    @flask_app.post('/predict')
    def legacy_predict():
        payload = request.get_json(silent=True)
        threshold = None
        doors = payload
        if isinstance(payload, dict):
            doors = payload.get('doors')
            threshold = payload.get('threshold')
        normalized, validation_error = _validate_doors({'doors': doors})
        if validation_error:
            return _problem(400, 'Payload inválido', validation_error)
        if not normalized:
            response = jsonify([])
            response.headers['Deprecation'] = 'true'
            return response, 200
        try:
            result = prediction.execute(normalized)
            fallback = float(prediction.load().bundle.get('threshold', 0.5))
            chosen = fallback if threshold is None else float(threshold)
            legacy = [dict(item, will_fail=item['probability'] >= chosen) for item in result['predictions']]
            response = jsonify(legacy)
            response.headers['Deprecation'] = 'true'
            response.headers['Sunset'] = 'Wed, 31 Dec 2026 23:59:59 GMT'
            return response, 200
        except Exception as error:
            return _problem(503, 'Inferência indisponível', str(error))

    @flask_app.get('/metadata')
    def legacy_metadata():
        try:
            response = jsonify(prediction.metadata())
            response.headers['Deprecation'] = 'true'
            return response, 200
        except Exception as error:
            return _problem(503, 'Modelo indisponível', str(error))

    @flask_app.get('/drift')
    def legacy_drift():
        try:
            response = jsonify(monitoring.execute())
            response.headers['Deprecation'] = 'true'
            return response, 200
        except Exception as error:
            return _problem(503, 'Drift indisponível', str(error))

    @flask_app.post('/retrain')
    def legacy_retrain():
        response = jsonify(training.start('legacy-client'))
        response.headers['Deprecation'] = 'true'
        return response, 202

    return flask_app


app = create_app()
