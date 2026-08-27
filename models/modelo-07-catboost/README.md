# Modelo 07: CatBoostClassifier (Plataforma × Dia)

## 1. Visão Geral
Modelo preditivo baseado no algoritmo `CatBoostClassifier`, estruturado com granularidade agregada por plataforma de estação para estimar a probabilidade de falha em portas de plataforma nos próximos 7 dias.

## 2. Estrutura do Pipeline
1. `01_construcao_tabela_mestra.ipynb`: Une alarmes, ZM01, ZM02 e ZM03 com agregação por plataforma e cálculo de lags estritos `shift(1)`.
2. `02_processamento.ipynb`: Preparação das 21 features (4 categóricas nativas + 17 numéricas), análise de correlação e divisão temporal (treino, validação e teste).
3. `03_treinamento_cat.ipynb`: Treinamento do CatBoost regularizado (`l2_leaf_reg=10`, `depth=4`), cálculo de importância global e explicabilidade SHAP.

## 3. Artefatos e Resultados
- `modelo_catboost.cbm`: Modelo CatBoost serializado.
- `previsoes_teste.csv`: Previsões e probabilidades no conjunto de teste.
- **Recall no Teste:** 91,93% (1.799 de 1.957 positivos identificados com limiar ajustado).
- **PR-AUC:** 0,5892 | **ROC-AUC:** 0,5915.
