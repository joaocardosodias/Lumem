# Documentação do Primeiro Modelo de Machine Learning
## Predição de falha emergencial em portas de plataforma (PSD) nos próximos 7 dias

---

## 1. Objetivo do documento

Este documento registra, de forma detalhada e rastreável, as decisões tomadas durante a construção do primeiro modelo de Machine Learning do projeto, seguindo as etapas 1 a 5 do CRISP-DM:

1. Entendimento do negócio;
2. Entendimento dos dados;
3. Preparação dos dados;
4. Modelagem;
5. Avaliação.

O objetivo do modelo é responder à seguinte pergunta:

> **“Essa porta apresentará uma falha real nos próximos 7 dias?”**

O problema foi formulado como uma tarefa de **classificação binária**, em que cada observação representa uma porta física em uma determinada data.

A variável alvo utilizada é:

```text
falha_emergencial_7d
```

onde:

- `1` significa que a porta terá uma falha real registrada em uma ordem corretiva emergencial ZM01 nos próximos 7 dias;
- `0` significa que não haverá uma falha real nesse horizonte.

É importante destacar que o modelo **não prevê se haverá um alarme nos próximos 7 dias**. Os alarmes são utilizados como sinais históricos de entrada para tentar antecipar uma falha real futura.

---

# 2. CRISP-DM — Etapa 1: Entendimento do negócio

## 2.1. Problema de negócio

O sistema de portas de plataforma (PSD) pode apresentar falhas que afetam disponibilidade, operação e confiabilidade.

O objetivo do projeto é criar um modelo capaz de identificar, com antecedência de até 7 dias, portas com maior probabilidade de apresentar uma falha real.

A intenção é transformar dados históricos de operação e manutenção em um sinal de risco antecipado.

A unidade principal de interesse é a **porta física**.

## 2.2. Questão preditiva

A pergunta modelada foi definida como:

> **“Essa porta apresentará uma falha real nos próximos 7 dias?”**

O horizonte de 7 dias foi adotado como janela de previsão.

A previsão é interpretada como realizada no **início de cada dia**.

Exemplo:

```text
Data da observação: 10/05/2025 00:00

Features:
informações conhecidas até 09/05/2025 23:59:59

Target:
existência de falha entre 10/05/2025 e antes de 17/05/2025
```

Essa definição foi essencial para prevenir vazamento de informação futura.

## 2.3. Métricas escolhidas

As métricas utilizadas para avaliação foram:

- Recall;
- Precision;
- Accuracy;
- F1-score;
- Matriz de confusão.

A métrica considerada **prioritária** foi o **Recall da classe positiva**.

### Justificativa

Um falso negativo significa:

> a porta realmente falhou, mas o modelo não gerou sinal de risco.

Esse tipo de erro é particularmente relevante para o problema, pois representa uma falha futura que não foi antecipada.

A Accuracy foi mantida como métrica complementar, mas não como métrica principal, devido ao forte desbalanceamento da variável alvo.

---

# 3. CRISP-DM — Etapa 2: Entendimento dos dados

## 3.1. Fontes utilizadas

As principais fontes de dados analisadas foram:

### Alarmes PSD

Contêm registros de alarmes do sistema de portas, incluindo:

- TAG;
- horário do alarme;
- estação;
- porta;
- tipo de alarme;
- horário de normalização;
- tempo de normalização.

### ZM01 — Corretivas emergenciais

Utilizada principalmente para:

- identificação de falhas reais;
- construção do target;
- histórico de falhas anteriores.

### ZM02 — Corretivas programadas

Utilizada em uma etapa posterior para investigar novas features relacionadas ao histórico de manutenção programada.

### ZM03 — Preventivas

Foi avaliada durante o entendimento dos dados, porém não entrou na versão final do primeiro modelo.

### Fluxo de passageiros

Foi estruturado durante a preparação dos dados, porém também não entrou na versão final do primeiro modelo.

## 3.2. Granularidade das fontes

As fontes apresentam granularidades diferentes.

### Alarmes

Foi possível identificar porta física para praticamente todos os registros relevantes.

### ZM01

Os registros podem existir em três níveis:

- estação;
- plataforma;
- porta.

Somente eventos explicitamente associados a uma porta foram utilizados para construção do target.

### ZM02

Também contém registros em diferentes níveis, mas foi possível identificar eventos em nível de porta.

### ZM03

