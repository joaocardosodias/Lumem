"""
Etapa 4: compara o Random Forest de producao (src/modelo/artifacts/modelo_5.joblib)
com a RNN otimizada + extras (Etapa 2/3) no MESMO conjunto de teste, e testa duas
formas simples de ensemble (OR/AND dos alertas), sem re-ajustar nenhum limiar --
os dois modelos usam o limiar que cada um ja tinha escolhido na propria validacao,
entao combinar so os alertas finais nao vaza informacao do teste.

Uso (a partir de models/modelo-rnn-otimizado/):
    python avaliar_ensemble.py
"""
import json
import re
import time
from pathlib import Path

import joblib
import numpy as np
import pandas as pd

import rnn_utils as R

t0_run = time.time()


def log(msg):
    print(f'[{time.time() - t0_run:5.0f}s] {msg}', flush=True)


RAIZ = Path(__file__).resolve().parents[2]
MODELO5 = Path(__file__).resolve().parents[1] / 'modelo-5'

# --------------------------------------------------- 1. features do Random Forest
# mesmo codigo de src/modelo/exportar_modelo.py, para reproduzir exatamente as
# features que o bundle de producao espera
log('reconstruindo as features do Random Forest de producao...')
painel = pd.read_parquet(MODELO5 / 'painel.parquet')

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
painel['rampa_7d'] = painel['n_total_7d'] - (painel['n_total_14d'] - painel['n_total_7d'])
painel['razao_7d_30d'] = painel['n_total_7d'] / (painel['n_total_30d'] / 4 + 1)
painel['frac_tecnicos_30d'] = painel['n_tecnicos_30d'] / (painel['n_total_30d'] + 1)
painel['est'] = painel['est'].astype('category')
painel['est_cod'] = painel['est'].cat.codes
painel['via_cod'] = painel['via'].astype(int)

bundle = joblib.load(RAIZ / 'src/modelo/artifacts/modelo_5.joblib')
feats_rf, limiar_rf = bundle['features'], bundle['threshold']
log(f'bundle RF: {len(feats_rf)} features, limiar={limiar_rf:.4f}')

FIM_TREINO, INI_TESTE = pd.Timestamp('2025-04-23'), pd.Timestamp('2025-05-01')
teste_rf = painel[painel['Data'] >= INI_TESTE].copy()
teste_rf['prob_rf'] = bundle['model'].predict_proba(teste_rf[feats_rf])[:, 1]
teste_rf['alerta_rf'] = (teste_rf['prob_rf'] >= limiar_rf).astype(int)
log(f'RF no teste: {len(teste_rf):,} linhas | alertas: {teste_rf["alerta_rf"].sum():,}')

# ------------------------------------------------------- 2. previsoes da RNN
log('retreinando RNN otimizada + extras (seed=42, mesma config da Etapa 2/3)...')
R.set_threads(1)
with open('etapa2_resultado.json', encoding='utf-8') as f:
    etapa2 = json.load(f)

X, y, idx, mask = R.carrega_dados()
norma = R.estatisticas_norm(X[mask['fit']])
E = np.load('extras.npz')['E']
norma_extra = R.estatisticas_norm_extra(E[mask['fit']])

R.seed_tudo(42)
modelo_rnn = R.treina_fixo(X[mask['treino']], y[mask['treino']], norma, etapa2['cfg'], etapa2['melhor_epoca'],
                            Etr=E[mask['treino']], norma_extra=norma_extra)
prob_rnn = R.prediz(modelo_rnn, X[mask['teste']], norma, Ea=E[mask['teste']], norma_extra=norma_extra)

teste_rnn = idx.loc[mask['teste'], ['id_porta', 'Data']].copy()
teste_rnn['y'] = y[mask['teste']]
teste_rnn['prob_rnn'] = prob_rnn
teste_rnn['alerta_rnn'] = (prob_rnn >= etapa2['limiar']).astype(int)
log(f'RNN no teste: {len(teste_rnn):,} linhas | alertas: {teste_rnn["alerta_rnn"].sum():,}')

# ------------------------------------------------------------- 3. juntar e comparar
T = teste_rnn.merge(teste_rf[['id_porta', 'Data', 'prob_rf', 'alerta_rf']], on=['id_porta', 'Data'], how='inner')
log(f'linhas com previsao dos dois modelos: {len(T):,} (esperado == teste da RNN: {len(teste_rnn):,})')

T['alerta_or'] = ((T['alerta_rf'] == 1) | (T['alerta_rnn'] == 1)).astype(int)
T['alerta_and'] = ((T['alerta_rf'] == 1) & (T['alerta_rnn'] == 1)).astype(int)


def metricas_linha(nome, y_, pred):
    from sklearn.metrics import precision_score, recall_score, f1_score, confusion_matrix
    tn, fp, fn, tp = confusion_matrix(y_, pred).ravel()
    return {'modelo': nome, 'precisao': precision_score(y_, pred, zero_division=0),
            'recall': recall_score(y_, pred, zero_division=0), 'f1': f1_score(y_, pred, zero_division=0),
            'alertas': int(pred.sum()), 'tp': int(tp), 'fp': int(fp), 'fn': int(fn)}


resultados = [
    metricas_linha('Random Forest (producao)', T['y'], T['alerta_rf']),
    metricas_linha('RNN otimizada + extras', T['y'], T['alerta_rnn']),
    metricas_linha('Ensemble OR (RF ou RNN alerta)', T['y'], T['alerta_or']),
    metricas_linha('Ensemble AND (RF e RNN alertam)', T['y'], T['alerta_and']),
]
RES = pd.DataFrame(resultados)
print(RES.round(3).to_string(index=False))
RES.to_csv('ensemble_resultado.csv', index=False)
T.to_parquet('ensemble_previsoes_teste.parquet')
log('salvo ensemble_resultado.csv e ensemble_previsoes_teste.parquet')
