import json
import os
import shlex
import subprocess
import sys
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

from ml_service.application.training_progress import (
    fail_current_stage,
    initial_progress,
    resume_progress,
)


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


class TrainingJobStore:
    def __init__(self, root: str | None = None):
        self.root = Path(root or os.getenv('TRAINING_JOB_DIR', '/tmp/g04-training-jobs'))
        self.root.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()

    def recover_interrupted(self) -> None:
        for target in self.root.glob('*.json'):
            try:
                job = json.loads(target.read_text(encoding='utf-8'))
            except (OSError, ValueError):
                continue
            if job.get('status') not in {'queued', 'running'}:
                continue
            job['status'] = 'failed'
            job['error'] = 'o serviço de treinamento reiniciou antes da conclusão do processo'
            job['progress'] = fail_current_stage(
                job.get('progress') or initial_progress(),
                job['error'],
            )
            job['finished_at'] = _now()
            self.save(job)

    def save(self, job: dict) -> None:
        with self._lock:
            target = self.root / f"{job['id']}.json"
            temporary = target.with_suffix('.tmp')
            temporary.write_text(json.dumps(job, ensure_ascii=False), encoding='utf-8')
            temporary.replace(target)

    def get(self, job_id: str) -> dict:
        target = self.root / f'{job_id}.json'
        if not target.is_file():
            raise LookupError('job de treinamento não encontrado')
        return json.loads(target.read_text(encoding='utf-8'))