A base preventiva não possuía granularidade de porta suficiente para ser utilizada diretamente como histórico da porta.

Essa diferença de granularidade foi importante para evitar atribuir uma manutenção de estação ou plataforma a uma porta específica de forma artificial.

---

# 4. CRISP-DM — Etapa 3: Preparação dos dados

## 4.1. Padronização dos nomes das colunas

Foi adotada uma convenção de nomes para indicar claramente a origem de cada variável.

Exemplos:

```text
causa_emergencial
defeito_emergencial
inicio_avaria_datetime_emergencial

causa_programada
inicio_avaria_datetime_programada

inicio_preventiva
duracao_h_preventiva
```

As chaves comuns de integração foram mantidas sem sufixo:

```text
data
estacao
plataforma
porta
chave_estacao
chave_plataforma
chave_porta
```

## 4.2. Extração da localização física

Foram criadas chaves hierárquicas para integração das bases.

### Chave de estação

```text
chave_estacao = estacao
```

### Chave de plataforma

Formato:

```text
ESTACAO_PPLATAFORMA
```

Exemplo:

```text
BUT_P1
```

### Chave de porta

Formato:

```text
ESTACAO_PPLATAFORMA_DPORTA
```

Exemplo:

```text
BUT_P1_D05
```

Essas chaves permitiram integrar alarmes e manutenções respeitando a granularidade física.

## 4.3. Tratamento das TAGs de alarme

As TAGs de alarmes possuíam diferentes padrões estruturais.

Foram utilizados padrões de extração capazes de identificar:

- estação;
- plataforma;
- porta.

A cobertura de localização obtida foi de aproximadamente 99,98%.

Os poucos registros sem porta explícita foram preservados, mas não utilizados nas análises de nível de porta.

Nenhuma porta foi inferida quando ela não estava explicitamente disponível na TAG.

## 4.4. Identificação de duplicidades nos alarmes

Durante a análise inicial das contagens de alarmes, foram observados casos extremos.

Exemplo:

```text
FRA_P1_D03
2024-05-29
5.668 registros
```

Uma investigação detalhada mostrou que grande parte desses registros não representava eventos independentes.

Foi identificado que:

- múltiplas linhas possuíam o mesmo TAG;
- o mesmo horário de disparo aparecia repetidamente;
- os horários de normalização podiam variar;
- em casos extremos, centenas de registros correspondiam ao mesmo disparo.

Foi então adotada a definição:

> **1 evento único de alarme = 1 combinação única de `tag_alarme` + `horario_alarme`.**

A base de eventos passou a ser criada com:

```python
alarmes_eventos = (
    alarmes
    .sort_values("horario_alarme")
    .drop_duplicates(
        subset=[
            "tag_alarme",
            "horario_alarme"
        ],
        keep="first"
    )
    .copy()
)
```

Essa decisão reduziu significativamente a inflação artificial das contagens.

## 4.5. Universo de portas

O universo de portas foi construído a partir das portas físicas identificadas na base de alarmes.

Foram identificadas:

```text
518 portas distintas
```

Distribuição por estação:

| Estação | Portas |
|---|---:|
| BUT | 48 |
| FAL | 48 |
| FRA | 48 |
| FRE | 38 |
| HIG | 48 |
| LUZ | 48 |
| MBI | 48 |
| PIN | 48 |
| PTA | 48 |
| REP | 48 |
| VSO | 48 |

## 4.6. Construção da base diária

A unidade de observação escolhida foi:

> **1 porta × 1 dia**

Foi criado um produto cartesiano entre:

- 518 portas;
- todas as datas do período de modelagem.

O período principal foi:

```text
27/02/2023 a 31/12/2025
```

A base inicial possuía aproximadamente:

```text
538.202 linhas
```

Cada linha representa o estado histórico conhecido de uma porta no início de um determinado dia.

## 4.7. Construção do target

### 4.7.1. Definição inicial

Inicialmente, considerou-se qualquer ZM01 futura em nível de porta como evento positivo.

Essa definição gerou aproximadamente:

```text
7,74% de positivos
```

Entretanto, verificou-se que nem toda ZM01 representava uma falha real.

### 4.7.2. Refinamento do target

A variável:

```text
falha_flag_emergencial
```

foi utilizada para separar:

- ordens emergenciais classificadas como falha;
- ordens emergenciais sem falha real.

O target final foi então definido como:

```text
falha_emergencial_7d = 1
```

