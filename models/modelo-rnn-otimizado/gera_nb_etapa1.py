import json
import nbformat as nbf

nb = nbf.v4.new_notebook()
md = lambda s: nb.cells.append(nbf.v4.new_markdown_cell(s.strip()))
code = lambda s: nb.cells.append(nbf.v4.new_code_cell(s.strip()))

md("""
# Otimização da RNN (LSTM/GRU) — Etapa 1: busca de hiperparâmetros

A Etapa 0 reproduziu o baseline (LSTM de 1 camada, 32 unidades, sem busca): PR-AUC de 0,154 no teste, contra 0,185 do Random Forest do Modelo 5.

Esta etapa faz o que a Etapa 0 apontou como faltante: uma busca sistemática de hiperparâmetros, do mesmo jeito que o Modelo 5 usou `RandomizedSearchCV`/`GridSearchCV`. As 12 features atuais e a janela de 30 dias continuam as mesmas (decisão tomada no planejamento); o que muda aqui é só a arquitetura e o treino.
""")

md("""
## 1. Como a busca foi feita

A busca usou [Optuna](optuna_search.py) com o amostrador TPE, otimizando o PR-AUC médio em **2 folds temporais expansivos dentro do treino** (mesma lógica do `TimeSeriesSplit` do Modelo 5, ver `rnn_utils.folds_temporais`). O teste não entrou na busca em nenhum momento.

Espaço de busca:

| Hiperparâmetro | Valores testados |
|---|---|
| Célula | LSTM ou GRU |
| Unidades escondidas | 16, 32, 64 |
| Camadas | 1 ou 2 |
| Dropout | 0 a 0,4 |
| Learning rate | 1e-4 a 5e-3 (log-uniforme) |
| Batch | 512 ou 1024 |
| Weight decay | 0, 1e-5, 1e-4, 1e-3 |
| Multiplicador do `pos_weight` | 0,5 a 2,0 |

Um benchmark prévio mostrou que o custo por época varia muito com o tamanho da rede: de ~3,5s (16 unidades, 1 camada) a ~40s (64 unidades, 2 camadas), em CPU com 1 thread por trial — não há GPU nesta máquina. Por isso a busca usou um orçamento reduzido de épocas (máximo 8, paciência 3) para caber no tempo disponível; a configuração vencedora é retreinada com o orçamento completo (paciência 3, até 15 épocas) nas seções seguintes, com o `fit`/`val` originais.

Os trials rodaram em paralelo (threads do Optuna), já que o PyTorch libera o GIL durante as operações pesadas.
""")

code("""
import json
import time

import pandas as pd
import torch
import numpy as np

import rnn_utils as R

R.set_threads(1)
""")

code("""
with open('optuna_best.json', encoding='utf-8') as f:
    busca = json.load(f)
print(json.dumps(busca, indent=2, ensure_ascii=False))
""")

md("""
## 2. Trials da busca
""")

code("""
trials = pd.read_csv('optuna_trials.csv')
cols = ['number', 'value', 'state', 'params_cell', 'params_hidden', 'params_layers',
        'params_dropout', 'params_lr', 'params_batch', 'params_weight_decay', 'params_pos_weight_mult']
trials_ok = trials[trials['state'] == 'COMPLETE'].sort_values('value', ascending=False)
print(f"{len(trials)} trials no total | {len(trials_ok)} completos | {(trials['state']=='PRUNED').sum()} podados")
trials_ok[cols].head(10).round(4).to_string(index=False)
""")

code("""
import matplotlib.pyplot as plt

plt.figure(figsize=(7, 4))
ordem = trials.sort_values('number')
completos = ordem[ordem['state'] == 'COMPLETE']
plt.scatter(completos['number'], completos['value'], label='completo')
podados = ordem[ordem['state'] == 'PRUNED']
if len(podados):
    # trials podados nao tem 'value' final -- marcados no eixo x, sem posicao no y
    plt.scatter(podados['number'], [0] * len(podados),
                marker='x', color='gray', alpha=0.5, label='podado (nao chegou a terminar)')
plt.axhline(busca['best_value_pr_auc_cv'], color='green', linestyle='--', label='melhor PR-AUC (CV)')
plt.xlabel('trial'); plt.ylabel('PR-AUC medio (2 folds)'); plt.legend(); plt.title('Progresso da busca Optuna')
plt.tight_layout(); plt.savefig('progresso_busca.png', dpi=110)
plt.show()
""")

