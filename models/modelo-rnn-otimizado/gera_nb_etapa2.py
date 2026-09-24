import nbformat as nbf

nb = nbf.v4.new_notebook()
md = lambda s: nb.cells.append(nbf.v4.new_markdown_cell(s.strip()))
code = lambda s: nb.cells.append(nbf.v4.new_code_cell(s.strip()))

md("""
# Otimização da RNN — Etapa 2: ablação (histórico ZM01/02/03 e estação/via)

Etapa 0 (baseline, sem tuning) e Etapa 1 (busca de hiperparâmetros) melhoraram a RNN de PR-AUC 0,154 para 0,172, mas ela continuou abaixo do Random Forest do Modelo 5 (0,185). A explicação levantada nas duas etapas anteriores foi a mesma: a RNN só recebe 12 contadores de alarme por dia, sem o histórico de ordens de manutenção (ZM01/02/03) nem a identificação de estação/via — informação que o Random Forest usa e que, segundo o próprio `modelo_5.ipynb`, foi a principal responsável pelo ganho de recall dele (seção 9: "as features de ciclo foram... as principais responsáveis pela melhora de recall").

Esta etapa testa essa explicação: adiciona essas features como entrada extra da RNN, **sem mudar mais nada** — mesma arquitetura e mesmos hiperparâmetros vencedores da Etapa 1 (GRU, 2 camadas, 32 unidades). Assim, qualquer diferença de resultado vem só da informação nova, e dá pra separar o que veio do tuning (Etapa 1) do que vem de dar mais dado à rede (esta etapa).
""")

md("""
## 1. Que features foram adicionadas

Essas features são conhecidas em `t0` (um valor por observação, não uma série de 30 dias), então não fazem sentido como mais um canal da sequência diária. Elas entram concatenadas ao estado escondido final da RNN, antes da camada de saída (ver `RNNFalha` em `rnn_utils.py`) — a sequência de alarmes continua sendo processada do mesmo jeito.

Construídas em [`construir_extras.py`](construir_extras.py), a partir do mesmo `painel.parquet` do Modelo 5:

| Grupo | Features | Origem |
|---|---|---|
| Janela ZM | `dias_desde_zm01`, `zm01_30d`, `zm01_90d`, `dias_desde_zm02`, `zm02_30d`, `zm02_90d`, `dias_desde_zm03`, `zm03_90d` | já existiam em `painel.parquet` |
| Ciclo/recorrência | `dias_desde_ultima_corretiva`, `n_corretivas_historico_total`, `intervalo_medio_zm01`, `razao_atraso_zm01` | recalculadas com o mesmo código da seção 2 do `modelo_5.ipynb` (não estavam salvas no parquet) |
| Identificação | estação (one-hot, 10 colunas) e via (inteiro) | `painel['est']`/`painel['via']` |

23 features no total, uma por observação (id_porta, Data) — o mesmo grão do `sequencias_index.parquet` usado pela RNN.
""")

code("""
import json
import time

import numpy as np
import pandas as pd
import torch

import rnn_utils as R

R.set_threads(1)
""")

md("""
## 2. Dados: sequências + features extra

As sequências (`X`, `y`) e a partição temporal são as mesmas das Etapas 0 e 1. `E` é a matriz de features extra, alinhada linha a linha com `X` (mesma ordem de `sequencias_index.parquet`).
""")

code("""
X, y, idx, m = R.carrega_dados()

extras = np.load('extras.npz')
E = extras['E']
cols_extra = open('extras_features.txt').read().splitlines()
print(f'X: {X.shape} | E: {E.shape} ({len(cols_extra)} features extra)')
assert len(E) == len(X), 'E e X precisam ter o mesmo numero de linhas, na mesma ordem'

norma = R.estatisticas_norm(X[m['fit']])
norma_extra = R.estatisticas_norm_extra(E[m['fit']])
""")

md("""
## 3. Configuração: a mesma vencedora da Etapa 1

Nenhum hiperparâmetro muda aqui — só o `n_extra` (calculado automaticamente pelo tamanho de `E`) passa a existir na rede.
""")

code("""
with open('etapa1_resultado.json', encoding='utf-8') as f:
    etapa1 = json.load(f)

COLS_METRICA = ['modelo', 'precisao', 'recall', 'f1', 'pr_auc', 'roc_auc', 'alertas', 'tp', 'fp', 'fn']

def so_metricas(d):
    return {c: d[c] for c in COLS_METRICA}

cfg = dict(etapa1['cfg'])
print(json.dumps(cfg, indent=2))
""")

md("""
## 4. Retreino com early stopping (fit/val originais)
""")

