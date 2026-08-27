# Hub de Modelos Preditivos — Sprint 2 (Linha 4-Amarela)

Bem-vindo ao diretório central de **Modelos Preditivos de Inteligência Artificial** desenvolvidos pelo Grupo 04 para antecipar falhas e ordens de manutenção corretiva emergencial (**ZM01**) nas 526 Portas de Plataforma (**PSD**) da Linha 4-Amarela do Metrô de São Paulo.

---

## 1. Estrutura dos Modelos no Repositório

Todos os modelos desenvolvidos pela equipe foram padronizados em subpastas dedicadas contendo seus respectivos notebooks, scripts de pré-processamento, tabelas intermediárias e artefatos serializados:

| Diretório | Modelo & Algoritmo | Unidade / Granularidade | Alvo / Janela | Destaque Técnico |
| :--- | :--- | :--- | :--- | :--- |
| [`modelo-1/`](./modelo-1/) | **Modelo 1 (XGBoost & Ensembles)** | Porta × Dia (526 PSDs) | Falha em 7 dias (ZM01) | 32 features, `scale_pos_weight=9.5`, regularização L1/L2, 97,94% de recall estrito e explicabilidade SHAP. |
| [`modelo-2/`](./modelo-2/) | **Modelo 2 (LightGBM & Random Forest)** | Porta × Dia (526 PSDs) | Falha em 7 dias (ZM01/ZM02) | 23 features, análise cruzada Gini vs Permutação no teste, ROC-AUC 0,634. |
| [`modelo-3/`](./modelo-3/) | **Modelo 3 (Árvore de Decisão)** | Porta individual × Dia | Falha em 7 dias (ZM01) | Metodologia CRISP-DM completa (1 a 5), alta interpretabilidade e regras de decisão explícitas. |
| [`modelo-4/`](./modelo-4/) | **Modelo 4 (CatBoostClassifier)** | Plataforma × Dia | Falha em 7 dias (ZM01) | 21 features (4 categóricas nativas), regularização L2 severa, recall de 91,93% no teste. |
| [`modelo-5/`](./modelo-5/) | **Modelo 5 (Random Forest Evento & Ciclos)** | Porta × Dia | Falha em 7 dias (ZM01) | Painel unificado, features de ciclo/recorrência pós-manutenção, métricas por episódio real (F1: 0,372). |

---

## 2. Quadro Comparativo de Métricas

A tabela abaixo resume o desempenho dos modelos nas principais réguas de engenharia de manutenção:

| Modelo | Algoritmo Principal | Features | Recall | Precisão | Métrica Complementar | Perfil Operacional |
| :--- | :--- | :---: | :---: | :---: | :---: | :--- |
| **Modelo 1** | XGBoost Classifier | 32 | **97,94%** (estrito) | 9,35% a 33,33% | F1: 0,1829 (p=0,52) | Proteção máxima contra quebras na linha |
| **Modelo 2** | LightGBM Tunado | 23 | **60,00%** | 22,00% | ROC-AUC: 0,634 | Equilíbrio e interpretabilidade de features |
| **Modelo 3** | Decision Tree | ~15 | Validação Temporal | Decisão Rápida | PR-AUC / F1 | Baseline de regras simples e auditoria |
| **Modelo 4** | CatBoostClassifier | 21 | **91,93%** | 54,63% (pontual) | PR-AUC: 0,5892 | Agregação por plataforma com variáveis categóricas |
| **Modelo 5** | Random Forest | 40+ | **65,00%** (por evento) | 26,00% (por episódio) | F1 evento: 0,372 | Sensibilidade a ciclos de fadiga pós-manutenção |

---

## 3. Reprodutibilidade e Execução

### Pré-requisitos
Certifique-se de que o ambiente virtual está ativo e as dependências instaladas:
```bash
pip install -r requirements.txt
```

### Ordem de Execução por Modelo:
1. **Modelo 1:**
   - Executar `modelo-1/01_modelo_1_construcao_tabela_mestra.ipynb`
   - Executar `modelo-1/02_modelo_1_preprocessamento.ipynb`
   - Executar `modelo-1/03_modelo_1_treinamento.ipynb`
2. **Modelo 2:**
   - Executar `python modelo-2/limpeza_consolidacao.py`
   - Executar `python modelo-2/construir_dataset_modelo.py`
   - Executar `python modelo-2/preparar_features_modelo.py`
   - Executar `modelo-2/modelo_2.ipynb`
3. **Modelo 3:**
   - Executar `modelo-3/modelo_3.ipynb`
4. **Modelo 4:**
   - Executar `modelo-4/01_modelo_4_construcao_tabela_mestra.ipynb`
   - Executar `modelo-4/02_modelo_4_processamento.ipynb`
   - Executar `modelo-4/03_modelo_4_treinamento.ipynb`
5. **Modelo 5:**
   - Executar `python modelo-5/construir_painel.py`
   - Executar `modelo-5/modelo_5.ipynb`

---

## 4. Documentação Completa
A documentação detalhada de cada modelo, com gráficos de explicabilidade SHAP, matrizes de confusão e curvas ROC/PR, pode ser acessada na documentação web em:
- [Documentação Sprint 2](../../docs/src/content/docs/sprints/Sprint%202/index.mdx)
