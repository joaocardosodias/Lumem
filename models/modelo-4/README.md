# Modelo 4: CatBoostClassifier (Plataforma × Dia)

## 1. Visão Geral
Modelo preditivo baseado no algoritmo `CatBoostClassifier` com agregação por plataforma de estação para estimar probabilidade de falha em 7 dias.

## 2. Pipeline e Artefatos
1. `01_construcao_tabela_mestra.ipynb`: Agregação temporal por plataforma e lags estritos.
2. `02_processamento.ipynb`: Preparação das 21 features (4 categóricas nativas + 17 numéricas) e split temporal.
3. `03_treinamento_cat.ipynb`: Treinamento do CatBoost regularizado (`l2_leaf_reg=10`, `depth=4`) e análise SHAP.
- `modelo_catboost.cbm`: Modelo serializado.
- `previsoes_teste.csv`: Previsões do conjunto de teste.

## 3. Resultados
- **Recall no Teste:** 91,93% (1.799 de 1.957 positivos identificados).
- **PR-AUC:** 0,5892 | **ROC-AUC:** 0,5915.
