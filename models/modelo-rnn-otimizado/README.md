# Otimização do Modelo RNN

Testa a hipótese de que a RNN (`models/modelo-rnn/modelo_rnn.ipynb`) performou abaixo do Random Forest (Modelo 5) por nunca ter passado por otimização nem tuning de hiperparâmetros. Documentação completa em [`otimizacao-rnn.mdx`](../../docs/src/content/docs/sprints/Sprint%204/otimizacao-rnn.mdx).

## Etapas

| Etapa | Notebook | O que muda |
|---|---|---|
| 0 — Baseline | `modelo_rnn_otimizado_0_baseline.ipynb` | nada; reproduz o resultado original de forma reutilizável (`rnn_utils.py`) |
| 1 — Tuning | `modelo_rnn_otimizado_1_busca_hiperparametros.ipynb` | busca de hiperparâmetros com Optuna, mesmas 12 features |
| 2 — Mais informação | `modelo_rnn_otimizado_2_ablacao_extras.ipynb` | mesma arquitetura da Etapa 1, + histórico ZM01/02/03 e estação/via |
| 3 — Avaliação final | `modelo_rnn_otimizado_3_avaliacao_final.ipynb` | os três modelos anteriores, por evento/episódio (não por linha) |
| 4 — Fechamento | `avaliar_ensemble.py`, `exportar_modelo.py` | comparação com o Random Forest de produção, ensemble simples e artefato final |

## Resultado final (teste, por evento/episódio)

| Modelo | Recall por evento | Precisão por episódio | F1 evento |
|---|---|---|---|
| RNN baseline (sem tuning) | 46,0% | 29,4% | 0,359 |
| RNN otimizada, só tuning | 65,8% | 27,1% | 0,384 |
| **RNN otimizada + extras** | **69,1%** | **29,8%** | **0,416** |
| Random Forest (Modelo 5) | 65,0% | 26,0% | 0,372 |

A hipótese se confirmou parcialmente: o tuning (Etapa 1) foi o maior ganho isolado; dar à RNN o mesmo histórico de manutenção que o Random Forest já usava (Etapa 2) foi o que faltava para ela superar o Random Forest na métrica operacional.

## Arquivos

- `rnn_utils.py`: treino/avaliação compartilhados entre as etapas (aceita sequência sozinha ou sequência + features extra).
- `construir_extras.py` → `extras.npz`, `extras_features.txt`: as 23 features extra da Etapa 2 (histórico ZM, ciclo/recorrência, estação/via), alinhadas com `../modelo-rnn/sequencias_index.parquet`.
- `optuna_search.py`, `optuna_study.db`, `optuna_trials.csv`, `optuna_best.json`: busca de hiperparâmetros da Etapa 1 (retomável — o `study.db` guarda os trials já rodados).
- `avaliar_ensemble.py` → `ensemble_resultado.csv`, `ensemble_previsoes_teste.parquet`: compara o Random Forest de produção (`src/modelo/artifacts/modelo_5.joblib`) com a RNN no mesmo teste, e testa ensemble OR/AND dos alertas.
- `exportar_modelo.py` → `artefatos/rnn_otimizada_extras.pt`: salva o modelo final (pesos, config, normalização, limiar), caso a equipe decida integrar a RNN à API depois. **Não altera nada em produção.**
- `baseline.json`, `etapa1_resultado.json`, `etapa2_resultado.json`, `etapa3_resultado.json`: resumo numérico de cada etapa, usado pelas etapas seguintes para comparação.

## O que não foi feito

A RNN otimizada + extras não foi colocada em produção. A API (`src/modelo/`) calcula, ao vivo, as 83 features agregadas que o Random Forest usa; a RNN precisaria da sequência bruta de 30 dias por porta mais as 23 features extra desta etapa — um pipeline de features diferente, que não foi construído. Essa é uma decisão de arquitetura separada, ver `README.md` de `src/modelo/`.
