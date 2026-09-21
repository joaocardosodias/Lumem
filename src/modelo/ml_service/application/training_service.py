import json
import os
import signal
import shlex
import subprocess
import sys
import threading
import time
import uuid
from contextlib import contextmanager
from datetime import datetime, timezone
from pathlib import Path

import fcntl

from ml_service.application.training_progress import (
    cancel_current_stage,
    fail_current_stage,
    initial_progress,
    resume_progress,
)


class TrainingCancelled(Exception):
    pass


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


class TrainingJobStore:
    def __init__(self, root: str | None = None):
        self.root = Path(root or os.getenv('TRAINING_JOB_DIR', '/tmp/g04-training-jobs'))
        self.root.mkdir(parents=True, exist_ok=True)
        self._lock = threading.Lock()

    @contextmanager
    def _exclusive(self):
        lock_path = self.root / '.training-jobs.lock'
        with lock_path.open('a+', encoding='utf-8') as lock_file:
            fcntl.flock(lock_file.fileno(), fcntl.LOCK_EX)
            try:
                yield
            finally:
                fcntl.flock(lock_file.fileno(), fcntl.LOCK_UN)

    def _write_unlocked(self, job: dict) -> None:
        target = self.root / f"{job['id']}.json"
        temporary = target.with_suffix('.tmp')
        temporary.write_text(json.dumps(job, ensure_ascii=False), encoding='utf-8')
        temporary.replace(target)

    def recover_interrupted(self) -> None:
        for target in self.root.glob('*.json'):
            try:
                job = json.loads(target.read_text(encoding='utf-8'))
            except (OSError, ValueError):
                continue
            if job.get('status') != 'running':
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
            with self._exclusive():
                self._write_unlocked(job)

    def get(self, job_id: str) -> dict:
        target = self.root / f'{job_id}.json'
        if not target.is_file():
            raise LookupError('job de treinamento não encontrado')
        return json.loads(target.read_text(encoding='utf-8'))

    def claim_next(self) -> dict | None:
        with self._lock:
            with self._exclusive():
                queued = []
                for target in self.root.glob('*.json'):
                    try:
                        job = json.loads(target.read_text(encoding='utf-8'))
                    except (OSError, ValueError):
                        continue
                    if job.get('status') == 'queued':
                        queued.append(job)
                if not queued:
                    return None

                job = min(queued, key=lambda item: item.get('created_at') or '')
                job['status'] = 'running'
                job['started_at'] = _now()
                job['finished_at'] = None
                self._write_unlocked(job)
                return job

    def save_unless_cancelled(self, job: dict) -> bool:
        with self._lock:
            with self._exclusive():
                target = self.root / f"{job['id']}.json"
                if target.is_file():
                    try:
                        current = json.loads(target.read_text(encoding='utf-8'))
                    except (OSError, ValueError):
                        current = {}
                    if current.get('status') == 'cancelled' and job.get('status') != 'cancelled':
                        return False
                self._write_unlocked(job)
                return True


