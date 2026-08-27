# Modelo 01: XGBoost & Ensembles (João)

## 1. Visão Geral
Pipeline de modelagem preditiva desenvolvido para antecipar ordens de manutenção corretiva emergencial (**ZM01**) nas 526 portas de plataforma (PSD) da Linha 4-Amarela do Metrô de São Paulo com horizonte de 7 dias.

## 2. Estrutura do Pipeline
O pipeline é executado sequencialmente através de 3 notebooks:
1. `01_construcao_tabela_mestra.ipynb`: Consolidação temporal contínua das 526 PSDs, unindo bases de alarmes, corretivas ZM01/ZM02 e preventivas ZM03 com blindagem contra vazamento de dados (*data leakage*). Saída: `data/tabela_mestra_bruta.csv`.
2. `02_preprocessamento_encoding_escalonamento.ipynb`: Engenharia de 32 features, One-Hot Encoding de estações e escalonamento `RobustScaler` restrito ao conjunto de treino. Saídas: `data/treino_pronto_modelagem_v2.csv` e `data/teste_pronto_modelagem_v2.csv`.
3. `03_treinamento_modelo_certeza_absoluta.ipynb`: Treinamento com XGBoost (histogramas), calibração de hiperparâmetros, análise de explicabilidade SHAP e avaliação multicritério sob a régua estrita de episódios de manutenção.

## 3. Principais Métricas (Régua Estrita de Episódios)
- **Perfil Proteção Máxima (p >= 0,45):** Recall de **97,94%** (evita 380 de 388 falhas reais na linha de teste).
- **Perfil Equilibrado (p >= 0,52):** F1-score de **0,1829** (Precisão 13,17%, Recall 29,90%).
- **Perfil Certeza Absoluta (p >= 0,65):** Precisão de **33,33%**.

## 4. Artefatos Gerados
Os modelos serializados e escalonadores estão salvos em `data/modelos_artefatos/`.
