"""
Constroi as sequencias (porta, dia) usadas pela RNN.

Reaproveita o mesmo conjunto de linhas (porta, t0) e o mesmo alvo y do
Modelo 5 -- le `painel.parquet` para isso -- e para cada linha monta a
sequencia bruta dos SEQ_LEN dias anteriores a t0, em vez das features ja
agregadas em janela movel. E o mesmo dado de entrada, so que sem a
agregacao manual: quem aprende a agregar e a rede.

Uso (a partir de models/modelo-rnn/):
    python construir_sequencias.py
"""
import re
import time
from pathlib import Path

import numpy as np
import pandas as pd

# mesmos valores do construir_painel.py do Modelo 5 (nao importa o modulo
# porque ele roda o pipeline inteiro -- inclusive grava painel.parquet -- so
# de ser importado)
DATA = Path(__file__).resolve().parents[2] / 'data'
INICIO, FIM = pd.Timestamp('2023-11-01'), pd.Timestamp('2025-10-31')
PAT_TAG = re.compile(r'^PSD_(?:[A-Z]{3}|P)_([A-Z]{3})(\d)_(PDM|MDU|PSE|PFP|PFX)(\d{1,2})_?([A-Z]+)?$')

PAINEL = Path(__file__).resolve().parents[1] / 'modelo-5' / 'painel.parquet'
SAIDA = Path(__file__).resolve().parent

SEQ_LEN = 30    # dias por sequencia -- mesmo tamanho da cobertura minima exigida no Modelo 5

t0_run = time.time()


def log(msg):
    print(f'[{time.time() - t0_run:5.0f}s] {msg}', flush=True)


# ------------------------------------------------------- 1. linhas e alvo
log('lendo painel do Modelo 5 (linhas e alvo)...')
painel = pd.read_parquet(PAINEL, columns=['id_porta', 'Data', 'y'])
log(f'{len(painel):,} linhas (porta, t0), mesmas do Modelo 5')

# ------------------------------------------------------------- 2. alarmes
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

pdm = alarmes[alarmes['comp'] == 'PDM'].copy()
pdm['id_porta'] = pdm['est'] + '_' + pdm['via'] + '_' + pdm['porta'].astype(str).str.zfill(2)
log(f'alarmes de porta: {len(pdm):,}')

# --------------------------------------------------- 3. contagem diaria bruta
# Mesma tabela 'diario' do construir_painel.py (sem o rolling): um vetor por
# dia com a contagem de cada tipo de alarme. E o "passo" da sequencia.
TIPOS = sorted(pdm['Alarme'].dropna().unique())
dias_ext = pd.date_range(INICIO - pd.Timedelta(days=SEQ_LEN), FIM, freq='D')
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
diario['dur_max'] = diario['dur_max'].fillna(0)
COLS = list(diario.columns)
log(f'{len(COLS)} features por dia (mesmas contagens do Modelo 5, sem o rolling)')

portas = sorted(painel['id_porta'].unique())
idx_full = pd.MultiIndex.from_product([portas, dias_ext], names=['id_porta', 'Data'])
d = diario.reindex(idx_full, fill_value=0).sort_index()

# ---------------------------------------------------- 4. janelas deslizantes
log('montando sequencias (isso demora um pouco)...')
mat = d[COLS].to_numpy(dtype=np.float32).reshape(len(portas), len(dias_ext), len(COLS))
janelas = np.lib.stride_tricks.sliding_window_view(mat, SEQ_LEN, axis=1)  # (portas, n_dias-SEQ_LEN+1, feat, SEQ_LEN)
janelas = janelas.transpose(0, 1, 3, 2)  # -> (portas, n_dias-SEQ_LEN+1, SEQ_LEN, feat)
# janela[p, i] cobre dias_ext[i : i+SEQ_LEN]; o dia previsto t0 e dias_ext[i+SEQ_LEN],
# ou seja o dia seguinte ao ultimo da janela -- nenhum dado de t0 entra na sequencia.
porta_para_linha = {p: i for i, p in enumerate(portas)}
# dias_ext[SEQ_LEN:] == dias em [INICIO, FIM]; dia_para_col[t0] = i tal que
# janelas[:, i] cobre dias_ext[i : i+SEQ_LEN] == [t0-SEQ_LEN, t0-1]
dia_para_col = {d_: i for i, d_ in enumerate(dias_ext[SEQ_LEN:])}

linhas_ok = painel['Data'].isin(dia_para_col.keys())
log(f'{(~linhas_ok).sum():,} linhas descartadas por ficarem fora do intervalo de sequencia (esperado perto de 0)')
painel = painel[linhas_ok].reset_index(drop=True)

idx_porta = painel['id_porta'].map(porta_para_linha).to_numpy()
idx_dia = painel['Data'].map(dia_para_col).to_numpy()
X = janelas[idx_porta, idx_dia]              # (n_obs, SEQ_LEN, n_feat)
y = painel['y'].to_numpy(dtype=np.int64)

log(f'X: {X.shape} | y: {y.shape} | taxa de positivos: {y.mean() * 100:.2f}%')

np.savez_compressed(SAIDA / 'sequencias.npz', X=X, y=y)
painel[['id_porta', 'Data']].to_parquet(SAIDA / 'sequencias_index.parquet')
(SAIDA / 'sequencias_features.txt').write_text('\n'.join(COLS))
log(f'salvo em {SAIDA}/sequencias.npz ({(SAIDA / "sequencias.npz").stat().st_size / 1e6:.0f} MB)')
