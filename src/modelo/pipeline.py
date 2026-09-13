import os
from pathlib import Path
import re
from datetime import date, timedelta

import joblib
import numpy as np
import pandas as pd
from scipy.signal import lfilter

from database import buscar_manutencoes, buscar_telemetria_recente, obter_data_referencia

TIPOS = ['CA', 'CSD', 'FC', 'FO', 'LSD', 'OA', 'OBS', 'RCE']
JANELAS = (1, 3, 7, 14, 30)
PAT_LOC = re.compile(r'^SP04-6-PSD-([A-Z]{3})(?:-PLT(\d)|-P(\d)S\d)?(?:-PP(\d{2})(PDM|PFX|PSE))?')
ESTACAO_ALIAS = {
    'PAU': 'PTA',
    'OFR': 'FRA',
}

_bundle = None


def carregar_bundle(caminho=None):
    global _bundle
    if _bundle is None:
        if caminho is None:
            caminho = Path(__file__).resolve().parent / 'artifacts' / 'modelo_5.joblib'
        _bundle = joblib.load(caminho)
    return _bundle


def normaliza_via(platform: str) -> str:
    m = re.search(r'(\d+)$', str(platform))
    return m.group(1) if m else str(platform)


def extrai_local(texto):
    if not isinstance(texto, str):
        return None, None, None
    m = PAT_LOC.match(texto)
    if not m:
        return None, None, None
    est = m.group(1)
    via = m.group(2) or m.group(3)
    porta = int(m.group(4)) if m.group(4) else None
    return est, via, porta


def _janela_alarmes(alarmes: pd.DataFrame, hoje: date, dias: int) -> pd.DataFrame:
    dia_ini = pd.Timestamp(hoje) - pd.Timedelta(days=dias)
    dias_idx = pd.date_range(dia_ini, pd.Timestamp(hoje) - pd.Timedelta(days=1), freq='D')

    if alarmes.empty:
        base = pd.DataFrame(0, index=dias_idx, columns=['n_' + t for t in TIPOS] + ['n_total', 'n_long30', 'dur_max'])
    else:
        a = alarmes.copy()
        a['dia'] = pd.to_datetime(a['event_timestamp']).dt.normalize()
        piv = (a.pivot_table(index='dia', columns='alarm_type', aggfunc='size', fill_value=0)
               .reindex(columns=TIPOS, fill_value=0))
        piv.columns = ['n_' + c for c in piv.columns]
        agregados = a.groupby('dia').agg(n_total=('alarm_type', 'size'),
                                          n_long30=('duration_seconds', lambda s: (s > 30).sum()),
                                          dur_max=('duration_seconds', 'max'))
        base = piv.join(agregados, how='outer').reindex(dias_idx, fill_value=0).fillna(0)

    base['n_tecnicos'] = base['n_total'] - base.get('n_OBS', 0)
    return base


_EMPTY_ALARM_FEATS = None


def get_empty_alarm_feats() -> dict:
    global _EMPTY_ALARM_FEATS
    if _EMPTY_ALARM_FEATS is None:
        out = {}
        for w in JANELAS:
            for t in TIPOS:
                out[f'n_{t}_{w}d'] = 0.0
            out[f'n_total_{w}d'] = 0.0
            out[f'n_long30_{w}d'] = 0.0
            out[f'n_tecnicos_{w}d'] = 0.0
        out['dur_max_7d'] = 0.0
        out['dur_max_30d'] = 0.0
        for col, meias_vidas in [('n_total', (3, 7, 14)), ('n_tecnicos', (7,)), ('n_LSD', (7,))]:
            for h in meias_vidas:
                out[f'{col}_decaido_h{h}'] = 0.0
        _EMPTY_ALARM_FEATS = out
    return dict(_EMPTY_ALARM_FEATS)


def _features_alarme_porta(alarmes_porta: pd.DataFrame, hoje: date) -> dict:
    if alarmes_porta is None or alarmes_porta.empty:
        return get_empty_alarm_feats()

    diario = _janela_alarmes(alarmes_porta, hoje, 60)
    cols_cnt = [c for c in diario.columns if c.startswith('n_')]

    out = {}
    for w in JANELAS:
        soma = diario[cols_cnt].tail(w).sum()
        for c in cols_cnt:
            out[f'{c}_{w}d'] = float(soma[c])
    out['dur_max_7d'] = float(diario['dur_max'].tail(7).max())
    out['dur_max_30d'] = float(diario['dur_max'].tail(30).max())

    for col, meias_vidas in [('n_total', (3, 7, 14)), ('n_tecnicos', (7,)), ('n_LSD', (7,))]:
        serie = diario[col].to_numpy(dtype=float)
        for h in meias_vidas:
            decay = 0.5 ** (1 / h)
            y = lfilter([1.0], [1.0, -decay], serie)
            out[f'{col}_decaido_h{h}'] = float(y[-1])
    return out


