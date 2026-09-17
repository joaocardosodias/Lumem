import json

from ml_service.application import training_service
from ml_service.application.training_progress import initial_progress
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


def test_retomadas_sucessivas_reutilizam_a_mesma_raiz_de_checkpoints(tmp_path, monkeypatch):
    class DeferredThread:
        def __init__(self, target, args, daemon):
            self.target = target
            self.args = args
            self.daemon = daemon

        def start(self):
            return None

    monkeypatch.setattr(training_service.threading, 'Thread', DeferredThread)
    progress = initial_progress()
    for stage in progress['stages'][:5]:
        stage['status'] = 'completed'
    progress['stages'][5]['status'] = 'failed'
    progress['current_stage'] = 'feature_selection'

    store = TrainingJobStore(str(tmp_path))
    store.save({
        'id': 'job-original',
        'version': 'v2.0.0',
        'status': 'failed',
        'requested_by': 'engineer',
        'parameters': {'checkpoint_job_id': 'job-original'},
        'progress': progress,
        'created_at': '2026-09-14T12:00:00+00:00',
        'started_at': '2026-09-14T12:01:00+00:00',
        'finished_at': '2026-09-14T12:02:00+00:00',
        'error': 'erro no SHAP',
    })
    service = TrainingService(store)

    first_resume = service.resume('job-original', 'engineer')
    first_resume['status'] = 'failed'
    first_resume['progress']['stages'][5]['status'] = 'failed'
    first_resume['progress']['current_stage'] = 'feature_selection'
    store.save(first_resume)
    second_resume = service.resume(first_resume['id'], 'engineer')

    assert first_resume['parameters']['checkpoint_job_id'] == 'job-original'
    assert second_resume['parameters']['checkpoint_job_id'] == 'job-original'
    assert second_resume['parameters']['resumed_from_job_id'] == first_resume['id']
