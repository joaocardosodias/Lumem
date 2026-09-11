import sys
from datetime import date, timedelta
from pathlib import Path
from unittest.mock import patch

import pandas as pd

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))


def alarmes_falsos(hoje):
    linhas = []
    for dias_atras in [1, 2, 2, 5, 10, 10, 10, 15]:
        linhas.append({
            'station_code': 'BUT', 'platform': 'BUT1', 'door_number': 8,
            'alarm_type': 'CA', 'duration_seconds': 12.0,
            'event_timestamp': str(hoje - timedelta(days=dias_atras)),
        })
    return pd.DataFrame(linhas)


def manutencoes_falsas():
    return pd.DataFrame([
        {'order_type': 'ZM01', 'station_code': 'BUT',
         'installation_location': 'SP04-6-PSD-BUT-P1S1-PP08PDM',
         'start_timestamp': '2026-06-01', 'end_timestamp': '2026-06-01'},
    ])


def test_predict_formato_e_execucao_real():
    with patch('pipeline.buscar_telemetria_recente') as mock_alarmes, \
         patch('pipeline.buscar_manutencoes') as mock_manut:
        hoje = date.today()
        mock_alarmes.return_value = alarmes_falsos(hoje)
        mock_manut.return_value = manutencoes_falsas()

        from app import app
        client = app.test_client()

        resp = client.post('/predict', json=[
            {'station_code': 'BUT', 'platform': 'BUT1', 'door_number': 8},
            {'station_code': 'BUT', 'platform': 'BUT1', 'door_number': 99},
        ])

        assert resp.status_code == 200
        corpo = resp.get_json()
        assert len(corpo) == 2
        for item in corpo:
            assert isinstance(item['will_fail'], bool)
            assert 0.0 <= item['probability'] <= 1.0

        com_historico = next(i for i in corpo if i['door_number'] == 8)
        sem_historico = next(i for i in corpo if i['door_number'] == 99)
        assert com_historico['will_fail'] is True
        assert sem_historico['will_fail'] is False
        assert com_historico['probability'] > sem_historico['probability']


def test_predict_lista_vazia():
    from app import app
    client = app.test_client()
    resp = client.post('/predict', json=[])
    assert resp.status_code == 200
    assert resp.get_json() == []


def test_predict_payload_invalido():
    from app import app
    client = app.test_client()
    resp = client.post('/predict', json={'nao': 'e uma lista'})
    assert resp.status_code == 400


def test_health():
    from app import app
    client = app.test_client()
    resp = client.get('/health')
    assert resp.get_json() == {'status': 'ok'}
