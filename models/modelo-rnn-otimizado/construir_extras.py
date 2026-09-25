"""
Etapa 2: monta as features "extra" (nao sequenciais) para a ablacao da RNN --
historico de manutencao (ZM01/02/03) e identificacao (estacao/via).

Sao features conhecidas em t0 (uma por observacao, nao uma serie de 30 dias),
entao nao fazem sentido como mais um canal da sequencia; entram concatenadas
ao estado escondido final da RNN (ver `rnn_utils.RNNFalha`). A ideia e a
mesma da secao 2 e 9 do modelo_5.ipynb: alem das janelas moveis de alarme,
o Random Forest usa o historico de ordens de manutencao e a identificacao da
porta, e isso e o que faltava na RNN nas Etapas 0 e 1.

Fonte dos dados: `painel.parquet` (Modelo 5) ja tem as features de ZM01/02/03
em janela (dias_desde_zm01, zm01_30d, zm01_90d, ...) e a identificacao
(est, via). As 4 features de "ciclo" (dias_desde_ultima_corretiva,
n_corretivas_historico_total, intervalo_medio_zm01, razao_atraso_zm01) sao
recalculadas aqui com o mesmo codigo da secao 2 do modelo_5.ipynb, porque
elas nao foram salvas no painel.parquet (foram calculadas direto no
notebook).

Uso (a partir de models/modelo-rnn-otimizado/):
    python construir_extras.py

Grava:
    extras.npz            -- matriz E [n_obs, n_extra], alinhada com
                              ../modelo-rnn/sequencias_index.parquet
    extras_features.txt   -- nome de cada coluna de E
"""
import re
import time
from pathlib import Path

import numpy as np
import pandas as pd

t0_run = time.time()


def log(msg):
    print(f'[{time.time() - t0_run:5.0f}s] {msg}', flush=True)


RAIZ = Path(__file__).resolve().parents[2]
MODELO5 = Path(__file__).resolve().parents[1] / 'modelo-5'
SEQ_DIR = Path(__file__).resolve().parents[1] / 'modelo-rnn'
SAIDA = Path(__file__).resolve().parent

# --------------------------------------------------------- 1. painel base
log('lendo painel do Modelo 5...')
painel = pd.read_parquet(MODELO5 / 'painel.parquet')

# ----------------------------------- 2. features de ciclo (secao 2 do Modelo 5)
# mesmo codigo do modelo_5.ipynb: reconstroi o historico causal de ZM01 por porta
PAT_LOC = re.compile(r'^SP04-6-PSD-([A-Z]{3})(?:-PLT(\d)|-P(\d)S\d)?(?:-PP(\d{2})(PDM|PFX|PSE))?')


def carrega_ordens(caminho, aba):
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


log('recalculando features de ciclo (mesmo codigo da secao 2 do modelo_5.ipynb)...')
zm01 = carrega_ordens(RAIZ / 'data/Corretiva Emergencial/processados/ZM01_limpa.xlsx', 'ZM01_limpa')

ev = zm01[['id_porta', 'Data']].dropna().drop_duplicates().sort_values(['id_porta', 'Data']).reset_index(drop=True)
ev['ordem'] = ev.groupby('id_porta').cumcount() + 1
ev['diff'] = ev.groupby('id_porta')['Data'].diff().dt.days
ev['intervalo_medio_ate_aqui'] = ev.groupby('id_porta')['diff'] \
    .apply(lambda s: s.expanding().mean()).reset_index(level=0, drop=True)

p = painel[['id_porta', 'Data']].copy()
p['_idx'] = painel.index
m = pd.merge_asof(p.sort_values('Data'),
                   ev[['id_porta', 'Data', 'ordem', 'intervalo_medio_ate_aqui']].sort_values('Data'),
                   on='Data', by='id_porta', direction='backward', allow_exact_matches=False)
painel['n_corretivas_historico_total'] = pd.Series(m['ordem'].values, index=m['_idx'].values) \
    .reindex(painel.index).fillna(0)
_interv = pd.Series(m['intervalo_medio_ate_aqui'].values, index=m['_idx'].values).reindex(painel.index)
painel['intervalo_medio_zm01'] = _interv.fillna(_interv.median())
painel['razao_atraso_zm01'] = painel['dias_desde_zm01'] / painel['intervalo_medio_zm01']
painel['dias_desde_ultima_corretiva'] = painel[['dias_desde_zm01', 'dias_desde_zm02']].min(axis=1)

# -------------------------------------------------- 3. selecao das colunas extra
ZM_JANELA = ['dias_desde_zm01', 'zm01_30d', 'zm01_90d', 'dias_desde_zm02', 'zm02_30d', 'zm02_90d',
             'dias_desde_zm03', 'zm03_90d']
CICLO = ['dias_desde_ultima_corretiva', 'n_corretivas_historico_total',
         'intervalo_medio_zm01', 'razao_atraso_zm01']

extra = painel[['id_porta', 'Data']].copy()
extra[ZM_JANELA + CICLO] = painel[ZM_JANELA + CICLO]

# identificacao: estacao em one-hot, via como inteiro (mesma logica de est_cod/via_cod do Modelo 5,
# mas one-hot em vez de codigo categorico arbitrario -- mais estavel para uma rede neural)
estacao_oh = pd.get_dummies(painel['est'], prefix='estacao', dtype=np.float32)
extra = pd.concat([extra, estacao_oh], axis=1)
extra['via'] = painel['via'].astype(int)

COLS_EXTRA = ZM_JANELA + CICLO + list(estacao_oh.columns) + ['via']
log(f'{len(COLS_EXTRA)} features extra: {len(ZM_JANELA)} de janela ZM + {len(CICLO)} de ciclo + '
    f'{len(estacao_oh.columns)} de estacao (one-hot) + 1 de via')

# -------------------------------------------- 4. alinhar com o indice das sequencias
log('alinhando com sequencias_index.parquet (mesma ordem de X/y da RNN)...')
seq_idx = pd.read_parquet(SEQ_DIR / 'sequencias_index.parquet')
alinhado = seq_idx.merge(extra, on=['id_porta', 'Data'], how='left', validate='one_to_one')
faltantes = alinhado[COLS_EXTRA].isna().any(axis=1).sum()
if faltantes:
    log(f'AVISO: {faltantes} linhas sem match no painel -- preenchidas com 0')
    alinhado[COLS_EXTRA] = alinhado[COLS_EXTRA].fillna(0)

E = alinhado[COLS_EXTRA].to_numpy(dtype=np.float32)
log(f'E: {E.shape}')

np.savez_compressed(SAIDA / 'extras.npz', E=E)
(SAIDA / 'extras_features.txt').write_text('\n'.join(COLS_EXTRA))
log(f'salvo em {SAIDA}/extras.npz')
