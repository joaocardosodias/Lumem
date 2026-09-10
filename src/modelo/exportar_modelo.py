import re
import time
from pathlib import Path

import joblib
import numpy as np
import pandas as pd
import shap
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import f1_score

PAINEL = Path(__file__).resolve().parents[2] / 'models' / 'modelo-5' / 'painel.parquet'
DATA = Path(__file__).resolve().parents[2] / 'data'
SAIDA = Path(__file__).resolve().parent / 'artifacts' / 'modelo_5.joblib'

t0 = time.time()


def log(msg):
    print(f'[{time.time() - t0:5.0f}s] {msg}', flush=True)


log('lendo painel...')
painel = pd.read_parquet(PAINEL)

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


log('recarregando zm01 pras features de ciclo...')
zm01 = carrega_ordens(DATA / 'Corretiva Emergencial/processados/ZM01_limpa.xlsx', 'ZM01_limpa')

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
EST_CATEGORIAS = painel['est'].cat.categories.tolist()

EXCLUI = {'id_porta', 'Data', 'est', 'via', 'id_plat', 'y', 'n_falhas_7d_futuro', 'cobertura_30d', 'mes'}
FEATS = [c for c in painel.columns if c not in EXCLUI]
log(f'{len(FEATS)} features candidatas')

FIM_TREINO, INI_TESTE, INI_VALID = (pd.Timestamp('2025-04-23'), pd.Timestamp('2025-05-01'),
                                     pd.Timestamp('2025-01-31'))
treino = painel[painel['Data'] <= FIM_TREINO]
teste = painel[painel['Data'] >= INI_TESTE]
tr_fit = treino[treino['Data'] <= INI_VALID - pd.Timedelta(days=8)]
tr_val = treino[treino['Data'] >= INI_VALID]
log(f'treino: {len(treino):,} | teste: {len(teste):,}')


def melhor_limiar(y, prob):
    grid = np.unique(np.quantile(prob, np.linspace(0.50, 0.999, 200)))
    return max(((f1_score(y, (prob >= t).astype(int), zero_division=0), t) for t in grid))[1]


def treina_avalia(feats):
    mod = RandomForestClassifier(n_estimators=400, max_depth=14, min_samples_leaf=40,
                                  class_weight='balanced', n_jobs=-1, random_state=42)
    mod.fit(tr_fit[feats], tr_fit['y'])
    lim = melhor_limiar(tr_val['y'], mod.predict_proba(tr_val[feats])[:, 1])
    mod.fit(treino[feats], treino['y'])
    return mod, lim


log('treinando com todas as features...')
mod_full, _ = treina_avalia(FEATS)

log('calculando SHAP (TreeExplainer, amostra de 4000 linhas do teste)...')
amostra = teste.sample(n=4000, random_state=42)
explainer = shap.TreeExplainer(mod_full)
sv = explainer.shap_values(amostra[FEATS])
if isinstance(sv, list):
    sv = sv[1]
if sv.ndim == 3:
    sv = sv[:, :, 1]

imp = pd.DataFrame({'feature': FEATS, 'shap_abs_medio': np.abs(sv).mean(axis=0)}) \
    .sort_values('shap_abs_medio', ascending=False).reset_index(drop=True)

limiar_corte = imp['shap_abs_medio'].iloc[0] * 0.01
mantidas = imp[imp['shap_abs_medio'] >= limiar_corte]['feature'].tolist()
log(f'mantidas: {len(mantidas)} de {len(FEATS)} (corte SHAP em {limiar_corte:.5f})')

log('treinando modelo final com as features selecionadas...')
mod_final, lim_final = treina_avalia(mantidas)
log(f'limiar final: {lim_final:.4f}')

SAIDA.parent.mkdir(parents=True, exist_ok=True)
joblib.dump({'model': mod_final, 'features': mantidas, 'threshold': float(lim_final),
             'est_categorias': EST_CATEGORIAS}, SAIDA)
log(f'bundle salvo em {SAIDA}')
