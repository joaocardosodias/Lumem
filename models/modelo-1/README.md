# Modelo 1: XGBoost & Ensembles (João)

## 1. Visão Geral
Pipeline de modelagem preditiva para antecipar ordens de manutenção corretiva emergencial (**ZM01**) nas 526 PSDs da Linha 4-Amarela com horizonte de 7 dias.

## 2. Notebooks do Pipeline
1. `01_modelo_1_construcao_tabela_mestra.ipynb`: Consolidação contínua das 526 PSDs unindo alarmes, ZM01/ZM02 e ZM03. Saída: `data/tabela_mestra_bruta.csv`.
2. `02_modelo_1_preprocessamento.ipynb`: Engenharia de 32 features, One-Hot Encoding e `RobustScaler` no treino. Saídas: `data/treino_pronto_modelagem_v2.csv` e `data/teste_pronto_modelagem_v2.csv`.
3. `03_modelo_1_treinamento.ipynb`: Treinamento XGBoost (histogramas), calibração de hiperparâmetros, explicabilidade SHAP e avaliação sob a régua estrita de manutenção.

## 3. Principais Métricas
- **Proteção Máxima (p >= 0,45):** Recall de **97,94%** (380 / 388 falhas reais evitadas).
- **Equilibrado (p >= 0,52):** F1 de **0,1829** (Precisão 13,17%, Recall 29,90%).
- **Certeza Absoluta (p >= 0,65):** Precisão de **33,33%**.