se existir, para a mesma porta física:

- evento ZM01;
- em nível de porta;
- `falha_flag_emergencial == True`;
- iniciado nos próximos 7 dias.

Caso contrário:

```text
falha_emergencial_7d = 0
```

A distribuição final foi:

| Classe | Quantidade | Percentual |
|---|---:|---:|
| 0 | 522.753 | 97,13% |
| 1 | 15.449 | 2,87% |

O problema, portanto, é fortemente desbalanceado.

## 4.8. Prevenção de data leakage

Uma regra foi aplicada em toda a engenharia de features:

> **Uma observação do dia D só pode utilizar dados conhecidos antes do início do dia D.**

Por isso, features históricas foram construídas com `shift(1)` antes das janelas móveis.

Exemplo:

```python
x.shift(1).rolling(7).sum()
```

Assim, para uma previsão em 10/05/2025:

```text
03/05 a 09/05 -> features históricas
10/05          -> momento da previsão
10/05 a 16/05 -> janela do target
```

Alarmes ocorridos no próprio dia da previsão não entram nas features daquela linha.

## 4.9. Período de aquecimento

Como foram utilizadas janelas de até 30 dias, os primeiros 30 dias foram reservados para criação de histórico.

O período efetivo utilizado no modelo passou a iniciar em:

```text
29/03/2023
```

Isso evitou comparar portas com histórico completo a portas sem janela histórica suficiente.

## 4.10. Features de alarmes

Foram criadas contagens históricas de eventos únicos.

### Volume total

```text
qtd_alarmes_1d
qtd_alarmes_3d
qtd_alarmes_7d
qtd_alarmes_30d
```

Interpretação:

- `qtd_alarmes_1d`: quantidade de eventos no dia anterior;
- `qtd_alarmes_3d`: quantidade nos 3 dias anteriores;
- `qtd_alarmes_7d`: quantidade nos 7 dias anteriores;
- `qtd_alarmes_30d`: quantidade nos 30 dias anteriores.

## 4.11. Features por tipo de alarme

Foram criadas contagens por tipo nos 7 dias anteriores.

```text
qtd_ca_7d
qtd_csd_7d
qtd_fc_7d
qtd_fo_7d
qtd_lsd_7d
qtd_oa_7d
qtd_obs_7d
qtd_rce_7d
qtd_sts_7d
```

Essas features permitem que o modelo diferencie não apenas o volume, mas também a natureza dos sinais apresentados pela porta.

## 4.12. Features de comportamento dos alarmes

Foram criadas features adicionais para representar persistência e diversidade.

### Dias com alarme

```text
dias_com_alarme_7d
```

Indica em quantos dos 7 dias anteriores houve pelo menos um alarme.

### Diversidade

```text
tipos_alarme_7d
```

Indica quantos tipos diferentes de alarmes apareceram nos 7 dias anteriores.

## 4.13. Features de histórico de falhas

A partir das falhas reais ZM01 anteriores foram criadas:

```text
qtd_falhas_30d
qtd_falhas_90d
qtd_falhas_anteriores
sem_falha_anterior
```

### `qtd_falhas_30d`

Quantidade de falhas reais da porta nos 30 dias anteriores.

### `qtd_falhas_90d`

Quantidade de falhas reais da porta nos 90 dias anteriores.

### `qtd_falhas_anteriores`

Quantidade acumulada de falhas reais anteriores à data da observação.

### `sem_falha_anterior`

```text
1 = porta nunca havia falhado anteriormente
0 = porta já possuía histórico de falha
```

## 4.14. Features exploradas posteriormente

Após observar desempenho limitado do primeiro modelo, foi realizada uma investigação adicional de features.

Foram avaliadas:

- contagens de tipos de alarme em 30 dias;
- diversidade de alarmes em 30 dias;
- dias com alarme em 30 dias;
- histórico de ZM02;
- recência de ZM02;
- quantidade de ZM02 em 30 e 90 dias.

## 4.15. Análise de correlação

Foi calculada a correlação linear das features com o target.

As correlações observadas foram baixas, em geral menores que 0,05.

Isso indicou que:

> nenhuma variável isolada apresentava forte relação linear com a ocorrência de falha futura.

Entretanto, esse resultado não implica que as features sejam inúteis para modelos baseados em árvores, já que relações não lineares e interações entre variáveis podem ser relevantes.

## 4.16. Informação mútua

Para investigar relações não lineares foi aplicada a métrica de informação mútua.

