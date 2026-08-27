"""
Constroi o painel (porta, dia) usado pelo modelo preditivo de falha em 7 dias.

Une as quatro bases tratadas na Sprint 1 pela chave estacao + plataforma + porta,
calcula as features de janela movel e o rotulo, e grava `painel.parquet`.

Uso (a partir de eda/modelo/):
    python construir_painel.py
"""
import re
import time
from pathlib import Path

import numpy as np
import pandas as pd

DATA = Path(__file__).resolve().parents[2] / 'data'
SAIDA = Path(__file__).resolve().parent / 'painel.parquet'

INICIO, FIM = pd.Timestamp('2023-11-01'), pd.Timestamp('2025-10-31')
HORIZONTE = 7          # dias da janela de previsao
COBERTURA = 30         # dias de telemetria exigidos na estacao

# A TAG do alarme aparece em tres formatos:
#   PSD_BUT_BUT2_PDM09_OBS   |   PSD_P_BUT1_MDU16_STS   |   PSD_P_MBI1_PDM13LSD
PAT_TAG = re.compile(r'^PSD_(?:[A-Z]{3}|P)_([A-Z]{3})(\d)_(PDM|MDU|PSE|PFP|PFX)(\d{1,2})_?([A-Z]+)?$')
PAT_LOC = re.compile(r'^SP04-6-PSD-([A-Z]{3})(?:-PLT(\d)|-P(\d)S\d)?(?:-PP(\d{2})(PDM|PFX|PSE))?')

t0 = time.time()


def log(msg):
    print(f'[{time.time() - t0:5.0f}s] {msg}', flush=True)


# ---------------------------------------------------------------- 1. alarmes
log('lendo alarmes...')
alarmes = pd.concat([
    pd.read_excel(DATA / 'Alarmes/Alarmes do sistema de PSD.xlsx'),
    pd.read_excel(DATA / 'Alarmes/Histórico de alarmes 23-26.xlsx'),
], ignore_index=True).drop_duplicates()

tag = alarmes['TAG'].str.extract(PAT_TAG)
alarmes['est'], alarmes['via'], alarmes['comp'], alarmes['porta'] = tag[0], tag[1], tag[2], tag[3]
alarmes['DT'] = pd.to_datetime(alarmes['Horário do Alarme'], errors='coerce')
alarmes['DTN'] = pd.to_datetime(alarmes['Horário da Normalizacao'], errors='coerce')
alarmes = alarmes.dropna(subset=['est', 'DT']).copy()
alarmes['porta'] = alarmes['porta'].astype(int)
alarmes['dur_s'] = (alarmes['DTN'] - alarmes['DT']).dt.total_seconds()
alarmes['Data'] = alarmes['DT'].dt.normalize()
hora = alarmes['DT'].dt.hour
alarmes['pico'] = hora.between(6, 8) | hora.between(16, 18)

pdm = alarmes[alarmes['comp'] == 'PDM'].copy()
pdm['id_porta'] = pdm['est'] + '_' + pdm['via'] + '_' + pdm['porta'].astype(str).str.zfill(2)
mdu = alarmes[alarmes['comp'] == 'MDU'].copy()
mdu['id_plat'] = mdu['est'] + '_' + mdu['via']
log(f'alarmes de porta: {len(pdm):,} | do modulo da plataforma: {len(mdu):,}')


# ------------------------------------------------------------- 2. ordens SAP
def carrega_ordens(caminho, aba):
    """Le uma base de ordens tratada e extrai estacao/plataforma/porta."""
    df = pd.read_excel(caminho, aba)
    m = df['local_de_instalacao'].apply(lambda s: PAT_LOC.match(s) if isinstance(s, str) else None)
    df['est'] = [x.group(1) if x else None for x in m]
    df['via'] = [(x.group(2) or x.group(3)) if x else None for x in m]
    df['porta'] = [int(x.group(4)) if x and x.group(4) else None for x in m]
    df = df.dropna(subset=['est', 'via', 'porta', 'inicio_avaria_datetime']).copy()
    df['porta'] = df['porta'].astype(int)
    df['id_porta'] = df['est'] + '_' + df['via'] + '_' + df['porta'].astype(str).str.zfill(2)
    df['Data'] = pd.to_datetime(df['inicio_avaria_datetime']).dt.normalize()
    return df


zm01 = carrega_ordens(DATA / 'Corretiva Emergencial/processados/ZM01_limpa.xlsx', 'ZM01_limpa')
zm02 = carrega_ordens(DATA / 'Corretiva Programada/processados/ZM02_limpa.xlsx', 'ZM02_limpa')
zm03 = pd.read_csv(DATA / 'Preventiva/preventiva_zm03_limpa.csv', parse_dates=['DT_Inicio'])
zm03 = zm03[zm03['Estacao'] != 'GERAL'].dropna(subset=['DT_Inicio']).copy()
zm03['Data'] = zm03['DT_Inicio'].dt.normalize()
log(f'ZM01 nivel porta: {len(zm01):,} | ZM02: {len(zm02):,} | ZM03 com estacao: {len(zm03):,}')

