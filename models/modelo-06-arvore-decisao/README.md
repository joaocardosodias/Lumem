# Modelo 06: Árvore de Decisão para Predição de Falhas em PSDs

## 1. Visão Geral
Implementação do modelo de Árvore de Decisão (Scikit-Learn) estruturado rigorosamente nas 5 primeiras fases da metodologia **CRISP-DM** (Entendimento do Negócio, Entendimento dos Dados, Preparação dos Dados, Modelagem e Avaliação).

## 2. Estrutura do Diretório
- `Modelo_6_decision_tree.ipynb`: Notebook executável contendo todo o ciclo de vida do modelo, desde o carregamento de alarmes e ordens ZM01 até a validação temporal e teste com análise de capacidade de generalização e regras de decisão explícitas.

## 3. Destaques
- Identificação individualizada por PSD (Estação × Via × Porta).
- Filtro de alarmes ativos e janelas temporais móveis (1d, 7d e 30d).
- Alta interpretabilidade e transparência nas regras de classificação.