Os maiores destaques foram:

```text
sem_programada_anterior
tipos_alarme_30d
sem_falha_anterior
qtd_falhas_anteriores
qtd_falhas_90d
```

A variável:

```text
sem_programada_anterior
```

apresentou destaque considerável em relação às demais.

## 4.17. Investigação de `sem_programada_anterior`

Essa feature indica:

```text
1 = a porta ainda não havia registrado ZM02
0 = a porta já possuía pelo menos uma ZM02 anterior
```

A taxa de falha foi analisada por ano.

### 2023

| ZM02 anterior | Taxa de falha |
|---|---:|
| Já teve ZM02 | 4,68% |
| Nunca teve ZM02 | 3,65% |

### 2024

| ZM02 anterior | Taxa de falha |
|---|---:|
| Já teve ZM02 | 2,29% |
| Nunca teve ZM02 | 1,86% |

### 2025

| ZM02 anterior | Taxa de falha |
|---|---:|
| Já teve ZM02 | 3,05% |
| Nunca teve ZM02 | 2,46% |

O padrão se repetiu nos três anos.

Assim, essa variável foi considerada candidata plausível para o modelo.

## 4.18. Versões do conjunto de features

### V1 — Conjunto original

Features:

```text
estacao
plataforma

qtd_alarmes_1d
qtd_alarmes_3d
qtd_alarmes_7d
qtd_alarmes_30d

qtd_ca_7d
qtd_csd_7d
qtd_fc_7d
qtd_fo_7d
qtd_lsd_7d
qtd_oa_7d
qtd_obs_7d
qtd_rce_7d
qtd_sts_7d

dias_com_alarme_7d
tipos_alarme_7d

qtd_falhas_30d
qtd_falhas_90d
qtd_falhas_anteriores
sem_falha_anterior
```

Total conceitual:

```text
21 features
```

### V2 — Inclusão ampla de novas features

Foram adicionadas várias variáveis de:

- ZM02;
- alarmes de 30 dias;
- diversidade de alarmes;
- manutenção programada.

Resultado:

| Métrica | V1 | V2 |
|---|---:|---:|
| Accuracy | 0,6217 | 0,6231 |
| Precision | 0,0319 | 0,0304 |
| Recall | 0,5225 | 0,4941 |
| F1 | 0,0601 | 0,0573 |

A V2 não apresentou melhoria.

### V3 — Inclusão seletiva

Com base na informação mútua, apenas duas features adicionais foram mantidas:

```text
sem_programada_anterior
tipos_alarme_30d
```

Resultado:

| Métrica | V1 | V3 |
|---|---:|---:|
| Accuracy | 0,6217 | 0,5904 |
| Precision | 0,0319 | 0,0326 |
| Recall | 0,5225 | 0,5824 |
| F1 | 0,0601 | 0,0618 |

A V3 apresentou:

- maior Recall;
- maior Precision;
- maior F1;
- menor Accuracy.

Como Recall foi definido como métrica prioritária, a V3 foi considerada a melhor configuração de features entre as versões testadas.

---

# 5. CRISP-DM — Etapa 4: Modelagem

## 5.1. Divisão temporal dos dados

Devido à natureza temporal do problema, não foi utilizado split aleatório.

Foi adotado:

### Treino

```text
29/03/2023 a 25/03/2025
```

```text
377.104 observações
10.230 positivos
2,71% de positivos
```

### Gap temporal

Foi aplicada uma janela de separação para impedir que o target do treino avançasse sobre o período de validação.

### Validação

```text
01/04/2025 a 24/06/2025
```

```text
44.030 observações
1.020 positivos
2,32% de positivos
```

### Teste final

```text
01/07/2025 a 31/12/2025
```

```text
95.312 observações
3.343 positivos
3,51% de positivos
```

O teste final foi mantido separado das decisões de modelagem.

## 5.2. Escolha da ferramenta AutoML

Inicialmente foi considerada a utilização do PyCaret.

A instalação apresentou incompatibilidades com o ambiente Python 3.13 e dependências antigas de pandas/numpy.

Foi então escolhido o:

```text
FLAML
```

O FLAML foi utilizado como ferramenta AutoML para comparação automática entre diferentes algoritmos e configurações.

Entre os modelos avaliados estavam:

- Extra Trees;
- Random Forest;
- LightGBM;
- XGBoost;
- SGD;
- regressões lineares/logísticas.

