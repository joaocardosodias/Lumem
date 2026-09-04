# Resumo da Sprint 2 — Manutenção Preditiva de Portas de Plataforma (PSDs)

**Projeto:** Manutenção Preditiva de PSDs — Linha 4-Amarela do Metrô de São Paulo
**Parceiro:** Motiva
**Grupo:** 04 — Subway Surfers · Inteli, Módulo 7 (2026-2A / T16)
**Versão:** 0.2.0 — encerramento em 27/08/2026

---

## 1. Objetivo da sprint

A Sprint 1 entregou o entendimento do negócio, as personas (Ronaldo e Marta), a validação estatística de 7 hipóteses de falha e os pipelines de limpeza das bases. A Sprint 2 partiu desse alicerce com um objetivo único: **construir, treinar, avaliar e comparar modelos de Machine Learning capazes de antecipar ordens de manutenção corretiva emergencial (ZM01) nas portas de plataforma**.

A pergunta de modelagem, compartilhada por todos os experimentos, é:

> Com base apenas no histórico conhecido até o final do dia `D`, este ativo apresentará uma falha confirmada entre `D+1` e `D+7`?

O horizonte de 7 dias foi escolhido por ser compatível com o planejamento semanal da manutenção e com as janelas noturnas de intervenção (4 horas).

---

## 2. Entregas da sprint

| Entrega | Descrição | Localização |
| :--- | :--- | :--- |
| Tabela Mestra Contínua | *Feature store* com calendário completo por ativo, garantindo dias sem eventos com contagem zero | `data/`, `data/processados/` |
| Engenharia de features | Entre 10 e 83 atributos candidatos por experimento, derivados das hipóteses da Sprint 1 | notebooks em `models/` |
| Blindagem temporal | Lags causais (`shift(1)`), *merge asof* sem correspondência exata, purga entre splits, escalonamento ajustado só no treino | notebooks em `models/` |
| 7 modelos preditivos | Decision Tree, Random Forest, XGBoost, LightGBM, CatBoost, Extra Trees e Ensembles | `models/modelo-1` a `models/modelo-7` |
| Comparação e seleção | Análise crítica das métricas e escolha do modelo campeão | `docs/.../Sprint 2/comparacao_modelos.mdx` |
| Wireframe da plataforma | 9 telas de baixa fidelidade cobrindo 5 fluxos + login | `docs/.../Sprint 2/wireframe.mdx` |
| Documentação | Portal Astro/Starlight com página por modelo, comparação e visão geral | `docs/src/content/docs/sprints/Sprint 2/` |
| Artefatos serializados | Modelos treinados, *scalers* e configurações | `data/modelos_artefatos/`, `models/modelo-7/modelo_catboost.cbm` |

---

## 3. Bases de dados utilizadas

Quatro fontes operacionais foram consolidadas. O período comum às quatro foi delimitado entre **27/02/2023 e 28/03/2026**, totalizando 1.126 dias.

| Fonte | Papel | Dimensão original | Retenção após filtro |
| :--- | :--- | ---: | ---: |
| Histórico consolidado de alarmes | Comportamento diário do sistema PSD | 742.506 × 15 | 741.138 (99,82%) |
| ZM01 — corretiva emergencial | Eventos corretivos e falhas confirmadas | 10.543 × 27 | 10.200 (96,75%) |
| ZM02 — corretiva programada | Intervenções programadas | 1.163 × 27 | 1.085 (93,29%) |
| ZM03 — preventiva | Manutenção preventiva | 569 × 19 | 530 (93,15%) |

A base de catracas (movimentação de passageiros) foi mantida disponível, mas descartada da maioria dos modelos porque a hipótese H3 — de que o fluxo de passageiros explicaria as falhas — foi refutada na Sprint 1.

### Como as hipóteses da Sprint 1 orientaram as features

| Hipótese | Resultado na Sprint 1 | Uso na Sprint 2 |
| :--- | :--- | :--- |
| H1 — padrões temporais dos alarmes | Sustentada | Duração de alarmes, alarmes críticos e janelas móveis de 7 dias |
| H2 — regime semanal e horários de pico | Sustentada | Dia da semana, flag de domingo, alarmes nos picos manhã (06–09h) e tarde (16–19h) |
| H3 — movimentação de passageiros | Refutada | Catracas excluídas (exceto no Modelo 2, como teste) |
| H4 — aumento de alarmes antes da corretiva | Não confirmada | Contagem total de alarmes descartada; só sinais críticos mantidos |
| H5 — tempo de normalização por estação | Sinal presente, insuficiente isolado | Duração de alarmes + identificador de estação e PSD |
| H6 — posição central vs. extremidade da porta | Efeito prático pequeno | Não priorizada nos modelos agregados |
| H7 — recorrência de falhas mecânicas | Sustentada | Preservação do histórico de manutenção; detalhamento de componente adiado |

