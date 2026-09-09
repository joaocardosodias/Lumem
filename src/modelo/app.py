from flask import Flask, jsonify, request

from pipeline import processar_e_prever

app = Flask(__name__)


@app.get('/health')
def health():
    return jsonify({'status': 'ok'}), 200


@app.post('/predict')
def predict():
    payload = request.get_json(silent=True)
    if not isinstance(payload, list):
        return jsonify({'error': 'payload invalido'}), 400
    if not payload:
        return jsonify([]), 200

    for item in payload:
        if not isinstance(item, dict) or not all(k in item for k in ('station_code', 'platform', 'door_number')):
            return jsonify({'error': 'payload invalido'}), 400

    resultado = processar_e_prever(payload)
    return jsonify(resultado), 200