class TrainingService:
    """Executa um comando real de treinamento e persiste o estado fora da memória do worker."""

    def __init__(self, store: TrainingJobStore | None = None):
        self.store = store or TrainingJobStore()
        self.store.recover_interrupted()
        self._start_lock = threading.Lock()

    def _has_active_job(self) -> bool:
        for target in self.store.root.glob('*.json'):
            try:
                status = json.loads(target.read_text(encoding='utf-8')).get('status')
            except (OSError, ValueError):
                continue
            if status in {'queued', 'running'}:
                return True
        return False

    def start(self, requested_by: str = '') -> dict:
        with self._start_lock:
            if self._has_active_job():
                raise RuntimeError('já existe um treinamento em execução')
            job_id = str(uuid.uuid4())
            version = f"v{datetime.now(timezone.utc).strftime('%Y.%m.%d.%H%M%S')}-{job_id[:8]}"
            job = {
                'id': job_id,
                'version': version,
                'status': 'queued',
                'requested_by': requested_by,
                'parameters': {
                    'random_state': 42,
                    'evaluation_horizon_days': 7,
                    'checkpoint_job_id': job_id,
                },
                'created_at': _now(),
                'started_at': None,
                'finished_at': None,
                'error': None,
                'manifest': None,
                'progress': initial_progress(),
            }
            self.store.save(job)
            threading.Thread(target=self._run, args=(job_id, None), daemon=True).start()
        return job

    def resume(self, source_job_id: str, requested_by: str = '') -> dict:
        with self._start_lock:
            if self._has_active_job():
                raise RuntimeError('já existe um treinamento em execução')
            source = self.store.get(source_job_id)
            if source.get('status') != 'failed':
                raise ValueError('somente treinamentos com falha podem ser retomados')

            failed_stage = next(
                (
                    stage.get('key')
                    for stage in source.get('progress', {}).get('stages', [])
                    if stage.get('status') == 'failed'
                ),
                None,
            )
            if not failed_stage:
                raise ValueError('o treinamento não informa a etapa que falhou')

            job_id = str(uuid.uuid4())
            parameters = dict(source.get('parameters') or {})
            parameters.update({
                'resume_from_stage': failed_stage,
                'resumed_from_job_id': source_job_id,
                'checkpoint_job_id': parameters.get('checkpoint_job_id') or source_job_id,
            })
            job = {
                'id': job_id,
                'version': source['version'],
                'status': 'queued',
                'requested_by': requested_by,
                'parameters': parameters,
                'created_at': _now(),
                'started_at': None,
                'finished_at': None,
                'error': None,
                'manifest': None,
                'progress': resume_progress(source.get('progress') or {}, failed_stage),
            }
            self.store.save(job)
            progress_dir = self.store.root / 'progress'
            progress_dir.mkdir(parents=True, exist_ok=True)
            (progress_dir / f'{job_id}.json').write_text(
                json.dumps(job['progress'], ensure_ascii=False),
                encoding='utf-8',
            )
            threading.Thread(
                target=self._run,
                args=(job_id, source_job_id),
                daemon=True,
            ).start()
        return job

    def get(self, job_id: str) -> dict:
        return self.store.get(job_id)

    def _run(self, job_id: str, resume_source_job_id: str | None) -> None:
        job = self.store.get(job_id)
        job['status'] = 'running'
        job['started_at'] = _now()
        self.store.save(job)
        try:
            project_dir = Path(__file__).resolve().parents[2]
            progress_dir = self.store.root / 'progress'
            log_dir = self.store.root / 'logs'
            progress_dir.mkdir(parents=True, exist_ok=True)
            log_dir.mkdir(parents=True, exist_ok=True)
            progress_path = progress_dir / f'{job_id}.json'
            log_path = log_dir / f'{job_id}.log'
            template = os.getenv(
                'MODEL_TRAIN_COMMAND',
                f'{shlex.quote(sys.executable)} exportar_modelo.py --version {{version}}',
            )
            command = [part.format(version=job['version'], job_id=job_id) for part in shlex.split(template)]
            environment = os.environ.copy()
            environment['MODEL_TRAIN_PROGRESS_PATH'] = str(progress_path)
            checkpoint_job_id = job.get('parameters', {}).get('checkpoint_job_id') or resume_source_job_id or job_id
            checkpoint_dir = self.store.root / 'checkpoints' / checkpoint_job_id
            checkpoint_dir.mkdir(parents=True, exist_ok=True)
            environment['MODEL_TRAIN_CHECKPOINT_DIR'] = str(checkpoint_dir)
            if resume_source_job_id:
                environment['MODEL_TRAIN_RESUME_FROM'] = str(job['parameters']['resume_from_stage'])
                environment['MODEL_TRAIN_RESUMED_FROM_JOB_ID'] = resume_source_job_id
            timeout_seconds = int(os.getenv('MODEL_TRAIN_TIMEOUT_SECONDS', '3600'))
            deadline = time.monotonic() + timeout_seconds

            with log_path.open('w+', encoding='utf-8') as output:
                process = subprocess.Popen(
                    command,
                    cwd=project_dir,
                    env=environment,
                    stdout=output,
                    stderr=subprocess.STDOUT,
                    text=True,
                )
                while process.poll() is None:
                    self._sync_progress(job, progress_path)
                    if time.monotonic() >= deadline:
                        process.kill()
                        process.wait()
                        raise TimeoutError(f'treinamento excedeu o limite de {timeout_seconds} segundos')
                    time.sleep(0.5)
                self._sync_progress(job, progress_path)
                output.flush()
                output.seek(0)
                output_tail = output.read()[-2000:]
                if process.returncode != 0:
                    raise RuntimeError(output_tail or f'treinamento terminou com código {process.returncode}')

            manifest_path = project_dir / 'artifacts' / 'models' / job['version'] / 'manifest.json'
            if not manifest_path.is_file():
                raise RuntimeError('treinamento terminou sem gerar o manifest versionado')
            job['manifest'] = json.loads(manifest_path.read_text(encoding='utf-8'))
            if job['manifest'].get('model_version') != job['version']:
                raise RuntimeError('manifest gerado pertence a outra versão')
            job['status'] = 'completed'
            job['output_tail'] = output_tail
        except Exception as error:
            job['status'] = 'failed'
            detail = str(error)[-2000:]
            job['error'] = detail
            job['progress'] = fail_current_stage(job.get('progress') or initial_progress(), detail)
        finally:
            job['finished_at'] = _now()
            self.store.save(job)

    def _sync_progress(self, job: dict, progress_path: Path) -> None:
        if not progress_path.is_file():
            return
        try:
            progress = json.loads(progress_path.read_text(encoding='utf-8'))
        except (OSError, ValueError):
            return
        if progress == job.get('progress'):
            return
        job['progress'] = progress
        self.store.save(job)