code("""
R.seed_tudo(42)
t0 = time.time()
melhor_epoca, melhor_prauc, historico = R.treina(
    X[m['fit']], y[m['fit']], X[m['val']], y[m['val']], norma, cfg,
    Etr=E[m['fit']], Eval=E[m['val']], norma_extra=norma_extra,
)
print(f"\\nmelhor epoca: {melhor_epoca} | PR-AUC val: {melhor_prauc:.4f} | {time.time()-t0:.0f}s")
""")

md("""
## 5. Treino final e avaliação no teste

Mesma lógica das etapas anteriores: o limiar de decisão é escolhido no `val` (maximizando F1) com o modelo treinado só no `fit`; depois o modelo é retreinado do zero no treino completo pelas épocas escolhidas e avaliado uma única vez no teste.
""")

code("""
modelo_fit = R.treina_fixo(X[m['fit']], y[m['fit']], norma, cfg, melhor_epoca,
                            Etr=E[m['fit']], norma_extra=norma_extra)
prob_val = R.prediz(modelo_fit, X[m['val']], norma, Ea=E[m['val']], norma_extra=norma_extra)
limiar = R.melhor_limiar(y[m['val']], prob_val)
print(f'limiar escolhido na validacao: {limiar:.4f}')

modelo_final = R.treina_fixo(X[m['treino']], y[m['treino']], norma, cfg, melhor_epoca,
                              Etr=E[m['treino']], norma_extra=norma_extra)
prob_teste = R.prediz(modelo_final, X[m['teste']], norma, Ea=E[m['teste']], norma_extra=norma_extra)
resultado_extras = R.avalia('RNN otimizada + extras (uma semente)', y[m['teste']],
                             (prob_teste >= limiar).astype(int), prob_teste)

with open('baseline.json', encoding='utf-8') as f:
    baseline = json.load(f)
rf = baseline['referencia_random_forest']

pd.DataFrame([so_metricas(baseline['teste']), so_metricas(etapa1['teste_uma_semente']), resultado_extras, rf]).round(3).to_string(index=False)
""")

md("""
## 6. Estabilidade: múltiplas sementes

Mesmo procedimento da Etapa 1: o retreino final (treino completo → teste) é repetido com 5 sementes diferentes, mantendo a configuração e o número de épocas.
""")

code("""
sementes = [42, 7, 123, 2024, 99]
resultados_sementes = []
for s in sementes:
    R.seed_tudo(s)
    mf = R.treina_fixo(X[m['treino']], y[m['treino']], norma, cfg, melhor_epoca,
                        Etr=E[m['treino']], norma_extra=norma_extra)
    pt = R.prediz(mf, X[m['teste']], norma, Ea=E[m['teste']], norma_extra=norma_extra)
    r = R.avalia(f'seed={s}', y[m['teste']], (pt >= limiar).astype(int), pt)
    resultados_sementes.append(r)
    print(f"seed={s:5d} | PR-AUC={r['pr_auc']:.4f} | recall={r['recall']:.3f} | f1={r['f1']:.3f}")

RS = pd.DataFrame(resultados_sementes)
print(f"\\nPR-AUC: media={RS['pr_auc'].mean():.4f} | desvio={RS['pr_auc'].std():.4f}")
print(f"recall: media={RS['recall'].mean():.3f} | desvio={RS['recall'].std():.3f}")
""")

md("""
## 7. Comparação final

Quatro pontos na mesma tabela: o baseline sem tuning (Etapa 0), a RNN só com tuning (Etapa 1), a RNN com tuning + as features extra (esta etapa) e o Random Forest. A diferença entre a segunda e a terceira linha isola o efeito de dar mais informação à rede, já que a arquitetura e os hiperparâmetros são os mesmos nas duas.
""")

code("""
media_extras = {'modelo': 'RNN otimizada + extras (media de 5 sementes)',
                 **{c: RS[c].mean() for c in ['precisao', 'recall', 'f1', 'pr_auc', 'roc_auc', 'alertas', 'tp', 'fp', 'fn']}}

comparacao = pd.DataFrame([
    so_metricas(baseline['teste']),
    so_metricas(etapa1['teste_media_5_sementes']),
    media_extras,
    rf,
])
comparacao.round(3).to_string(index=False)
""")

md("""
## 8. Síntese

*(preenchido depois de rodar, com os números reais)*

**Próximo passo (Etapa 3):** avaliação final por evento/episódio (não só por linha porta-dia), aplicada ao melhor candidato encontrado até aqui, no mesmo formato da seção 8 do `modelo_5.ipynb`.
""")

nbf.write(nb, 'modelo_rnn_otimizado_2_ablacao_extras.ipynb')
print('notebook gerado')
