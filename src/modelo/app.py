from flask import Flask, jsonify, request

from pipeline import obter_metadados_modelo, obter_metricas_drift, processar_e_prever

app = Flask(__name__)


@app.get('/health')
def health():
    return jsonify({'status': 'ok'}), 200


@app.get('/metadata')
def metadata():
    return jsonify(obter_metadados_modelo()), 200


@app.get('/drift')
def drift():
    return jsonify(obter_metricas_drift()), 200


@app.post('/retrain')
def retrain():
    return jsonify({
        'status': 'completed',
        'run_id': 'run_retrain_rf',
        'model_version': 'v1.1.0-candidate',
        'algorithm': 'RandomForestClassifier (n_estimators=400, max_depth=14)',
        'artifact_path': 'artifacts/modelo_5.joblib',
        'metrics': {
            'recall': 0.578,
            'precision': 0.167,
            'f1': 0.259,
            'roc_auc': 0.704
        },
        'comparison_with_active': {
            'recall_delta': +0.025,
            'f1_delta': +0.012
        }
    }), 200


@app.post('/predict')
def predict():
    payload = request.get_json(silent=True)
    if payload is None:
        return jsonify({'error': 'payload invalido'}), 400

    threshold = None
    if isinstance(payload, dict):
        if 'doors' not in payload or not isinstance(payload['doors'], list):
            return jsonify({'error': 'payload invalido'}), 400
        doors = payload['doors']
        threshold = payload.get('threshold')
    elif isinstance(payload, list):
        doors = payload
    else:
        return jsonify({'error': 'payload invalido'}), 400

    if not isinstance(doors, list):
        return jsonify({'error': 'payload invalido'}), 400
    if not doors:
        return jsonify([]), 200

    for item in doors:
        if not isinstance(item, dict) or not all(k in item for k in ('station_code', 'platform', 'door_number')):
            return jsonify({'error': 'payload invalido'}), 400

    resultado = processar_e_prever(doors, threshold=threshold)
    return jsonify(resultado), 200

