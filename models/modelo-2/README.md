# Modelo 2: Classificação de Risco com LightGBM e Random Forest

## 1. Visão Geral
Classificação binária de risco de falha em 7 dias (`falha_7d`) utilizando histórico de alarmes, manutenções anteriores e fluxo de passageiros (Catracas).

## 2. Estrutura de Arquivos
- `limpeza_consolidacao.py`: Script de limpeza e consolidação das fontes brutas em `data/`.
- `construir_dataset_modelo.py`: Constrói a base agregada porta × dia.
- `preparar_features_modelo.py`: Gera as 23 features de histórico e passageiros.
- `modelo_2.ipynb`: Notebook com baseline Random Forest, importância Gini vs. Permutação, tuning bayesiano e avaliação do LightGBM final.
- `dataset_modelo_final.csv`: Dataset processado de 296.472 registros.

## 3. Principais Resultados
- **ROC-AUC (teste):** 0,634
- **Recall (falha em 7d):** 0,60 | **Precisão:** 0,22