## 5.3. Primeiro experimento AutoML

No experimento inicial, sem tratamento específico do desbalanceamento, o FLAML selecionou:

```text
Extra Trees
```

Configuração encontrada inicialmente:

```text
n_estimators = 4
max_leaves = 52
max_features ≈ 0,6591
criterion = gini
```

Na validação, o modelo apresentou aproximadamente:

```text
Accuracy  = 97,61%
Precision = 20,69%
Recall    = 1,18%
F1        = 2,23%
```

Matriz de confusão:

```text
[[42964,    46],
 [ 1008,    12]]
```

Embora a Accuracy fosse alta, o modelo detectava apenas 12 de 1.020 falhas.

Esse resultado confirmou que a Accuracy era inadequada como principal critério no contexto desbalanceado.

## 5.4. Ajuste de threshold no modelo não balanceado

Foram avaliados diversos thresholds.

O comportamento observado mostrou que, para obter Recall alto, o modelo precisava classificar grande parte da base como positiva.

Exemplo:

```text
threshold = 0,02
Recall ≈ 84,51%
Precision ≈ 2,51%
34.407 previsões positivas em 44.030 observações
```

Isso indicou baixa capacidade de separação entre as classes.

## 5.5. Segundo experimento AutoML com balanceamento

Foi aplicado balanceamento por peso de amostras no treino.

O FLAML novamente selecionou Extra Trees.

Configuração:

```text
n_estimators = 24
max_leaf_nodes = 4
max_features ≈ 0,1214
criterion = entropy
```

Resultados de validação:

```text
Accuracy  = 82,84%
Precision = 4,57%
Recall    = 32,25%
F1        = 8,01%
```

Matriz de confusão:

```text
[[36147, 6863],
 [  691,  329]]
```

O Recall aumentou significativamente em comparação ao primeiro experimento.

## 5.6. Foco no Extra Trees

Após os experimentos AutoML, foi decidido aprofundar a análise no algoritmo:

```text
ExtraTreesClassifier
```

Foi treinada uma versão mais robusta:

```python
ExtraTreesClassifier(
    n_estimators=300,
    criterion="entropy",
    max_features=0.15,
    max_leaf_nodes=8,
    class_weight="balanced",
    random_state=42,
    n_jobs=-1
)
```

As variáveis categóricas:

```text
estacao
plataforma
```

foram convertidas para one-hot encoding antes do treino.

## 5.7. Extra Trees com `class_weight="balanced"`

Resultados:

```text
Accuracy  = 32,26%
Precision = 2,71%
Recall    = 80,88%
F1        = 5,24%
```

Matriz:

```text
[[13381, 29629],
 [  195,   825]]
```

O Recall foi elevado, porém o modelo produziu muitos falsos positivos.

## 5.8. Ajuste manual do peso da classe positiva

Em vez de utilizar o balanceamento automático, foram testados diferentes pesos para a classe positiva:

```text
class_weight = {0: 1, 1: peso}
```

Pesos testados inicialmente:

```text
1, 2, 4, 6, 8, 10, 12, 16, 20, 25, 30
```

A região mais relevante apareceu acima de 30.

## 5.9. Busca fina de peso

Foram avaliados:

```text
30, 31, 32, 33, 34, 35, 36
```

Resultados:

| Peso | Accuracy | Precision | Recall | F1 |
|---:|---:|---:|---:|---:|
| 30 | 0,9243 | 0,0578 | 0,1480 | 0,0831 |
| 31 | 0,8946 | 0,0477 | 0,1873 | 0,0761 |
| 32 | 0,8380 | 0,0406 | 0,2647 | 0,0704 |
| 33 | 0,7339 | 0,0343 | 0,3863 | 0,0630 |
| 34 | 0,6217 | 0,0319 | 0,5225 | 0,0601 |
| 35 | 0,4568 | 0,0307 | 0,7353 | 0,0590 |
| 36 | 0,3135 | 0,0267 | 0,8088 | 0,0518 |

O peso 34 foi utilizado como ponto de equilíbrio inicial.

## 5.10. Ajuste de threshold com peso 34

Mantendo:

```text
class_weight = {0: 1, 1: 34}
```

foram testados thresholds entre 0,40 e 0,60.

Resultados principais:

