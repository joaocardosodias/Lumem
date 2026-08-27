# Modelo 5: Random Forest (Evento & Ciclos de Manutenção)

## 1. Visão Geral
Modelo preditivo baseado em Random Forest que modela os ciclos de recorrência pós-reparo e avalia o impacto operacional sob a métrica de episódios reais de manutenção.

## 2. Arquivos
- `construir_painel.py`: Constrói o painel analítico `painel.parquet` integrando as 4 fontes de dados por estação, via e porta.
- `painel.parquet`: Painel pré-processado.
- `modelo_preditivo_falhas.ipynb`: Notebook com seleção de features por SHAP, treinamento do Random Forest e avaliação por episódio real.

## 3. Resultados
- **Captura por Evento Real:** 65,0% das falhas reais (766 de 1.178) antecipadas com 26,0% de precisão por episódio.
- **F1 por Evento:** 0,372 com as features de ciclo de manutenção.
