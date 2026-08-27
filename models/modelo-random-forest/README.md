# Modelo Random Forest: Predição por Eventos e Ciclos de Manutenção

## 1. Visão Geral
Modelo preditivo baseado em Random Forest que incorpora uma modelagem explícita dos ciclos de recorrência de falha e avalia o impacto operacional sob a métrica de episódios reais de manutenção.

## 2. Estrutura do Diretório
- `construir_painel.py`: Constrói o painel analítico `painel.parquet` integrando as 4 fontes de dados por estação, via e porta.
- `painel.parquet`: Painel pré-processado otimizado para carregamento rápido.
- `modelo_preditivo_falhas.ipynb`: Notebook com seleção de features por SHAP, treinamento do Random Forest e avaliação por episódio real.

## 3. Principais Resultados
- **Captura por Evento Real:** 65,0% das falhas reais (766 de 1.178) antecipadas com 26,0% de precisão por episódio de alerta.
- **Ganho com Features de Ciclo:** Aumento do F1 por evento de 0,325 para 0,372 com as variáveis de atraso no ciclo de quebra (`razao_atraso_zm01` e `dias_desde_ultima_corretiva`).
