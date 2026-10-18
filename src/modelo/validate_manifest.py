import argparse
import hashlib
import json
import math
from datetime import date
from pathlib import Path

import joblib


def file_hash(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open('rb') as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def validate(manifest_path: Path, artifact_path: Path) -> None:
    manifest = json.loads(manifest_path.read_text(encoding='utf-8'))
    for field in (
        'model_version', 'model_sha256', 'dataset_sha256', 'artifact_uri', 'algorithm',
        'target', 'features', 'train_end', 'test_start', 'test_end', 'test_rows',
        'positive_rows', 'recommended_threshold', 'threshold_metrics',
    ):
        if field not in manifest:
            raise ValueError(f'manifest sem {field}')
    for field in ('model_version', 'artifact_uri', 'algorithm', 'target'):
        if not manifest.get(field):
            raise ValueError(f'manifest sem {field}')
    for field in ('model_sha256', 'dataset_sha256'):
        value = manifest[field]
        if len(value) != 64 or any(character not in '0123456789abcdef' for character in value.lower()):
            raise ValueError(f'{field} não é um SHA-256 válido')
    if file_hash(artifact_path) != manifest['model_sha256']:
        raise ValueError('SHA-256 do artefato difere do manifest')

    features = manifest['features']
    if (
        not isinstance(features, list)
        or not features
        or any(not isinstance(feature, str) or not feature for feature in features)
        or len(set(features)) != len(features)
    ):
        raise ValueError('lista de features inválida ou duplicada')

    train_end = date.fromisoformat(manifest['train_end'])
    test_start = date.fromisoformat(manifest['test_start'])
    test_end = date.fromisoformat(manifest['test_end'])
    if train_end >= test_start or test_start > test_end:
        raise ValueError('períodos de treino e teste estão sobrepostos ou invertidos')
    test_rows = int(manifest['test_rows'])
    positive_rows = int(manifest['positive_rows'])
    if test_rows <= 0 or positive_rows < 0 or positive_rows > test_rows:
        raise ValueError('contagens do conjunto de teste são inválidas')

    bundle = joblib.load(artifact_path)
    if list(bundle.get('features', [])) != features:
        raise ValueError('ordem das features difere entre artefato e manifest')

    seen = set()
    for point in manifest['threshold_metrics']:
        threshold = float(point['threshold'])
        if threshold in seen:
            raise ValueError(f'threshold duplicado: {threshold}')
        seen.add(threshold)
        tp = int(point['true_positives'])
        fp = int(point['false_positives'])
        fn = int(point['false_negatives'])
        tn = int(point['true_negatives'])
        precision = float(point['precision'])
        recall = float(point['recall'])
        f1 = float(point['f1_score'])
        alert_rate = float(point['alert_rate'])
        if any(value < 0 for value in (tp, fp, fn, tn)):
            raise ValueError(f'matriz negativa no threshold {threshold}')
        if any(not 0 <= value <= 1 for value in (threshold, precision, recall, f1, alert_rate)):
            raise ValueError(f'métrica fora de [0,1] no threshold {threshold}')
        expected_precision = tp / (tp + fp) if tp + fp else 0
        expected_recall = tp / (tp + fn) if tp + fn else 0
        expected_f1 = 2 * expected_precision * expected_recall / (expected_precision + expected_recall) if expected_precision + expected_recall else 0
        if not math.isclose(precision, expected_precision, abs_tol=1e-9):
            raise ValueError(f'precision divergente no threshold {threshold}')
        if not math.isclose(recall, expected_recall, abs_tol=1e-9):
            raise ValueError(f'recall divergente no threshold {threshold}')
        if not math.isclose(f1, expected_f1, abs_tol=1e-9):
            raise ValueError(f'F1 divergente no threshold {threshold}')
        if int(point['alert_count']) != tp + fp:
            raise ValueError(f'alert_count divergente no threshold {threshold}')
        total = tp + fp + fn + tn
        if total != test_rows:
            raise ValueError(f'tamanho do teste divergente no threshold {threshold}')
        if tp + fn != positive_rows:
            raise ValueError(f'quantidade positiva divergente no threshold {threshold}')
        expected_alert_rate = (tp + fp) / total if total else 0
        if not math.isclose(alert_rate, expected_alert_rate, abs_tol=1e-9):
            raise ValueError(f'alert_rate divergente no threshold {threshold}')

    expected_thresholds = {round(step / 100, 2) for step in range(5, 96)}
    normalized_thresholds = {round(value, 2) for value in seen}
    if len(seen) != 91 or normalized_thresholds != expected_thresholds:
        raise ValueError('curva deve conter cada incremento de 0.01 entre 0.05 e 0.95')
    recommended = round(float(manifest['recommended_threshold']), 2)
    if recommended not in normalized_thresholds:
        raise ValueError('threshold recomendado não pertence à curva')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('manifest', type=Path)
    parser.add_argument('artifact', type=Path)
    args = parser.parse_args()
    validate(args.manifest, args.artifact)
    print(f'manifest válido: {args.manifest}')
