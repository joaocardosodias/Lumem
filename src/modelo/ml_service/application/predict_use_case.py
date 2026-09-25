import threading

from ml_service.domain.feature_builder import construir_features
from ml_service.domain.predictor import load_model


class PredictUseCase:
    def __init__(self, feature_repository, model_registry):
        self.feature_repository = feature_repository
        self.model_registry = model_registry
        self._model = None
        self._lock = threading.Lock()

    def load(self):
        with self._lock:
            if self._model is None:
                path, manifest = self.model_registry.materialize_active()
                self._model = load_model(path, manifest)
        return self._model

    def ready(self) -> bool:
        try:
            self.load()
            return True
        except Exception:
            return False

    def execute(self, doors: list[dict]) -> dict:
        model = self.load()
        matrix, reference_date = construir_features(
            doors, model.bundle, self.feature_repository
        )
        probabilities = model.predict_probabilities(matrix)
        return {
            'model_version': model.version,
            'reference_date': reference_date.isoformat(),
            'predictions': [
                {
                    'station_code': door['station_code'],
                    'platform': door['platform'],
                    'door_number': door['door_number'],
                    'probability': float(probability),
                }
                for door, probability in zip(doors, probabilities)
            ],
        }

    def metadata(self) -> dict:
        model = self.load()
        manifest = dict(model.manifest)
        manifest.setdefault('features', model.features)
        manifest['features_count'] = len(model.features)
        return manifest

    def activate(self, version: str) -> dict:
        with self._lock:
            candidate_path, candidate_manifest = self.model_registry.materialize_version(version)
            candidate = load_model(candidate_path, candidate_manifest)
            if candidate.version != version:
                raise ValueError('candidato materializado pertence a outra versão')
            manifest = self.model_registry.activate(version)
            if manifest.get('model_version') != version:
                raise ValueError('registry não ativou a versão solicitada')
            self._model = candidate
            return manifest