---

## 4. Os sete modelos desenvolvidos

Cada integrante conduziu um experimento independente, com granularidade e estratégia próprias. Essa diversidade foi intencional: permitiu testar em paralelo se o problema é melhor atacado no nível da porta individual, da plataforma ou da estação.

### Modelo 1 — XGBoost & Ensemble (proteção máxima)

Granularidade **porta × dia** sobre as 526 PSDs, com tabela mestra de 607.530 linhas e ~27 features. Foi o modelo mais agressivo em recall: com limiar em 0,45, evita **380 de 388 falhas reais** (97,94% de recall na régua estrita por episódio), ao custo de precisão de 9,35%. O melhor F1 contínuo (0,1829) aparece no limiar 0,52, com recall de 29,90%.

Técnicas notáveis: `scale_pos_weight = 9.5`, regularização L1/L2, calibração via Optuna, lags causais com `shift(1)`, *merge asof* com `allow_exact_matches=False` e explicabilidade global com SHAP Beeswarm. As features mais influentes foram `dias_desde_ultima_zm01`, `fator_risco_ciclo_zm01` e `qtd_anomalias_30s_14d`.

### Modelo 2 — LightGBM tunado + Random Forest baseline (equilíbrio)

Granularidade **porta × dia**, 296.472 linhas e 15,42% de positivos. Partiu de 23 features e chegou a 16 após remover as 7 com importância por permutação ≤ 0 — sem perda de AUC. Split temporal 80/20 e *tuning* com validação *walk-forward* de 4 folds (AUC médio 0,6183 ± 0,0293).

Resultado final no teste (limiar 0,5): ROC-AUC 0,6066, recall 60,50%, precisão 20,60%, F1 0,3080, acurácia 55,00%. A contribuição metodológica mais relevante foi a **comparação sistemática entre importância por Gini e por permutação**, que expôs features que pareciam úteis no treino mas não generalizavam.

### Modelo 3 — Decision Tree sob CRISP-DM (interpretabilidade)

Granularidade **PSD × domingo**, com previsão semanal para as 526 PSDs. Partiu de 47 features candidatas e selecionou 13 via árvore auxiliar. Aplicou split temporal cronológico **com purga de 7 dias** e manteve o teste congelado até a avaliação final.

No teste (limiar 0,2088, escolhido por F1 máximo na validação): recall 31,63%, precisão 12,42%, F1 17,84%, ROC-AUC 57,13%, PR-AUC 10,65%, acurácia 76,89%. Por evento, detectou 293 de 888 falhas (33%) com antecedência média de 3,6 dias e ~77,9 alertas por semana. A feature dominante foi `dias_desde_ultima_corretiva_30d` (38,62% de importância).

O próprio documento classifica o modelo como **prova de conceito**, não aprovado para uso autônomo — em parte porque a cobertura de PSDs cai de 478 para 286 em 2026.

### Modelo 4 — Extra Trees via FLAML AutoML

Granularidade **porta × dia** sobre 518 portas, ~538 mil linhas e apenas 2,87% de positivos. Usou AutoML (FLAML) para selecionar a arquitetura e 23 features na versão final, com `class_weight={0:1, 1:34}` para compensar o desbalanceamento severo.

Na validação (limiar 0,50): recall 58,24%, precisão 3,26%, F1 6,18%, acurácia 59,04%. **O conjunto de teste foi reservado e ainda não avaliado**, o que limita a comparabilidade com os demais. A precisão muito baixa reflete a raridade extrema da classe positiva nessa granularidade.

### Modelo 5 — Random Forest por evento e ciclos (sensibilidade a recorrência)

Granularidade **porta × dia** sobre 480 portas em 10 estações (~307 mil linhas). Partiu de 83 features candidatas e chegou a 71 após seleção por contribuição SHAP. Split temporal com **7 dias de folga** para neutralizar o vazamento causado por médias móveis.

