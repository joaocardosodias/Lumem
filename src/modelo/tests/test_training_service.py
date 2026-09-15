import json

from ml_service.application.training_service import TrainingJobStore, TrainingService


def test_reinicio_marca_job_interrompido_como_falha(tmp_path):
    store = TrainingJobStore(str(tmp_path))
    store.save({
        'id': 'interrompido',
        'version': 'v2.0.0',
        'status': 'running',
        'created_at': '2026-09-14T12:00:00+00:00',
        'started_at': '2026-09-14T12:01:00+00:00',
        'finished_at': None,
        'error': None,
    })

    service = TrainingService(store)
    recovered = service.get('interrompido')

    assert recovered['status'] == 'failed'
    assert recovered['finished_at']
    assert 'reiniciou' in recovered['error']
    assert json.loads((tmp_path / 'interrompido.json').read_text())['status'] == 'failed'