| Threshold | Accuracy | Precision | Recall | F1 |
|---:|---:|---:|---:|---:|
| 0,40 | 0,1212 | 0,0250 | 0,9716 | 0,0487 |
| 0,44 | 0,1690 | 0,0259 | 0,9539 | 0,0505 |
| 0,48 | 0,3083 | 0,0266 | 0,8108 | 0,0515 |
| 0,50 | 0,6217 | 0,0319 | 0,5225 | 0,0601 |
| 0,52 | 0,8763 | 0,0424 | 0,2010 | 0,0700 |
| 0,54 | 0,9443 | 0,0608 | 0,0971 | 0,0747 |
| 0,56 | 0,9667 | 0,0884 | 0,0471 | 0,0614 |

O threshold 0,50 foi utilizado durante a comparação entre versões de features.

---

# 6. CRISP-DM — Etapa 5: Avaliação

## 6.1. Comparação das versões

A comparação final na validação ficou:

| Versão | Accuracy | Precision | Recall | F1 |
|---|---:|---:|---:|---:|
| V1 | 0,6217 | 0,0319 | 0,5225 | 0,0601 |
| V2 | 0,6231 | 0,0304 | 0,4941 | 0,0573 |
| V3 | 0,5904 | 0,0326 | 0,5824 | 0,0618 |

A V3 apresentou o melhor resultado considerando a prioridade de negócio.

## 6.2. Matriz de confusão da V3

Na validação:

```text
[[25401, 17609],
 [  426,   594]]
```

Interpretando:

```text
TN = 25.401
FP = 17.609
FN =    426
TP =    594
```

Das 1.020 falhas reais presentes na validação, o modelo detectou 594 e deixou de detectar 426.

O Recall obtido foi:

```text
58,24%
```

## 6.3. Interpretação do resultado

O modelo ainda apresenta Precision baixa.

Isso significa que muitos alertas positivos não correspondem a falhas reais.

Entretanto, considerando a natureza desbalanceada do problema e a prioridade de reduzir falsos negativos, a V3 apresentou uma melhoria em relação ao conjunto original de features.

O principal ganho foi aumentar a quantidade de falhas antecipadas.

## 6.4. Configuração escolhida ao final da discussão

A configuração escolhida para seguir para avaliação final foi:

```text
Algoritmo:
Extra Trees Classifier

n_estimators:
300

criterion:
entropy

max_features:
0.15

max_leaf_nodes:
8

class_weight:
classe 0 = 1
classe 1 = 34

random_state:
42

threshold utilizado durante comparação:
0.50
```

Conjunto de features:

```text
estacao
plataforma

qtd_alarmes_1d
qtd_alarmes_3d
qtd_alarmes_7d
qtd_alarmes_30d

qtd_ca_7d
qtd_csd_7d
qtd_fc_7d
qtd_fo_7d
qtd_lsd_7d
qtd_oa_7d
qtd_obs_7d
qtd_rce_7d
qtd_sts_7d

dias_com_alarme_7d
tipos_alarme_7d

qtd_falhas_30d
qtd_falhas_90d
qtd_falhas_anteriores
sem_falha_anterior

sem_programada_anterior
tipos_alarme_30d
```

---

# 7. Decisões metodológicas principais

## 7.1. O target representa falha, não alarme

O target é:

```text
falha_emergencial_7d
```

e não a ocorrência de um alarme futuro.

Alarmes são sinais antecedentes.

## 7.2. Somente falhas ZM01 reais entram no target

Uma ZM01 somente gera target positivo quando:

```text
falha_flag_emergencial == True
```

## 7.3. Somente eventos em nível de porta entram no target

Ordens em nível de estação ou plataforma não são arbitrariamente atribuídas a portas.

## 7.4. O split é temporal

Não foi utilizado embaralhamento aleatório.

Isso aproxima o experimento de uma situação real:

```text
treinar no passado
validar no futuro
testar em um período ainda posterior
```

## 7.5. Foi utilizado gap temporal

O gap reduz o risco de sobreposição entre o horizonte futuro do target do treino e a janela seguinte.

## 7.6. Recall é prioritário

O objetivo não é maximizar Accuracy.

O modelo deve reduzir a quantidade de falhas reais que passam sem sinalização.

## 7.7. Features futuras foram excluídas

Não foram utilizadas como entrada:

- falha futura;
- causa da falha futura;
- defeito futuro;
- ação futura;
- duração futura da manutenção;
- horário de encerramento futuro;
- `dias_ate_falha`;
- próxima ZM01;
- informações que só seriam conhecidas após a data de previsão.