Sua principal inovação foi avaliar o modelo **por episódio, não por linha**: precisão 26,0% (496 de 1.905 episódios) e recall 65,0% (766 de 1.178 falhas), com F1 por evento de 0,372. O teste de ablação mostrou o valor das features de ciclo — sem elas, o F1 por evento cai para 0,325. Registrou o **melhor ROC-AUC da comparação (0,6970)**.

### Modelo 6 — XGBoost por estação × evento

Granularidade **estação × evento de manutenção**, com 11.504 exemplos e 10 sinais preditivos. *Tuning* por `RandomizedSearchCV` (25 combinações) otimizando acurácia balanceada.

As métricas nominais são as mais altas de todos os modelos (precisão 0,99, recall 0,95, F1 0,97 na classe positiva), mas **esses números são enganosos**: a classe 1 representa ~98,85% do conjunto de teste (2.312 contra 27 observações). A acurácia de 93,93% é *inferior* ao baseline trivial de sempre prever a classe majoritária (98,85%), e a acurácia balanceada real é de apenas 58,49%. Na classe minoritária, precisão 0,05 e recall 0,22.

### Modelo 7 — CatBoostClassifier por plataforma × dia (modelo campeão)

Granularidade **estação × plataforma × dia**, cobrindo 22 sistemas PSD (11 estações × 2 plataformas) ao longo de 1.126 dias — 24.772 linhas na tabela mestra, 0 duplicatas e 0 nulos. Importante: "22 PSDs" significa 22 *sistemas de plataforma*, não 22 portas individuais.

Recebeu **21 features** (4 categóricas + 17 numéricas), com `qtd_falhas_confirmadas`, `qtd_falhas_proximos_7d` e `target_falha_7d` explicitamente bloqueadas como entrada. A divisão temporal usou **purga de 7 dias** entre conjuntos:

| Conjunto | Período | Linhas | Positivos | Taxa positiva |
| :--- | :--- | ---: | ---: | ---: |
| Treino | 28/03/2023 a 24/03/2025 | 16.016 | 7.264 | 45,35% |
| Validação | 01/04/2025 a 23/09/2025 | 3.872 | 2.019 | 52,14% |
| Teste | 01/10/2025 a 21/03/2026 | 3.784 | 1.957 | 51,72% |

Cinco arquiteturas foram comparadas na validação. A vencedora, **"rasa regularizada"** (`depth=4`, `learning_rate=0,03`, `l2_leaf_reg=5`, melhor iteração 308), atingiu PR-AUC 0,6046 e recall 91,63%. O limiar foi definido **exclusivamente na validação** em **0,317380**, buscando a maior precisão entre os pontos com recall ≥ 80%.

Resultado no teste, aplicado uma única vez:

| Métrica | Resultado |
| :--- | ---: |
| Recall | **91,93%** |
| Precisão | 54,63% |
| F1 | 68,53% |
| PR-AUC | 0,5874 |
| ROC-AUC | 0,5911 |
| Acurácia | 56,34% |
| Proporção de alertas | 87,02% |

Matriz de confusão: 1.799 verdadeiros positivos, 158 falsos negativos, 1.494 falsos positivos, 333 verdadeiros negativos. As features mais importantes foram `id_psd` (25,51%), `dias_desde_ultima_preventiva` (13,25%), `estacao_padrao` (10,62%) e `qtd_corretivas_emergenciais_30d` (9,77%).

---

## 5. Comparação e seleção do modelo

A comparação seguiu uma regra metodológica estrita: **nenhum modelo foi retreinado, recalibrado ou reavaliado**. Cada um foi comparado exatamente como entregue, e métricas ausentes permaneceram ausentes em vez de serem recalculadas — evitando que alguns modelos fossem julgados por critérios diferentes dos outros.

| Modelo | Accuracy | Precision | Recall | F1 | ROC-AUC | PR-AUC |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| Modelo 1 | N/A | 0,1047 | 0,5361 | 0,1751 | 0,5235 | 0,0913 |
| Modelo 2 | 0,5500 | 0,2060 | 0,6050 | 0,3080 | 0,6066 | N/A |
| Modelo 3 | 0,7689 | 0,1242 | 0,3163 | 0,1784 | 0,5713 | 0,1065 |
| Modelo 4 | 0,5904 | 0,0326 | 0,5824 | 0,0618 | 0,6075 | 0,0407 |
| Modelo 5 | N/A | 0,1590 | 0,5530 | 0,2470 | **0,6970** | 0,1850 |
| Modelo 6 | **0,9372** | **0,9900** | **0,9500** | **0,9700** | N/A | N/A |
| Modelo 7 | 0,5634 | 0,5463 | 0,9193 | 0,6853 | 0,5911 | **0,5874** |

