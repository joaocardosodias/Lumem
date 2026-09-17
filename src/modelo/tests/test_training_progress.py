import json
import sys
from pathlib import Path


MODEL_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(MODEL_ROOT))

from ml_service.application.training_progress import (
    ProgressReporter,
    fail_current_stage,
    initial_progress,
)


def test_reporter_persiste_checkpoints_em_ordem(tmp_path):
    target = tmp_path / 'progress.json'
    reporter = ProgressReporter(str(target))

    reporter.advance('snapshot_acquisition')
    reporter.advance('data_loading')

    progress = json.loads(target.read_text(encoding='utf-8'))
    assert progress['current_stage'] == 'data_loading'
    assert progress['completed_steps'] == 1
    assert progress['percent'] == 10
    assert progress['stages'][0]['status'] == 'completed'
    assert progress['stages'][1]['status'] == 'running'


def test_falha_fica_associada_ao_checkpoint_ativo():
    progress = initial_progress()
    progress['current_stage'] = 'feature_selection'
    progress['stages'][5]['status'] = 'running'

    failed = fail_current_stage(progress, 'erro no SHAP')

    assert failed['stages'][5]['status'] == 'failed'
    assert failed['stages'][5]['detail'] == 'erro no SHAP'


def test_etapas_opcionais_podem_ser_marcadas_como_ignoradas(tmp_path):
    reporter = ProgressReporter(str(tmp_path / 'progress.json'))
    reporter.skip(['initial_training', 'feature_selection'], 'artefato existente')

    assert reporter.progress['completed_steps'] == 2
    assert reporter.progress['stages'][4]['status'] == 'skipped'
    assert reporter.progress['stages'][5]['status'] == 'skipped'
