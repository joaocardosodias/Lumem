import json
import os
import shlex
import subprocess
import sys
import threading
import uuid
from datetime import datetime, timezone
from pathlib import Path


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
                'parameters': {'random_state': 42, 'evaluation_horizon_days': 7},
                'created_at': _now(),
                'started_at': None,
                'finished_at': None,
                'error': None,
                'manifest': None,
            }
            self.store.save(job)
            threading.Thread(target=self._run, args=(job_id,), daemon=True).start()
        return job

    def get(self, job_id: str) -> dict:
        return self.store.get(job_id)

    def _run(self, job_id: str) -> None:
        job = self.store.get(job_id)
        job['status'] = 'running'
        job['started_at'] = _now()
        self.store.save(job)
        try:
            project_dir = Path(__file__).resolve().parents[2]
            template = os.getenv(
                'MODEL_TRAIN_COMMAND',
                f'{shlex.quote(sys.executable)} exportar_modelo.py --version {{version}}',
            )
            command = [part.format(version=job['version'], job_id=job_id) for part in shlex.split(template)]
            result = subprocess.run(
                command,
                cwd=project_dir,
                check=True,
                capture_output=True,
                text=True,
                timeout=int(os.getenv('MODEL_TRAIN_TIMEOUT_SECONDS', '3600')),
            )
            manifest_path = project_dir / 'artifacts' / 'models' / job['version'] / 'manifest.json'
            if not manifest_path.is_file():
                raise RuntimeError('treinamento terminou sem gerar o manifest versionado')
            job['manifest'] = json.loads(manifest_path.read_text(encoding='utf-8'))
            if job['manifest'].get('model_version') != job['version']:
                raise RuntimeError('manifest gerado pertence a outra versão')
            job['status'] = 'completed'
            job['output_tail'] = result.stdout[-2000:]
        except Exception as error:
            job['status'] = 'failed'
            if isinstance(error, subprocess.CalledProcessError):
                detail = (error.stderr or error.stdout or str(error))[-2000:]
            else:
                detail = str(error)
            job['error'] = detail
        finally:
            job['finished_at'] = _now()
            self.store.save(job)