# -------------------------------------------------------- 3. grade porta-dia
portas = sorted(set(pdm['id_porta']) | set(zm01['id_porta']))
dias = pd.date_range(INICIO, FIM, freq='D')
dias_ext = pd.date_range(INICIO - pd.Timedelta(days=COBERTURA), FIM, freq='D')
grade = pd.MultiIndex.from_product([portas, dias], names=['id_porta', 'Data']).to_frame(index=False)
grade['est'] = grade['id_porta'].str[:3]
grade['via'] = grade['id_porta'].str[4]
grade['num_porta'] = grade['id_porta'].str[-2:].astype(int)
log(f'grade: {len(grade):,} linhas ({len(portas)} portas x {len(dias)} dias)')

# ------------------------------------------- 4. contagens diarias e janelas
TIPOS = sorted(pdm['Alarme'].dropna().unique())
pa = pdm[pdm['Data'].between(dias_ext[0], FIM)]

diario = pa.pivot_table(index=['id_porta', 'Data'], columns='Alarme', aggfunc='size', fill_value=0)
diario = diario.reindex(columns=TIPOS, fill_value=0)
diario.columns = ['n_' + c for c in diario.columns]
diario = diario.join(pa.groupby(['id_porta', 'Data']).agg(
    n_total=('Alarme', 'size'),
    n_long30=('dur_s', lambda s: (s > 30).sum()),
    dur_max=('dur_s', 'max'),
))
diario['n_tecnicos'] = diario['n_total'] - diario.get('n_OBS', 0)

log('montando janelas moveis...')
cols_cnt = [c for c in diario.columns if c.startswith('n_')]
idx_full = pd.MultiIndex.from_product([portas, dias_ext], names=['id_porta', 'Data'])
d = diario.reindex(idx_full, fill_value=0).sort_index()
d['dur_max'] = d['dur_max'].fillna(0)

g = d.groupby(level='id_porta')
feats = {}
for w in (1, 3, 7, 14, 30):
    r = g[cols_cnt].rolling(w, min_periods=1).sum().reset_index(level=0, drop=True)
    for c in cols_cnt:
        feats[f'{c}_{w}d'] = r[c]
feats['dur_max_7d'] = g['dur_max'].rolling(7, min_periods=1).max().reset_index(level=0, drop=True)
feats['dur_max_30d'] = g['dur_max'].rolling(30, min_periods=1).max().reset_index(level=0, drop=True)

# --------------------------------------- 4.1 peso por recencia (decaimento)
# Em vez de so somar dentro de uma janela fixa, um alarme de ontem deve pesar
# mais que um de 29 dias atras. y_t = x_t + d*y_(t-1), com d = 0.5**(1/meia_vida)
# -- um filtro IIR de 1a ordem (scipy.signal.lfilter), causal por construcao.
from scipy.signal import lfilter

log('calculando decaimento por recencia...')
for col, meias_vidas in [('n_total', (3, 7, 14)), ('n_tecnicos', (7,)), ('n_LSD', (7,))]:
    serie = d[col].unstack('id_porta').sort_index()  # linhas=dias, colunas=portas
    for h in meias_vidas:
        decay = 0.5 ** (1 / h)
        y = lfilter([1.0], [1.0, -decay], serie.values, axis=0)
        # stack() da index (Data, id_porta); d.index e (id_porta, Data) -- precisa inverter
        feats[f'{col}_decaido_h{h}'] = pd.DataFrame(y, index=serie.index, columns=serie.columns) \
            .stack().swaplevel(0, 1).reindex(d.index)

# defasagem de 1 dia: nenhuma feature enxerga o proprio dia t nem o futuro
F = pd.DataFrame(feats).groupby(level='id_porta').shift(1)
F = F.loc[(slice(None), slice(INICIO, FIM)), :]
painel = grade.set_index(['id_porta', 'Data']).join(F).reset_index()

# modulo da plataforma (MDU), mesma logica
plats = sorted(set(mdu['id_plat']))
dm = (mdu[mdu['Data'].between(dias_ext[0], FIM)].groupby(['id_plat', 'Data']).size()
      .reindex(pd.MultiIndex.from_product([plats, dias_ext], names=['id_plat', 'Data']), fill_value=0)
      .sort_index())
mdu_7 = (dm.groupby(level='id_plat').rolling(7, min_periods=1).sum()
         .reset_index(level=0, drop=True).groupby(level='id_plat').shift(1).rename('n_mdu_7d'))
painel['id_plat'] = painel['est'] + '_' + painel['via']
painel = painel.merge(mdu_7.reset_index(), on=['id_plat', 'Data'], how='left')

