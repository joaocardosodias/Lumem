import re
from datetime import date, timedelta

import joblib
import numpy as np
import pandas as pd
from scipy.signal import lfilter

from database import buscar_manutencoes, buscar_telemetria_recente

TIPOS = ['CA', 'CSD', 'FC', 'FO', 'LSD', 'OA', 'OBS', 'RCE']
JANELAS = (1, 3, 7, 14, 30)
PAT_LOC = re.compile(r'^SP04-6-PSD-([A-Z]{3})(?:-PLT(\d)|-P(\d)S\d)?(?:-PP(\d{2})(PDM|PFX|PSE))?')

_bundle = None


def carregar_bundle(caminho='artifacts/modelo_5.joblib'):
    global _bundle
    if _bundle is None:
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


def _features_alarme_porta(alarmes_porta: pd.DataFrame, hoje: date) -> dict:
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


def processar_e_prever(portas: list[dict]) -> list[dict]:
    bundle = carregar_bundle()
    hoje = date.today()

    estacoes = sorted({p['station_code'] for p in portas})
    alarmes = buscar_telemetria_recente(estacoes, hoje, janela_dias=60)
    manutencoes = buscar_manutencoes(estacoes)

    if not manutencoes.empty:
        loc = manutencoes['installation_location'].apply(extrai_local)
        manutencoes = manutencoes.assign(
            est_m=[x[0] for x in loc], via_m=[x[1] for x in loc], porta_m=[x[2] for x in loc],
        )

    linhas = []
    for p in portas:
        station_code, platform, door_number = p['station_code'], p['platform'], p['door_number']
        via = normaliza_via(platform)

        alarmes_porta = alarmes[
            (alarmes['station_code'] == station_code) &
            (alarmes['platform'] == platform) &
            (alarmes['door_number'] == door_number)
        ] if not alarmes.empty else alarmes

        alarmes_mdu = alarmes[
            (alarmes['station_code'] == station_code) &
            (alarmes['platform'] == platform) &
            (alarmes['door_number'].isna())
        ] if not alarmes.empty else alarmes

        feats = _features_alarme_porta(alarmes_porta, hoje)

        if alarmes_mdu.empty:
            feats['n_mdu_7d'] = 0.0
        else:
            am = alarmes_mdu.copy()
            am['dia'] = pd.to_datetime(am['event_timestamp']).dt.normalize()
            desde = pd.Timestamp(hoje) - pd.Timedelta(days=7)
            feats['n_mdu_7d'] = float((am['dia'] >= desde).sum())

        if manutencoes.empty:
            m_porta = manutencoes
        else:
            m_porta = manutencoes[
                (manutencoes['est_m'] == station_code) &
                (manutencoes['via_m'] == via) &
                (manutencoes['porta_m'] == door_number)
            ]
        zm01_eventos = m_porta.loc[m_porta['order_type'] == 'ZM01', 'start_timestamp'] if not m_porta.empty else pd.Series(dtype='object')
        zm02_eventos = m_porta.loc[m_porta['order_type'] == 'ZM02', 'start_timestamp'] if not m_porta.empty else pd.Series(dtype='object')

        h_zm01 = _historico_ordens(zm01_eventos, hoje, janelas=(30, 90))
        h_zm02 = _historico_ordens(zm02_eventos, hoje, janelas=(30, 90))
        feats['dias_desde_zm01'] = h_zm01['dias_desde']
        feats['zm01_30d'] = h_zm01['30d']
        feats['zm01_90d'] = h_zm01['90d']
        feats['dias_desde_zm02'] = h_zm02['dias_desde']
        feats['zm02_30d'] = h_zm02['30d']
        feats['zm02_90d'] = h_zm02['90d']
        feats['dias_desde_ultima_corretiva'] = min(h_zm01['dias_desde'], h_zm02['dias_desde'])

        if manutencoes.empty:
            m_estacao = manutencoes
        else:
            m_estacao = manutencoes[manutencoes['est_m'] == station_code]
        zm03_eventos = m_estacao.loc[m_estacao['order_type'] == 'ZM03', 'start_timestamp'] if not m_estacao.empty else pd.Series(dtype='object')
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
        feats['est_cod'] = (bundle['est_categorias'].index(station_code)
                             if station_code in bundle['est_categorias'] else -1)

        linhas.append(feats)

    X = pd.DataFrame(linhas)[bundle['features']].fillna(0)
    probs = bundle['model'].predict_proba(X)[:, 1]

    return [
        {'station_code': p['station_code'], 'platform': p['platform'], 'door_number': p['door_number'],
         'will_fail': bool(prob >= bundle['threshold']), 'probability': float(prob)}
        for p, prob in zip(portas, probs)
    ]
