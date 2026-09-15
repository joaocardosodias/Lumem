from dataclasses import dataclass
from pathlib import Path

import joblib
import numpy as np
import pandas as pd


@dataclass(frozen=True)
class LoadedModel:
    artifact_path: Path
    bundle: dict
    manifest: dict

    @property
    def version(self) -> str:
        return str(self.manifest.get('model_version') or self.bundle.get('version') or 'unversioned')

    @property
    def features(self) -> list[str]:
        return list(self.bundle['features'])

    def predict_probabilities(self, matrix: pd.DataFrame) -> np.ndarray:
        if list(matrix.columns) != self.features:
            raise ValueError('ordem de features incompatível com o artefato')
        probabilities = self.bundle['model'].predict_proba(matrix)[:, 1]
        if not np.isfinite(probabilities).all() or ((probabilities < 0) | (probabilities > 1)).any():
            raise ValueError('o modelo retornou probabilidades fora do intervalo [0,1]')
        return probabilities


def load_model(path: Path, manifest: dict) -> LoadedModel:
    bundle = joblib.load(path)
    required = {'model', 'features', 'est_categorias'}
    missing = required.difference(bundle)
    if missing:
        raise ValueError(f'artefato inválido; campos ausentes: {sorted(missing)}')

    expected_features = manifest.get('features')
    if expected_features and list(expected_features) != list(bundle['features']):
        raise ValueError('manifest e artefato possuem ordens de features diferentes')
    return LoadedModel(path, bundle, manifest)