# --------------------------------------------------- 5. historico das ordens
log('historico SAP...')
painel = painel.sort_values(['id_porta', 'Data']).reset_index(drop=True)


def historico(painel, ordens, chave, prefixo, janelas=(30, 90)):
    """Dias desde a ultima ordem (estritamente antes de t) e contagens em janelas."""
    out = pd.DataFrame(index=painel.index)
    ev = ordens.groupby([chave, 'Data']).size().rename('n').reset_index().sort_values([chave, 'Data'])

    ult = ev[[chave, 'Data']].copy()
    ult['ultima'] = ult['Data']
    p = painel[[chave, 'Data']].copy()
    p['_idx'] = painel.index          # merge_asof descarta o indice; ele viaja como coluna
    m = pd.merge_asof(p.sort_values('Data'), ult.sort_values('Data'), on='Data', by=chave,
                      direction='backward', allow_exact_matches=False)
    out[f'dias_desde_{prefixo}'] = pd.Series((m['Data'] - m['ultima']).dt.days.values,
                                             index=m['_idx'].values).reindex(painel.index)

    piv = ev.pivot_table(index=chave, columns='Data', values='n', fill_value=0)
    piv = piv.reindex(columns=dias_ext, fill_value=0)
    alvo = painel.set_index([chave, 'Data']).index
    for w in janelas:
        s = piv.T.rolling(w, min_periods=1).sum().T.shift(1, axis=1).stack().rename(f'{prefixo}_{w}d')
        s.index.names = [chave, 'Data']
        out[f'{prefixo}_{w}d'] = alvo.map(s).astype(float)
    return out


painel = pd.concat([painel,
                    historico(painel, zm01, 'id_porta', 'zm01'),
                    historico(painel, zm02, 'id_porta', 'zm02'),
                    historico(painel, zm03.rename(columns={'Estacao': 'est'})[['est', 'Data']],
                              'est', 'zm03', janelas=(90,))], axis=1)

# ------------------------------------- 6. rotulo: houve ZM01 em (t, t+7]?
log('rotulo...')
piv_f = (zm01.groupby(['id_porta', 'Data']).size().rename('n').reset_index()
         .pivot_table(index='id_porta', columns='Data', values='n', fill_value=0)
         .reindex(index=portas,
                  columns=pd.date_range(INICIO, FIM + pd.Timedelta(days=HORIZONTE), freq='D'),
                  fill_value=0))
fut = piv_f.T[::-1].rolling(HORIZONTE, min_periods=1).sum()[::-1].T.shift(-1, axis=1)
s_fut = fut.stack().rename('n_falhas_7d_futuro')
s_fut.index.names = ['id_porta', 'Data']
painel['n_falhas_7d_futuro'] = painel.set_index(['id_porta', 'Data']).index.map(s_fut).astype(float)
painel['y'] = (painel['n_falhas_7d_futuro'] > 0).astype(int)

# ------------------------------------------------ 7. filtro de cobertura
cob = (alarmes.groupby(['est', 'Data']).size().rename('n').reset_index()
       .pivot_table(index='est', columns='Data', values='n', fill_value=0)
       .reindex(columns=dias_ext, fill_value=0))
s_cob = cob.T.rolling(COBERTURA, min_periods=1).sum().T.shift(1, axis=1).stack().rename('cob')
s_cob.index.names = ['est', 'Data']
painel['cobertura_30d'] = painel.set_index(['est', 'Data']).index.map(s_cob).astype(float)

antes = len(painel)
painel = painel[painel['cobertura_30d'] > 0].copy()
log(f'linhas sem telemetria na estacao removidas: {antes - len(painel):,} '
    f'({(antes - len(painel)) / antes * 100:.1f}%)')

# ------------------------------------------------------------ 8. calendario
painel['dia_semana'] = painel['Data'].dt.dayofweek
painel['fim_de_semana'] = (painel['dia_semana'] >= 5).astype(int)
painel['mes'] = painel['Data'].dt.month

painel = painel.fillna({c: 0 for c in painel.columns if c.startswith(('n_', 'zm0', 'dur_'))})
for c in ('dias_desde_zm01', 'dias_desde_zm02', 'dias_desde_zm03'):
    painel[c] = painel[c].fillna(9999)   # porta/estacao sem nenhuma ordem anterior

painel.to_parquet(SAIDA)
log(f'painel salvo em {SAIDA}: {painel.shape[0]:,} linhas x {painel.shape[1]} colunas')
print(f"periodo .......... {painel['Data'].min():%d/%m/%Y} a {painel['Data'].max():%d/%m/%Y}")
print(f"portas ........... {painel['id_porta'].nunique()} em {painel['est'].nunique()} estacoes")
print(f"taxa de falha .... {painel['y'].mean() * 100:.2f}%")
