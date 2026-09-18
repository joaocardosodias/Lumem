# LSTM experimental — etapa 5

A arquitetura solicitada foi treinada com os registros disponíveis de ZM01 e ZM02. O rótulo 0 é provisório: significa ausência de novo início de falha ou ocorrência pendente no horizonte de sete dias, sob as regras documentadas na seção 11.1 do notebook. Ele não confirma ausência de defeitos não registrados.

## Configuração e seleção

- Entrada temporal: 30 dias × 12 features; entrada estática: 13 colunas codificadas.
- LSTM(64), LayerNormalization, Dropout(0,3), concatenação estática, Dense(32, ReLU), Dropout(0,3), Dense(1, Sigmoid).
- Adam, taxa 0,001, entropia cruzada binária, batch de 256, semente 42, CPU.
- Limite de 30 épocas; parada após cinco épocas sem melhora da perda de validação.
- Execução concluída após **6 épocas**; pesos da **época 1** restaurados.
- Nenhum hiperparâmetro ou limiar foi escolhido com resultados do teste.

## Conjuntos utilizados

| Conjunto | Amostras | Positivos | Negativos provisórios |
| --- | ---: | ---: | ---: |
| Treino | 206.648 | 10.384 | 196.264 |
| Validação | 42.359 | 2.008 | 40.351 |

Os cortes cronológicos foram preservados. Os números representam janelas por porta/dia, não quantidades de falhas físicas distintas.

## Validação na melhor época

| Métrica | Valor |
| --- | ---: |
| Perda binária | 0,184914 |
| Average Precision (AP) | 0,097335 |
| Proporção de positivos, referência para AP | 0,047404 |
| Precisão no limiar 0,5 | 1,000000 |
| Recall no limiar 0,5 | 0,000996 |
| F1 no limiar 0,5 | 0,001990 |

No limiar 0,5, foram emitidos somente **2 alertas**, ambos positivos entre 2.008 janelas positivas. A precisão de 100% se refere a esses dois alertas e não representa boa cobertura das falhas. O recall é de aproximadamente **0,10%**.

A AP supera a proporção de positivos usada como referência, mas a identificação de janelas positivas no limiar inicial é muito baixa. A próxima etapa deverá analisar o limiar na validação e depois avaliar uma única vez o teste reservado. A avaliação de teste não foi realizada nesta etapa.

## Artefatos

- [Pesos da melhor época](artefatos/lstm-experimental/pesos_lstm.pt)
- [Configuração e hipóteses](artefatos/lstm-experimental/configuracao.json)
- [Pré-processadores](artefatos/lstm-experimental/preprocessadores.joblib)
- [Histórico por época](artefatos/lstm-experimental/historico_treinamento.csv)
- [Previsões de validação](artefatos/lstm-experimental/predicoes_validacao.csv)
- [Curvas de validação](artefatos/lstm-experimental/curvas_validacao.png)

A recarga dos pesos foi conferida comparando suas saídas com as do modelo em memória. Os rótulos e as sequências conservadoras foram preservados separadamente. As planilhas processadas não foram alteradas.


A etapa 6 foi concluída posteriormente e está documentada no [relatório da avaliação final](resultado-avaliacao-experimental.md).
