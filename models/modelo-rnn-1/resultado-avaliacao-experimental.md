# LSTM experimental — avaliação final

A etapa 6 foi concluída com os pesos da melhor época, sem novo treinamento. O limiar foi escolhido pelo maior F1 na validação, com desempate pelo maior limiar. O teste passou pelo modelo uma única vez e suas previsões foram preservadas.

Limiar de alerta: **0.08316838**. A regra utiliza probabilidade maior ou igual ao valor exato salvo em `limiar_alerta.json`.

| Conjunto | Janelas | Positivos | Alertas | Precisão | Recall | F1 | AP | Referência AP |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| validacao | 42359 | 2008 | 3731 | 11.53% | 21.41% | 0.1499 | 0.0973 | 0.0474 |
| teste | 41239 | 1711 | 1987 | 5.54% | 6.43% | 0.0595 | 0.0513 | 0.0415 |

O teste identificou 110 das 1.711 janelas positivas e produziu 1.877 falsos alertas. O desempenho ficou abaixo da validação e ainda é baixo para orientar alertas de manutenção.

Matriz de confusão do teste:

| Rótulo | Sem alerta | Com alerta |
| --- | ---: | ---: |
| 0 provisório | 37651 | 1877 |
| 1 | 1601 | 110 |

Os números representam janelas por porta/dia: uma ocorrência pode produzir várias janelas positivas. O rótulo 0 indica ausência de novo início registrado sob as regras experimentais; não comprova ausência de defeitos. As métricas dependem dessa hipótese e não validam uso operacional.

O F1 foi adotado como critério inicial, sem custos operacionais definidos. O resultado do teste não foi utilizado para ajustar o limiar ou o modelo.

## Arquivos salvos

Os arquivos estão em [artefatos/lstm-experimental](artefatos/lstm-experimental/):

- `pesos_lstm.pt` e `arquitetura_lstm.py`: pesos e definição da rede.
- `preprocessadores.joblib`: transformações ajustadas no treino.
- `configuracao.json` e `limiar_alerta.json`: entradas, parâmetros e decisão de alerta.
- `metricas_avaliacao.csv`, `matriz_confusao_teste.csv` e `resultado_avaliacao.json`: resultados agregados.
- `predicoes_validacao.csv` e `predicoes_teste.csv`: previsões por janela, mantidas localmente.
- `registro_avaliacao.json`: data e assinaturas dos artefatos avaliados.
- `avaliacao_final.png`: curva da validação e matriz de confusão do teste.

Em nova sessão, a preparação pode ser executada até a seção 11.2 do notebook, seguida das seções 12.1 a 12.4, sem repetir as seções de treinamento 11.3 a 11.5. As previsões de teste existentes são reutilizadas quando os artefatos e as entradas coincidem.
