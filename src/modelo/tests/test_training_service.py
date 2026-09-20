import json
import threading
import time

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


def test_cancela_job_na_fila_e_preserva_checkpoints(tmp_path, monkeypatch):
    class DeferredThread:
        def __init__(self, target, args, daemon):
            self.target = target
            self.args = args
            self.daemon = daemon

        def start(self):
            return None

    monkeypatch.setattr(training_service.threading, 'Thread', DeferredThread)
    service = TrainingService(TrainingJobStore(str(tmp_path)))
    job = service.start('engineer')

    cancelled = service.cancel(job['id'], 'engineer')

    assert cancelled['status'] == 'cancelled'
    assert cancelled['cancelled_by'] == 'engineer'
    assert cancelled['finished_at']
    assert cancelled['progress']['stages'][0]['status'] == 'cancelled'
    assert service._has_active_job() is False


def test_retomada_de_job_cancelado_reabre_etapa_interrompida(tmp_path, monkeypatch):
    class DeferredThread:
        def __init__(self, target, args, daemon):
            self.target = target
            self.args = args
            self.daemon = daemon

        def start(self):
            return None

    monkeypatch.setattr(training_service.threading, 'Thread', DeferredThread)
    progress = initial_progress()
    progress['stages'][0]['status'] = 'completed'
    progress['stages'][1]['status'] = 'cancelled'
    progress['current_stage'] = 'data_loading'
    store = TrainingJobStore(str(tmp_path))
    store.save({
        'id': 'job-cancelled',
        'version': 'v2.0.0',
        'status': 'cancelled',
        'requested_by': 'engineer',
        'parameters': {'checkpoint_job_id': 'job-cancelled'},
        'progress': progress,
        'created_at': '2026-09-17T12:00:00+00:00',
        'started_at': '2026-09-17T12:01:00+00:00',
        'finished_at': '2026-09-17T12:02:00+00:00',
        'error': None,
    })
    service = TrainingService(store)

    resumed = service.resume('job-cancelled', 'engineer')

    assert resumed['status'] == 'queued'
    assert resumed['parameters']['resume_from_stage'] == 'data_loading'
    assert resumed['progress']['stages'][0]['status'] == 'completed'
    assert resumed['progress']['stages'][1]['status'] == 'pending'


def test_worker_separado_consume_fila_e_recebe_cancelamento(tmp_path, monkeypatch):
    monkeypatch.setenv('MODEL_TRAIN_COMMAND', '/bin/sleep 30')
    api_service = TrainingService(
        TrainingJobStore(str(tmp_path)),
        auto_start=False,
        recover_interrupted=False,
    )
    worker_service = TrainingService(
        TrainingJobStore(str(tmp_path)),
        auto_start=False,
        recover_interrupted=False,
    )
    job = api_service.start('engineer')
    worker_thread = threading.Thread(target=worker_service.run_next)
    worker_thread.start()

    deadline = time.monotonic() + 3
    while api_service.get(job['id'])['status'] != 'running' and time.monotonic() < deadline:
        time.sleep(0.02)

    api_service.cancel(job['id'], 'engineer')
    worker_thread.join(timeout=5)

    assert worker_thread.is_alive() is False
    assert api_service.get(job['id'])['status'] == 'cancelled'