---

# 8. Limitações identificadas

## 8.1. Forte desbalanceamento

A classe positiva representa aproximadamente:

```text
2,87%
```

da base completa.

Isso torna a classificação significativamente mais difícil.

## 8.2. Precision baixa

Mesmo com Recall mais alto, ainda há muitos falsos positivos.

Esse ponto deve ser considerado caso o modelo seja levado para uso operacional.

## 8.3. Relações individuais fracas

As features apresentam baixa correlação linear individual com o target.

O modelo depende de relações não lineares e combinações entre sinais.

## 8.4. Alarmes não representam diretamente falhas

Um alto volume de alarmes pode ocorrer sem falha futura.

Assim, contagens de alarmes isoladamente não são suficientes.

## 8.5. ZM03 não possui granularidade de porta

A manutenção preventiva não foi incorporada diretamente ao primeiro modelo por não existir granularidade de porta suficiente.

## 8.6. Fluxo de passageiros ainda não foi utilizado

A base de fluxo foi estruturada, mas não incorporada ao primeiro modelo.

Ela pode ser explorada em uma iteração futura.

---

# 9. Possíveis trabalhos futuros

A partir do primeiro modelo, algumas extensões possíveis são:

1. testar novas famílias de features operacionais;
2. incorporar fluxo de passageiros;
3. testar preventiva em nível de plataforma;
4. explorar tempo desde último alarme;
5. explorar tempo desde última falha;
6. usar permutation importance;
7. aplicar SHAP;
8. testar calibração de probabilidades;
9. utilizar Precision-Recall Curve;
10. selecionar threshold com base em custo operacional;
11. comparar Extra Trees com LightGBM e XGBoost;
12. realizar validação temporal com múltiplas janelas;
13. investigar previsão em outros horizontes, como 3, 14 ou 30 dias;
14. avaliar desempenho por estação;
15. avaliar desempenho por plataforma;
16. avaliar desempenho por tipo de porta;
17. investigar falhas reincidentes separadamente de primeiras falhas.

---

# 10. Estado atual do projeto

Até este ponto foram concluídas:

```text
CRISP-DM 1 — Entendimento do negócio
CRISP-DM 2 — Entendimento dos dados
CRISP-DM 3 — Preparação dos dados
CRISP-DM 4 — Modelagem
CRISP-DM 5 — Avaliação em conjunto de validação
```

A discussão de engenharia de features foi encerrada com a seleção da versão V3.

O conjunto de teste final foi mantido separado das decisões de:

- seleção de features;
- seleção de modelo;
- escolha de peso da classe;
- análise de threshold.

Assim, o próximo passo natural é realizar a avaliação final no período de teste previamente reservado.

---

# 11. Conclusão

O primeiro modelo foi estruturado para prever falhas emergenciais reais de portas PSD com horizonte de 7 dias.

O processo mostrou que:

- o problema é fortemente desbalanceado;
- Accuracy isoladamente produz uma visão enganosa;
- o Recall precisa ser priorizado;
- alarmes históricos possuem algum sinal, mas não determinam a falha de forma simples;
- relações não lineares são relevantes;
- o Extra Trees apresentou melhor comportamento quando a classe positiva recebeu maior peso;
- a seleção criteriosa de features foi mais eficaz do que simplesmente adicionar grande quantidade de variáveis.

A versão V3 apresentou, na validação:

```text
Accuracy  = 59,04%
Precision = 3,26%
Recall    = 58,24%
F1        = 6,18%
```

Esses números não representam ainda o desempenho final definitivo do modelo em produção. Eles representam o melhor resultado obtido durante a etapa de validação e seleção do primeiro modelo.

O teste temporal final deve ser utilizado apenas após o congelamento das decisões metodológicas, permitindo uma avaliação mais honesta da capacidade de generalização.

---

## Resumo executivo

```text
Problema:
Classificação binária

Pergunta:
Essa porta apresentará uma falha real nos próximos 7 dias?

Target:
falha_emergencial_7d

Unidade:
1 porta × 1 dia

Horizonte:
7 dias

Modelo:
Extra Trees Classifier

Métrica principal:
Recall

Melhor versão de features:
V3

Recall de validação:
58,24%

Principais fontes:
Alarmes PSD
ZM01
ZM02

Validação:
Temporal

Prevenção de leakage:
Features usam somente informações anteriores à data da previsão
```
