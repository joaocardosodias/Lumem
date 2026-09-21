import os
from datetime import date, timedelta
from urllib.parse import parse_qs, urlencode, urlparse, urlunparse

import pandas as pd
from sqlalchemy import create_engine, text

from ml_service.ports.feature_repository import FeatureRepository


def _clean_database_url(url: str) -> str:
    parsed = urlparse(url)
    query = parse_qs(parsed.query)
    query.pop('default_query_exec_mode', None)
    return urlunparse(parsed._replace(query=urlencode(query, doseq=True)))


class PostgresFeatureRepository(FeatureRepository):
    """Adaptador somente leitura. Apenas este arquivo conhece as views de ML."""

    def __init__(self, database_url: str | None = None):
        self._database_url = database_url or os.getenv('ML_DATABASE_URL')
        if not self._database_url and os.getenv('ENV', 'development') != 'production':
            self._database_url = os.getenv('DATABASE_URL')
        self._engine = None

    @property
    def engine(self):
        if self._engine is None:
            if not self._database_url:
                raise RuntimeError('ML_DATABASE_URL é obrigatória e deve usar uma credencial somente leitura')
            self._engine = create_engine(_clean_database_url(self._database_url), pool_pre_ping=True)
        return self._engine

    def obter_data_referencia(self, estacoes: list[str]) -> date:
        hoje = date.today()
        with self.engine.connect() as connection:
            result = connection.execute(
                text('SELECT MAX(event_timestamp) FROM ml_telemetry_v1 WHERE station_code = ANY(:estacoes)'),
                {'estacoes': estacoes},
            ).scalar()
        if result is not None:
            max_date = pd.to_datetime(result).date()
            if (hoje - max_date).days > 60:
                return max_date + timedelta(days=1)
        return hoje

    def buscar_telemetria_recente(
        self, estacoes: list[str], hoje: date, janela_dias: int = 60
    ) -> pd.DataFrame:
        since = hoje - timedelta(days=janela_dias)
        with self.engine.connect() as connection:
            return pd.read_sql(
                text('''
                    SELECT station_code, platform, door_number, alarm_type,
                           duration_seconds, event_timestamp
                    FROM ml_telemetry_v1
                    WHERE station_code = ANY(:estacoes)
                      AND event_timestamp >= :desde
                      AND event_timestamp <= :hoje
                '''),
                connection,
                params={
                    'estacoes': estacoes,
                    'desde': since.isoformat(),
                    'hoje': f'{hoje.isoformat()} 23:59:59',
                },
            )

    def buscar_manutencoes(self, estacoes: list[str]) -> pd.DataFrame:
        with self.engine.connect() as connection:
            return pd.read_sql(
                text('''
                    SELECT order_type, station_code, installation_location,
                           start_timestamp, end_timestamp
                    FROM ml_maintenance_v1
                    WHERE station_code = ANY(:estacoes)
                '''),
                connection,
                params={'estacoes': estacoes},
            )

    def listar_portas_ativas(self) -> list[dict]:
        with self.engine.connect() as connection:
            rows = connection.execute(text('''
                SELECT station_code, platform, door_number
                FROM ml_active_doors_v1
                ORDER BY station_code, platform, door_number
            '''))
            return [dict(row._mapping) for row in rows]