class TrainingService:
    """Executa um comando real de treinamento e persiste o estado fora da memória do worker."""

    def __init__(
        self,
        store: TrainingJobStore | None = None,
        auto_start: bool | None = None,
        recover_interrupted: bool | None = None,
    ):
        self.store = store or TrainingJobStore()
        external_mode = os.getenv('TRAINING_EXECUTION_MODE', 'inline').lower() == 'external'
        self.auto_start = not external_mode if auto_start is None else auto_start
        should_recover = self.auto_start if recover_interrupted is None else recover_interrupted
        if should_recover:
            self.store.recover_interrupted()
        self._start_lock = threading.Lock()
        self._process_lock = threading.Lock()
        self._processes: dict[str, subprocess.Popen] = {}
        self._cancelled_jobs: set[str] = set()

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
            if self.auto_start:
                threading.Thread(target=self._run, args=(job_id, None), daemon=True).start()
        return job

    def resume(self, source_job_id: str, requested_by: str = '') -> dict:
        with self._start_lock:
            if self._has_active_job():
                raise RuntimeError('já existe um treinamento em execução')
            source = self.store.get(source_job_id)
            if source.get('status') not in {'failed', 'cancelled'}:
                raise ValueError('somente treinamentos com falha ou cancelados podem ser retomados')

            interrupted_stage = next(
                (
                    stage.get('key')
                    for stage in source.get('progress', {}).get('stages', [])
                    if stage.get('status') in {'failed', 'cancelled'}
                ),
                None,
            )
            if not interrupted_stage:
                raise ValueError('o treinamento não informa a etapa interrompida')

            job_id = str(uuid.uuid4())
            parameters = dict(source.get('parameters') or {})
            parameters.update({
                'resume_from_stage': interrupted_stage,
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
                'progress': resume_progress(source.get('progress') or {}, interrupted_stage),
            }
            self.store.save(job)
            progress_dir = self.store.root / 'progress'
            progress_dir.mkdir(parents=True, exist_ok=True)
            (progress_dir / f'{job_id}.json').write_text(
                json.dumps(job['progress'], ensure_ascii=False),
                encoding='utf-8',
            )
            if self.auto_start:
                threading.Thread(
                    target=self._run,
                    args=(job_id, source_job_id),
                    daemon=True,
                ).start()
        return job

    def run_next(self) -> bool:
        job = self.store.claim_next()
        if job is None:
            return False
        resumed_from = (job.get('parameters') or {}).get('resumed_from_job_id')
        self._run(job['id'], resumed_from)
        return True

    def run_forever(self, poll_seconds: float = 1.0) -> None:
        interval = max(0.1, poll_seconds)
        while True:
            if not self.run_next():
                time.sleep(interval)

    def get(self, job_id: str) -> dict:
        return self.store.get(job_id)

    def cancel(self, job_id: str, requested_by: str = '') -> dict:
        with self._start_lock:
            job = self.store.get(job_id)
            if job.get('status') not in {'queued', 'running'}:
                raise ValueError('somente treinamentos na fila ou em execução podem ser cancelados')

            with self._process_lock:
                self._cancelled_jobs.add(job_id)
                process = self._processes.get(job_id)

            detail = 'Treinamento cancelado pelo usuário.'
            job['status'] = 'cancelled'
            job['cancelled_by'] = requested_by
            job['error'] = None
            job['progress'] = cancel_current_stage(
                job.get('progress') or initial_progress(),
                detail,
            )
            job['finished_at'] = _now()
            self.store.save(job)

            if process is not None:
                self._terminate_process(process)
            else:
                with self._process_lock:
                    self._cancelled_jobs.discard(job_id)
            return job

    def _is_cancelled(self, job_id: str) -> bool:
        with self._process_lock:
            locally_cancelled = job_id in self._cancelled_jobs
        if locally_cancelled:
            return True
        try:
            return self.store.get(job_id).get('status') == 'cancelled'
        except LookupError:
            return False

    @staticmethod
    def _terminate_process(process: subprocess.Popen) -> None:
        if process.poll() is not None:
            return
        try:
            os.killpg(process.pid, signal.SIGTERM)
        except (ProcessLookupError, PermissionError):
            if process.poll() is None:
                process.terminate()
        try:
            process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except (ProcessLookupError, PermissionError):
                if process.poll() is None:
                    process.kill()
            process.wait(timeout=2)

    def _run(self, job_id: str, resume_source_job_id: str | None) -> None:
        job = self.store.get(job_id)
        if self._is_cancelled(job_id) or job.get('status') == 'cancelled':
            with self._process_lock:
                self._cancelled_jobs.discard(job_id)
            return
        if job.get('status') == 'queued':
            job['status'] = 'running'
            job['started_at'] = _now()
            self.store.save(job)
        elif job.get('status') != 'running':
            return
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
                    start_new_session=True,
                )
                with self._process_lock:
                    self._processes[job_id] = process
                while process.poll() is None:
                    if self._is_cancelled(job_id):
                        self._terminate_process(process)
                        raise TrainingCancelled()
                    self._sync_progress(job, progress_path)
                    if time.monotonic() >= deadline:
                        process.kill()
                        process.wait()
                        raise TimeoutError(f'treinamento excedeu o limite de {timeout_seconds} segundos')
                    time.sleep(0.5)
                if self._is_cancelled(job_id):
                    raise TrainingCancelled()
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
        except TrainingCancelled:
            job['status'] = 'cancelled'
            job['error'] = None
            job['progress'] = cancel_current_stage(
                job.get('progress') or initial_progress(),
                'Treinamento cancelado pelo usuário.',
            )
        except Exception as error:
            job['status'] = 'failed'
            detail = str(error)[-2000:]
            job['error'] = detail
            job['progress'] = fail_current_stage(job.get('progress') or initial_progress(), detail)
        finally:
            job['finished_at'] = _now()
            self.store.save_unless_cancelled(job)
            with self._process_lock:
                self._processes.pop(job_id, None)
                self._cancelled_jobs.discard(job_id)

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
        if not self.store.save_unless_cancelled(job):
            with self._process_lock:
                self._cancelled_jobs.add(job['id'])
