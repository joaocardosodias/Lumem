"""Read-only check of the real model, imported histories and persisted predictions."""
import json
import math
import sys

sys.path.insert(0, '/app')
from sqlalchemy import text
from ml_service.adapters.postgres_feature_repository import PostgresFeatureRepository
from ml_service.adapters.model_registry import registry_from_environment
from ml_service.application.predict_use_case import PredictUseCase
from ml_service.domain.feature_builder import construir_features

repo = PostgresFeatureRepository()
use_case = PredictUseCase(repo, registry_from_environment())
model = use_case.load()
doors = [
    {'station_code': 'FAL', 'platform': 'P1', 'door_number': 4},
    {'station_code': 'PIN', 'platform': 'P2', 'door_number': 8},
]
matrix, reference = construir_features(doors, model.bundle, repo)
probabilities = model.predict_probabilities(matrix)
report = {'model_version': model.version, 'model_sha256': model.manifest['model_sha256'], 'features_count': len(matrix.columns), 'doors': []}
with repo.engine.connect() as connection:
    report['counts'] = {
        table: connection.execute(text('SELECT count(*) FROM ' + table)).scalar()
        for table in ['telemetry_alarms', 'maintenance_events', 'predictions', 'prediction_runs', 'inspections']
    }
    report['jobs'] = [dict(row._mapping) for row in connection.execute(text('SELECT kind, state::text, count(*) AS total FROM river_job GROUP BY kind, state ORDER BY kind, state'))]
    for index, door in enumerate(doors):
        values = matrix.iloc[index]
        assert values['n_total_7d'] == 2, f'Imported telemetry missing from features: {door}: {values["n_total_7d"]}'
        saved = connection.execute(text('''
            SELECT p.probability, p.will_fail FROM predictions p
            JOIN prediction_runs r ON r.id = p.run_id
            WHERE p.station_code = :station_code AND p.platform = :platform
              AND p.door_number = :door_number
            ORDER BY r.created_at DESC LIMIT 1
        '''), door).mappings().one()
        assert math.isclose(saved['probability'], probabilities[index], rel_tol=1e-6)
        report['doors'].append({**door, 'alarms_7d': float(values['n_total_7d']), 'probability': float(probabilities[index]), 'saved_probability': saved['probability'], 'will_fail': saved['will_fail']})
assert report['counts']['telemetry_alarms'] == 4
assert report['counts']['maintenance_events'] == 5
print(json.dumps(report, indent=2))