### A ressalva sobre o Modelo 6

O Modelo 6 parecia o vencedor absoluto até a distribuição de classes ser examinada. Como ~98,85% do seu conjunto de teste pertence à classe positiva, as métricas dessa classe são naturalmente infladas. O teste decisivo: um classificador trivial que sempre previsse a classe majoritária atingiria 98,85% de acurácia, enquanto o modelo obteve 93,72% — ou seja, **desempenho inferior ao baseline mais simples possível**. Na classe minoritária, precisão 0,05, recall 0,22 e F1 0,08; a acurácia balanceada ficou em ~58,4%. O modelo foi mantido na comparação, mas suas métricas foram interpretadas com cautela.

### Por que o Modelo 7 foi escolhido

O Modelo 5 foi o principal concorrente, liderando em ROC-AUC (0,6970 contra 0,5911). A decisão pelo Modelo 7 se apoiou em quatro pontos:

1. **Recall de 91,93%** — o mais alto entre os modelos sem distorção de classe, alinhado à prioridade operacional de não perder falhas.
2. **Precisão de 54,63% mantida junto ao recall alto** — recall elevado é fácil de obter alertando quase tudo; manter precisão acima de 50% mostra que não é o caso.
3. **F1 de 0,6853** — o maior entre os modelos comparáveis, indicando equilíbrio real entre as duas métricas.
4. **PR-AUC de 0,5874** — o argumento mais forte. O segundo colocado (Modelo 5) ficou em 0,1850, uma diferença de mais de 3×. Isso mostra que o bom desempenho não depende de um limiar específico afortunado, mas se sustenta ao longo de diferentes limiares de decisão.

> **Modelo selecionado: Modelo 7 — CatBoostClassifier.** A escolha não veio do maior valor isolado de uma métrica, mas da combinação entre desempenho, consistência entre métricas e interpretação crítica da distribuição de classes de cada experimento.

---

## 6. Wireframe da plataforma

Foi produzido o wireframe de baixa fidelidade da plataforma, pensado para a persona **Marta** (operadora/planejadora de manutenção) mapeada na Sprint 1. O propósito de um artefato de baixa fidelidade é validar **estrutura e fluxo**, não aparência — daí a ausência de cores de marca e imagens finais.

Todas as telas (exceto login) compartilham uma barra lateral fixa com logo, 5 abas de navegação e botão de sair.

| Tela | Conteúdo |
| :--- | :--- |
| Login | Autenticação por e-mail e senha; única tela sem barra lateral |
| Dashboard | Cards macro (ex.: portas que precisam de manutenção nos próximos 7 dias), filtros e tabela com *ranking* de portas priorizadas pelo risco previsto |
| Estações (lista) | Filtro + cards por estação, com acesso às portas de cada uma |
| Estações (detalhes) | Métricas de alarmes, histórico de alarmes e histórico de manutenções da porta |
| Estações (agendamento) | Pop-up sobre a tela de detalhes para agendar manutenção sem perder o contexto |
| Manutenções (lista) | Cards macro (agendadas, em atraso, realizadas), filtro e tabela |
| Manutenções (calendário) | Mesma informação distribuída ao longo dos dias, alternável pela lista |
| Histórico | Duas colunas lado a lado: histórico de alarmes à esquerda, de manutenções à direita, para comparação no mesmo período |
| Novos Dados | Ingestão via upload de planilha Excel ou inserção manual de registros |

---

## 7. Decisões técnicas transversais

Três práticas foram adotadas de forma consistente entre os experimentos e merecem destaque:

**Blindagem contra vazamento temporal.** Nenhum modelo usou embaralhamento aleatório. Todos os splits são cronológicos, com o teste representando o período mais recente. Os modelos 3, 5 e 7 aplicaram explicitamente uma **purga de 7 dias** entre conjuntos, impedindo que a janela do alvo de uma observação de treino alcance o período de validação ou teste. Features derivadas de eventos passados usam `shift(1)`, e o *merge asof* foi configurado com `allow_exact_matches=False`.