md("""
## 3. Configuração vencedora

Estes são os hiperparâmetros escolhidos pela busca (maior PR-AUC médio nos 2 folds de validação temporal). Nas próximas seções, essa configuração é retreinada com o orçamento completo (early stopping com paciência 3 e até 15 épocas), primeiro no `fit`/`val` originais para escolher a época, depois no treino completo para a avaliação final no teste — a mesma lógica da Etapa 0 e do Modelo 5.
""")

code("""
cfg = dict(busca['best_params'])
cfg['max_epocas'], cfg['paciencia'] = 15, 3
print(json.dumps(cfg, indent=2))
""")

md("""
## 4. Retreino com early stopping (fit/val originais)
""")

code("""
X, y, idx, m = R.carrega_dados()
norma = R.estatisticas_norm(X[m['fit']])

R.seed_tudo(42)
t0 = time.time()
melhor_epoca, melhor_prauc, historico = R.treina(X[m['fit']], y[m['fit']], X[m['val']], y[m['val']], norma, cfg)
print(f"\\nmelhor epoca: {melhor_epoca} | PR-AUC val: {melhor_prauc:.4f} | {time.time()-t0:.0f}s")
""")

md("""
## 5. Treino final e avaliação no teste

Mesma lógica da Etapa 0 e do Modelo 5: o modelo do `fit` decide o limiar de decisão no `val` (maximizando F1); depois o modelo é retreinado do zero no treino completo pelas épocas escolhidas e avaliado **uma única vez** no teste.
""")

code("""
modelo_fit = R.treina_fixo(X[m['fit']], y[m['fit']], norma, cfg, melhor_epoca)
limiar = R.melhor_limiar(y[m['val']], R.prediz(modelo_fit, X[m['val']], norma))
print(f'limiar escolhido na validacao: {limiar:.4f}')

modelo_final = R.treina_fixo(X[m['treino']], y[m['treino']], norma, cfg, melhor_epoca)
prob_teste = R.prediz(modelo_final, X[m['teste']], norma)
resultado_otim = R.avalia('RNN otimizada (Optuna)', y[m['teste']], (prob_teste >= limiar).astype(int), prob_teste)

with open('baseline.json', encoding='utf-8') as f:
    baseline = json.load(f)

rf = baseline['referencia_random_forest']
R_ = pd.DataFrame([baseline['teste'], resultado_otim, rf])
R_.round(3).to_string(index=False)
""")

md("""
## 6. Estabilidade: múltiplas sementes

Um único treino pode ter dado sorte ou azar com a inicialização dos pesos. Para não escolher a configuração vencedora com base em ruído, o retreino final (treino completo → teste) é repetido com outras sementes, mantendo a mesma configuração e o mesmo número de épocas.
""")

code("""
sementes = [42, 7, 123, 2024, 99]
resultados_sementes = []
for s in sementes:
    R.seed_tudo(s)
    mf = R.treina_fixo(X[m['treino']], y[m['treino']], norma, cfg, melhor_epoca)
    pt = R.prediz(mf, X[m['teste']], norma)
    r = R.avalia(f'seed={s}', y[m['teste']], (pt >= limiar).astype(int), pt)
    resultados_sementes.append(r)
    print(f"seed={s:5d} | PR-AUC={r['pr_auc']:.4f} | recall={r['recall']:.3f} | f1={r['f1']:.3f}")

RS = pd.DataFrame(resultados_sementes)
print(f"\\nPR-AUC: media={RS['pr_auc'].mean():.4f} | desvio={RS['pr_auc'].std():.4f}")
print(f"recall: media={RS['recall'].mean():.3f} | desvio={RS['recall'].std():.3f}")
""")

md("""
## 7. Comparação final
""")

code("""
comparacao = pd.DataFrame([
    baseline['teste'],
    resultado_otim,
    {**{'modelo': 'RNN otimizada (media de 5 sementes)'},
     **{c: RS[c].mean() for c in ['precisao', 'recall', 'f1', 'pr_auc', 'roc_auc', 'alertas', 'tp', 'fp', 'fn']}},
    rf,
])
comparacao.round(3).to_string(index=False)
""")

md("""
## 8. Síntese e próximo passo

*(preenchido depois de rodar, com os números reais)*

**Etapa 2:** ablação — adicionar histórico ZM01/02/03 e estação/via como entradas da RNN, para separar o que veio do tuning do que viria de dar mais informação à rede.
""")

nbf.write(nb, 'modelo_rnn_otimizado_1_busca_hiperparametros.ipynb')
print('notebook gerado')