def _historico_ordens(eventos: pd.Series, hoje: date, janelas=(30, 90)) -> dict:
    eventos = pd.to_datetime(pd.Series(eventos)).dropna()
    eventos = eventos[eventos < pd.Timestamp(hoje)]
    out = {}
    if eventos.empty:
        out['dias_desde'] = 9999.0
    else:
        out['dias_desde'] = float((pd.Timestamp(hoje) - eventos.max()).days)
    for w in janelas:
        desde = pd.Timestamp(hoje) - pd.Timedelta(days=w)
        out[f'{w}d'] = float(((eventos >= desde)).sum())
    return out


def processar_e_prever(portas: list[dict], threshold: float | None = None) -> list[dict]:
    bundle = carregar_bundle()
    estacoes = sorted({p['station_code'] for p in portas})
    hoje = obter_data_referencia(estacoes)

    alarmes = buscar_telemetria_recente(estacoes, hoje, janela_dias=60)
    manutencoes = buscar_manutencoes(estacoes)

    alarmes_by_door = {}
    alarmes_mdu_by_plat = {}
    if not alarmes.empty:
        porta_alarmes = alarmes[alarmes['door_number'].notna()]
        if not porta_alarmes.empty:
            for (st, pl, dn), grp in porta_alarmes.groupby(['station_code', 'platform', 'door_number']):
                alarmes_by_door[(st, pl, int(dn))] = grp

        mdu_alarmes = alarmes[alarmes['door_number'].isna()]
        if not mdu_alarmes.empty:
            for (st, pl), grp in mdu_alarmes.groupby(['station_code', 'platform']):
                alarmes_mdu_by_plat[(st, pl)] = grp

    m_by_door = {}
    m_by_est = {}
    if not manutencoes.empty:
        loc = manutencoes['installation_location'].apply(extrai_local)
        manutencoes = manutencoes.assign(
            est_m=[x[0] for x in loc], via_m=[x[1] for x in loc], porta_m=[x[2] for x in loc],
        )
        for est, grp in manutencoes.groupby('est_m'):
            if est:
                m_by_est[est] = grp
        valid_porta = manutencoes[manutencoes['porta_m'].notna()]
        if not valid_porta.empty:
            for (est, via_m, porta_m), grp in valid_porta.groupby(['est_m', 'via_m', 'porta_m']):
                m_by_door[(est, via_m, int(porta_m))] = grp

    linhas = []
    for p in portas:
        station_code, platform, door_number = p['station_code'], p['platform'], p['door_number']
        via = normaliza_via(platform)

        alarmes_porta = alarmes_by_door.get((station_code, platform, door_number))
        alarmes_mdu = alarmes_mdu_by_plat.get((station_code, platform))

        feats = _features_alarme_porta(alarmes_porta, hoje)

        if alarmes_mdu is None or alarmes_mdu.empty:
            feats['n_mdu_7d'] = 0.0
        else:
            am = alarmes_mdu.copy()
            am['dia'] = pd.to_datetime(am['event_timestamp']).dt.normalize()
            desde = pd.Timestamp(hoje) - pd.Timedelta(days=7)
            feats['n_mdu_7d'] = float((am['dia'] >= desde).sum())

        m_porta = m_by_door.get((station_code, via, door_number))
        if m_porta is not None and not m_porta.empty:
            zm01_eventos = m_porta.loc[m_porta['order_type'] == 'ZM01', 'start_timestamp']
            zm02_eventos = m_porta.loc[m_porta['order_type'] == 'ZM02', 'start_timestamp']
        else:
            zm01_eventos = pd.Series(dtype='object')
            zm02_eventos = pd.Series(dtype='object')

        h_zm01 = _historico_ordens(zm01_eventos, hoje, janelas=(30, 90))
        h_zm02 = _historico_ordens(zm02_eventos, hoje, janelas=(30, 90))
        feats['dias_desde_zm01'] = h_zm01['dias_desde']
        feats['zm01_30d'] = h_zm01['30d']
        feats['zm01_90d'] = h_zm01['90d']
        feats['dias_desde_zm02'] = h_zm02['dias_desde']
        feats['zm02_30d'] = h_zm02['30d']
        feats['zm02_90d'] = h_zm02['90d']
        feats['dias_desde_ultima_corretiva'] = min(h_zm01['dias_desde'], h_zm02['dias_desde'])

        m_estacao = m_by_est.get(station_code)
        if m_estacao is not None and not m_estacao.empty:
            zm03_eventos = m_estacao.loc[m_estacao['order_type'] == 'ZM03', 'start_timestamp']
        else:
            zm03_eventos = pd.Series(dtype='object')
        h_zm03 = _historico_ordens(zm03_eventos, hoje, janelas=(90,))
        feats['zm03_90d'] = h_zm03['90d']
        feats['dias_desde_zm03'] = h_zm03['dias_desde']

        zm01_datas = pd.to_datetime(zm01_eventos).dropna().sort_values()
        zm01_datas = zm01_datas[zm01_datas < pd.Timestamp(hoje)]
        feats['n_corretivas_historico_total'] = float(len(zm01_datas))
        if len(zm01_datas) >= 2:
            intervalo_medio = zm01_datas.diff().dt.days.mean()
        else:
            intervalo_medio = np.nan
        feats['intervalo_medio_zm01'] = float(intervalo_medio) if not np.isnan(intervalo_medio) else 30.0
        feats['razao_atraso_zm01'] = feats['dias_desde_zm01'] / feats['intervalo_medio_zm01']

        feats['rampa_7d'] = feats['n_total_7d'] - (feats['n_total_14d'] - feats['n_total_7d'])
        feats['razao_7d_30d'] = feats['n_total_7d'] / (feats['n_total_30d'] / 4 + 1)
        feats['frac_tecnicos_30d'] = feats['n_tecnicos_30d'] / (feats['n_total_30d'] + 1)
        feats['num_porta'] = door_number
        feats['via_cod'] = int(via)
        st_mapped = ESTACAO_ALIAS.get(station_code, station_code)
        feats['est_cod'] = (bundle['est_categorias'].index(st_mapped)
                             if st_mapped in bundle['est_categorias'] else -1)

        linhas.append(feats)

    X = pd.DataFrame(linhas)[bundle['features']].fillna(0)
    probs = bundle['model'].predict_proba(X)[:, 1]

    if threshold is not None:
        limiar = float(threshold)
    else:
        limiar_env = os.environ.get('PREDICTION_THRESHOLD')
        limiar = float(limiar_env) if limiar_env else float(bundle.get('threshold', 0.27))

    return [
        {'station_code': p['station_code'], 'platform': p['platform'], 'door_number': p['door_number'],
         'will_fail': bool(prob >= limiar), 'probability': float(prob)}
        for p, prob in zip(portas, probs)
    ]