**Recall como métrica prioritária.** A premissa de negócio é que o custo de deixar uma falha futura sem alerta é maior que o custo de inspecionar um sistema que não falharia. Por isso os limiares foram deslocados para baixo de 0,5 sempre que necessário, e a acurácia foi tratada apenas como métrica complementar — ela não distingue os dois tipos de erro, que têm custos muito diferentes aqui.

**Limiar definido só na validação.** No Modelo 7, o limiar de 0,317380 foi escolhido na validação e o conjunto de teste permaneceu intocado até a decisão final estar tomada. Essa disciplina é o que dá credibilidade ao recall de 91,93% reportado.

---

## 8. Limitações reconhecidas

A documentação é explícita ao afirmar que **o modelo campeão ainda não está pronto para uso operacional irrestrito**:

- **Volume de alertas alto:** 87,02% das linhas de teste geram alerta, e 45,37% dos alertas são falsos. Não foi validado se a equipe de manutenção consegue atender esse volume diário.
- **Discriminação moderada:** ROC-AUC e PR-AUC próximos de 0,59 indicam capacidade de separação limitada. Nenhuma feature numérica isolada tem correlação superior a 0,18 com o alvo.
- **Granularidade agregada:** o Modelo 7 aponta a plataforma, não qual porta PDM específica inspecionar.
- **Contagem dupla no alvo:** uma mesma falha física pode tornar positivos até 7 dias anteriores, então "1.799 positivos detectados" não equivale a 1.799 falhas distintas.
- **Perda por cobertura de chaves:** no período comum, 209 ordens ZM01 (51 marcadas como falha), 56 ZM02 e 238 ZM03 ficaram fora das agregações por não terem plataforma identificável. Elas não foram replicadas artificialmente para não criar eventos inexistentes.
- **Deriva de prevalência:** a taxa positiva sobe de 45,35% no treino para ~52% em validação e teste, e a causa ainda não foi investigada.
- **Redundância entre features:** vários pares apresentam Spearman acima de 0,90 (ex.: duração média e máxima de alarmes, 0,9926), o que divide a importância entre variáveis e torna a interpretação individual menos estável.

---

## 9. Próximos passos (Sprint 3)

Do plano de aprimoramento registrado na documentação:

1. Testar as features de histórico de falha e janelas de 30/90 dias já preparadas, mas ainda não incorporadas ao modelo salvo.
2. Avaliar o modelo **por falha física única**, e não apenas por PSD-dia, seguindo a abordagem por episódio do Modelo 5.
3. Selecionar o limiar considerando um **limite operacional de alertas por dia**, transformando a decisão técnica em decisão de negócio.
4. Comparar CatBoost com LightGBM, XGBoost e regressão logística sob a mesma régua e no mesmo conjunto de dados.
5. Executar validação temporal em **múltiplas janelas**, não apenas um corte único.
6. Avaliar a calibração das probabilidades previstas.
7. Estudar modelagem por porta PDM quando a cobertura das chaves permitir.
8. Documentar com a área operacional o custo real de um falso negativo e de um falso positivo.
9. Investigar a mudança de prevalência entre treino e períodos posteriores.
10. Evoluir o wireframe para protótipos de média e alta fidelidade, incorporando identidade visual e testes de usabilidade.

---

## 10. Reprodutibilidade

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

Os pipelines estão centralizados em `models/`, um diretório por modelo. Para o Modelo 7, os notebooks devem ser executados nesta ordem:

| Ordem | Notebook | Saída principal |
| ---: | :--- | :--- |
| 1 | `models/modelo-7/01_modelo_7_construcao_tabela_mestra.ipynb` | `tabela_mestra_psd.csv` |
| 2 | `models/modelo-7/02_modelo_7_processamento.ipynb` | bases de treino, validação e teste + configuração |
| 3 | `models/modelo-7/03_modelo_7_treinamento.ipynb` | `modelo_catboost.cbm`, metadata, previsões e importâncias |

Para reproduzir a decisão operacional não basta carregar o `.cbm`: é necessário usar as mesmas 21 features com a mesma semântica, preservar `random_seed=42` e aplicar o limiar `0,317380` registrado em `metadata_modelo.json`.

A documentação completa está publicada em [graduacao.pages.git.inteli.edu.br/2026-2a/t16/g04](https://graduacao.pages.git.inteli.edu.br/2026-2a/t16/g04/) e o código-fonte do portal fica em `docs/`.
