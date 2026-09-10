import os
from datetime import date, timedelta
from urllib.parse import parse_qs, urlencode, urlparse, urlunparse

import pandas as pd
from sqlalchemy import create_engine, text

_engine = None


def _clean_database_url(url: str) -> str:
    parsed = urlparse(url)
    qs = parse_qs(parsed.query)
    unsupported = {'default_query_exec_mode'}
    clean_qs = {k: v for k, v in qs.items() if k not in unsupported}
    new_query = urlencode(clean_qs, doseq=True)
    return urlunparse(parsed._replace(query=new_query))


def get_engine():
    global _engine
    if _engine is None:
        raw_url = os.environ['DATABASE_URL']
        clean_url = _clean_database_url(raw_url)
        _engine = create_engine(clean_url)
    return _engine


def obter_data_referencia(estacoes: list[str]) -> date:
    hoje = date.today()
    try:
        with get_engine().connect() as conn:
            res = conn.execute(text("""
                SELECT MAX(event_timestamp)
                FROM telemetry_alarms
                WHERE station_code = ANY(:estacoes)
            """), {'estacoes': estacoes}).scalar()
            if res is not None:
                max_dt = pd.to_datetime(res).date()
                if (hoje - max_dt).days > 60:
                    return max_dt + timedelta(days=1)
    except Exception:
        pass
    return hoje


def buscar_telemetria_recente(estacoes: list[str], hoje: date, janela_dias: int = 60) -> pd.DataFrame:
    desde = hoje - timedelta(days=janela_dias)
    with get_engine().connect() as conn:
        return pd.read_sql(text("""
            SELECT station_code, platform, door_number, alarm_type, duration_seconds, event_timestamp
            FROM telemetry_alarms
            WHERE station_code = ANY(:estacoes)
              AND event_timestamp >= :desde
              AND event_timestamp <= :hoje
        """), conn, params={'estacoes': estacoes, 'desde': str(desde), 'hoje': f"{hoje} 23:59:59"})


def buscar_manutencoes(estacoes: list[str]) -> pd.DataFrame:
    with get_engine().connect() as conn:
        return pd.read_sql(text("""
            SELECT order_type, station_code, installation_location, start_timestamp, end_timestamp
            FROM maintenance_events
            WHERE station_code = ANY(:estacoes)
        """), conn, params={'estacoes': estacoes})
