import os
from datetime import date, timedelta

import pandas as pd
from sqlalchemy import create_engine, text

_engine = None


def get_engine():
    global _engine
    if _engine is None:
        _engine = create_engine(os.environ['DATABASE_URL'])
    return _engine


def buscar_telemetria_recente(estacoes: list[str], hoje: date, janela_dias: int = 60) -> pd.DataFrame:
    desde = hoje - timedelta(days=janela_dias)
    with get_engine().connect() as conn:
        return pd.read_sql(text("""
            SELECT station_code, platform, door_number, alarm_type, duration_seconds, event_timestamp
            FROM telemetry_alarms
            WHERE station_code = ANY(:estacoes)
              AND event_timestamp >= :desde
              AND event_timestamp < :hoje
        """), conn, params={'estacoes': estacoes, 'desde': str(desde), 'hoje': str(hoje)})


def buscar_manutencoes(estacoes: list[str]) -> pd.DataFrame:
    with get_engine().connect() as conn:
        return pd.read_sql(text("""
            SELECT order_type, station_code, installation_location, start_timestamp, end_timestamp
            FROM maintenance_events
            WHERE station_code = ANY(:estacoes)
        """), conn, params={'estacoes': estacoes})