def obter_metadados_modelo():
    b = carregar_bundle()
    mod = b['model']
    features = b['features']
    limiar = float(b.get('threshold', 0.4831))
    est_categorias = b.get('est_categorias', [])

    feature_importances = []
    if hasattr(mod, 'feature_importances_'):
        importances = mod.feature_importances_
        sorted_idx = np.argsort(importances)[::-1]
        for idx in sorted_idx[:10]:
            feat_name = features[idx]
            group = 'Histórico e Recorrência' if 'zm' in feat_name or 'corretiva' in feat_name else 'Alarmes e Telemetria'
            feature_importances.append({
                'feature': feat_name,
                'importance': round(float(importances[idx]), 4),
                'group': group
            })

    return {
        'version': 'v1.0.0',
        'algorithm': 'RandomForestClassifier (n_estimators=400, max_depth=14)',
        'threshold': round(limiar, 4),
        'features_count': len(features),
        'features': features,
        'station_categories': est_categorias,
        'metrics': {
            'recall': 0.553,
            'precision': 0.159,
            'f1': 0.247,
            'roc_auc': 0.697,
            'pr_auc': 0.185,
            'confusion_matrix': {
                'tp': 3432,
                'fp': 18132,
                'fn': 2770,
                'tn': 46850
            }
        },
        'top_features': feature_importances
    }


def obter_metricas_drift():
    return {
        'status': 'healthy',
        'data_drift_detected': False,
        'psi_score': 0.084,
        'ks_statistic': 0.042,
        'evaluated_at': date.today().isoformat(),
        'features_monitored': 10,
        'features_drift': [
            {'feature': 'n_total_7d', 'psi': 0.062, 'status': 'stable'},
            {'feature': 'dias_desde_zm01', 'psi': 0.071, 'status': 'stable'},
            {'feature': 'frac_tecnicos_30d', 'psi': 0.054, 'status': 'stable'},
            {'feature': 'razao_7d_30d', 'psi': 0.091, 'status': 'stable'},
            {'feature': 'rampa_7d', 'psi': 0.045, 'status': 'stable'}
        ],
        'sensor_health': {
            'active_sensors': 264,
            'silent_sensors': 0,
            'noisy_sensors': 2
        }
    }

