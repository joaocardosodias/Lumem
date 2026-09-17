import json
from datetime import datetime, timezone
from pathlib import Path


STAGE_DEFINITIONS = (
    ('snapshot_acquisition', 'Obter snapshots de treinamento'),
    ('data_loading', 'Carregar painel e manutenções'),
    ('feature_engineering', 'Preparar features'),
    ('temporal_split', 'Separar treino e teste no tempo'),
    ('initial_training', 'Treinar modelo inicial'),
    ('feature_selection', 'Selecionar features com SHAP'),
    ('final_training', 'Treinar modelo final'),
    ('evaluation', 'Calcular métricas e thresholds'),
    ('artifact_publish', 'Publicar artefato e manifest'),
    ('catalog_registration', 'Registrar modelo candidato'),
)


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


def initial_progress() -> dict:
    return {
        'current_stage': 'queued',
        'completed_steps': 0,
        'total_steps': len(STAGE_DEFINITIONS),
        'percent': 0,
        'updated_at': _now(),
        'stages': [
            {
                'key': key,
                'label': label,
                'status': 'pending',
                'started_at': None,
                'finished_at': None,
                'duration_ms': None,
                'detail': None,
            }
            for key, label in STAGE_DEFINITIONS
        ],
    }


def resume_progress(previous: dict, failed_stage: str) -> dict:
    """Build a fresh progress document that preserves only successful predecessors."""
    progress = initial_progress()
    previous_by_key = {
        stage.get('key'): stage
        for stage in (previous.get('stages') or [])
        if stage.get('key')
    }
    stage_keys = [key for key, _ in STAGE_DEFINITIONS]
    if failed_stage not in stage_keys:
        raise KeyError(f'checkpoint de retomada desconhecido: {failed_stage}')

    resume_index = stage_keys.index(failed_stage)
    for index, stage in enumerate(progress['stages']):
        if index >= resume_index:
            continue
        previous_stage = previous_by_key.get(stage['key']) or {}
        if previous_stage.get('status') not in {'completed', 'skipped'}:
            raise ValueError(f'a etapa anterior {stage["key"]} não possui checkpoint concluído')
        stage.update(previous_stage)
        stage['status'] = 'completed'
        stage['detail'] = 'Checkpoint reutilizado na retomada.'

    progress['current_stage'] = 'queued'
    return refresh_summary(progress)


def _parse_time(value: str | None):
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace('Z', '+00:00'))
    except ValueError:
        return None


def refresh_summary(progress: dict) -> dict:
    stages = progress.get('stages') or []
    completed = sum(stage.get('status') in {'completed', 'skipped'} for stage in stages)
    total = len(stages) or len(STAGE_DEFINITIONS)
    progress['completed_steps'] = completed
    progress['total_steps'] = total
    progress['percent'] = round(completed * 100 / total) if total else 0
    progress['updated_at'] = _now()
    return progress


def update_stage(progress: dict, key: str, status: str, detail: str | None = None) -> dict:
    stage = next((item for item in progress.get('stages', []) if item.get('key') == key), None)
    if stage is None:
        raise KeyError(f'checkpoint de treinamento desconhecido: {key}')

    now = _now()
    if status == 'running' and not stage.get('started_at'):
        stage['started_at'] = now
    if status in {'completed', 'failed', 'skipped'}:
        stage['started_at'] = stage.get('started_at') or now
        stage['finished_at'] = now
        started_at = _parse_time(stage.get('started_at'))
        finished_at = _parse_time(now)
        if started_at and finished_at:
            stage['duration_ms'] = max(0, round((finished_at - started_at).total_seconds() * 1000))
    stage['status'] = status
    stage['detail'] = detail
    progress['current_stage'] = key
    return refresh_summary(progress)


def fail_current_stage(progress: dict, detail: str) -> dict:
    key = progress.get('current_stage')
    if not key or key == 'queued':
        stages = progress.get('stages') or []
        if not stages:
            return progress
        key = stages[0].get('key')
    return update_stage(progress, key, 'failed', detail)


class ProgressReporter:
    def __init__(self, path: str | None):
        self.path = Path(path) if path else None
        self.progress = self._load_existing() or initial_progress()
        self._write()

    def _load_existing(self) -> dict | None:
        if self.path is None or not self.path.is_file():
            return None
        try:
            loaded = json.loads(self.path.read_text(encoding='utf-8'))
        except (OSError, ValueError):
            return None
        return loaded if isinstance(loaded.get('stages'), list) else None

    def _write(self) -> None:
        if self.path is None:
            return
        self.path.parent.mkdir(parents=True, exist_ok=True)
        temporary = self.path.with_suffix(self.path.suffix + '.tmp')
        temporary.write_text(json.dumps(self.progress, ensure_ascii=False), encoding='utf-8')
        temporary.replace(self.path)

    def advance(self, key: str, detail: str | None = None) -> None:
        current = self.progress.get('current_stage')
        if current and current != 'queued':
            active = next(
                (item for item in self.progress['stages'] if item.get('key') == current),
                None,
            )
            if current == key and active and active.get('status') == 'running':
                if detail is not None:
                    active['detail'] = detail
                    refresh_summary(self.progress)
                self._write()
                return
            if active and active.get('status') == 'running':
                update_stage(self.progress, current, 'completed')
        update_stage(self.progress, key, 'running', detail)
        self._write()

    def skip(self, keys: list[str], detail: str) -> None:
        for key in keys:
            update_stage(self.progress, key, 'skipped', detail)
        self._write()

    def complete_current(self, detail: str | None = None) -> None:
        current = self.progress.get('current_stage')
        if current and current != 'queued':
            update_stage(self.progress, current, 'completed', detail)
            self._write()
