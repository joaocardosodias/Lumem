# Modelo 02: Classificação de Risco de Falha com LightGBM e Random Forest

## 1. Visão Geral
Modelo de classificação binária para previsão de risco de falha em Portas de Plataforma (PSD) nos próximos 7 dias utilizando histórico de alarmes, manutenções anteriores e fluxo de passageiros (Catracas).

## 2. Estrutura do Diretório
- `limpeza_consolidacao.py`: Script de limpeza e consolidação das fontes brutas em `data/`.
- `construir_dataset_modelo.py`: Constrói a base agregada por porta e dia.
- `preparar_features_modelo.py`: Gera as 23 features de histórico, normalização e passageiros.
- `modelo_2.ipynb`: Notebook com treinamento do Random Forest baseline, análise de importância Gini vs. Permutação, tuning bayesiano e avaliação do LightGBM final.
- `dataset_modelo_final.csv`: Dataset processado de 296.472 registros.

## 3. Principais Resultados
- **ROC-AUC (teste):** 0,634
- **Recall (falha em 7d):** 0,60
- **Precisão (falha em 7d):** 0,22
- **Features Dominantes:** `dias_desde_ultima_corretiva`, `n_corretivas_90d` e `tempo_medio_normalizacao_7d`.
